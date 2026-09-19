// Package checker provides functionality for checking endpoints over HTTP,
// plain TCP, and TLS certificates.
package checker

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/caarlos0/env/v11"
)

// Check types, stored in the monitors table and derived from the URL scheme.
const (
	TypeHTTP = "http"
	TypeTCP  = "tcp"
	TypeSSL  = "ssl"

	// CertWarnDays is how many days before certificate expiry an ssl monitor
	// counts as degraded and subscribers get a warning.
	CertWarnDays = 30
)

// TypeForScheme maps a URL scheme to its check type, defaulting to http.
func TypeForScheme(scheme string) string {
	switch scheme {
	case "tcp":
		return TypeTCP
	case "ssl":
		return TypeSSL
	default:
		return TypeHTTP
	}
}

// validSchemes are the URL schemes a monitor may use.
var validSchemes = []string{"http", "https", "tcp", "ssl"}

// ValidScheme reports whether scheme is one a monitor may use.
func ValidScheme(scheme string) bool {
	return slices.Contains(validSchemes, scheme)
}

type Checker struct {
	client *http.Client
	dialer *net.Dialer

	Timeout  time.Duration `env:"BEACON_TIMEOUT"  envDefault:"30s"`
	Insecure bool          `env:"BEACON_INSECURE" envDefault:"false"`
}

type Result struct {
	StatusCode   int64  // HTTP status, zero for tcp and ssl checks
	ResponseTime int64  // in ms
	DaysLeft     *int64 // https and ssl: days until the certificate expires
	Error        *string
	IsUp         bool
}

func New() (*Checker, error) {
	c, err := env.ParseAs[Checker]()
	if err != nil {
		return nil, err
	}

	c.client = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig:   c.tlsConfig(),
			DisableKeepAlives: true,
		},
	}
	c.dialer = &net.Dialer{Timeout: c.Timeout}

	return &c, nil
}

func (c *Checker) tlsConfig() *tls.Config {
	return &tls.Config{InsecureSkipVerify: c.Insecure} // #nosec G402
}

// Check dispatches on the URL scheme: http and https do an HTTP GET, tcp
// measures a plain connection, ssl validates a TLS certificate.
func (c *Checker) Check(ctx context.Context, rawURL string) Result {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return failed("invalid url", err, 0)
	}

	switch parsed.Scheme {
	case "http", "https":
		return c.checkHTTP(ctx, rawURL)
	case "tcp":
		if parsed.Port() == "" {
			return failedMsg("connection failed: tcp url requires an explicit port, e.g. tcp://host:5432", 0)
		}
		return c.checkTCP(ctx, parsed.Host)
	case "ssl":
		addr := parsed.Host
		if parsed.Port() == "" {
			addr = net.JoinHostPort(parsed.Hostname(), "443")
		}
		return c.checkSSL(ctx, addr)
	default:
		return failedMsg(
			fmt.Sprintf("connection failed: unsupported scheme %q, use http, https, tcp or ssl", parsed.Scheme),
			0,
		)
	}
}

func (c *Checker) checkHTTP(ctx context.Context, url string) Result {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return failed("request failed", err, 0)
	}
	req.Header.Set("User-Agent", "Beacon/1.0")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Connection", "close")

	resp, err := c.client.Do(req)
	ms := time.Since(start).Milliseconds()
	if err != nil {
		return failed("request failed", err, ms)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	result := Result{
		IsUp:         resp.StatusCode >= 200 && resp.StatusCode < 400,
		StatusCode:   int64(resp.StatusCode),
		ResponseTime: ms,
	}
	// https responses carry the peer certificate from the handshake, so every
	// https monitor tracks expiry for free.
	if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		result.DaysLeft = certDaysLeft(resp.TLS.PeerCertificates[0])
	}
	return result
}

func (c *Checker) checkTCP(ctx context.Context, addr string) Result {
	start := time.Now()
	conn, err := c.dialer.DialContext(ctx, "tcp", addr)
	ms := time.Since(start).Milliseconds()
	if err != nil {
		return failed("connection failed", err, ms)
	}
	_ = conn.Close()

	return Result{IsUp: true, ResponseTime: ms}
}

func (c *Checker) checkSSL(ctx context.Context, addr string) Result {
	dialer := &tls.Dialer{
		NetDialer: c.dialer,
		Config:    c.tlsConfig(),
	}

	start := time.Now()
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	ms := time.Since(start).Milliseconds()
	if err != nil {
		if certErr, ok := errors.AsType[x509.CertificateInvalidError](err); ok && certErr.Reason == x509.Expired {
			return failed("certificate check failed", certErr, ms)
		}
		return failed("tls handshake failed", err, ms)
	}
	defer func() { _ = conn.Close() }()

	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		return failedMsg("certificate check failed: connection is not TLS", ms)
	}
	certs := tlsConn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return failedMsg("certificate check failed: server sent no certificate", ms)
	}

	days := certDaysLeft(certs[0])
	if days == nil {
		return failedMsg(
			fmt.Sprintf("certificate check failed: certificate expired on %s", certs[0].NotAfter.Format(time.DateOnly)),
			ms,
		)
	}

	return Result{IsUp: true, ResponseTime: ms, DaysLeft: days}
}

// certDaysLeft returns whole days until the certificate expires, nil when it
// is already expired.
func certDaysLeft(cert *x509.Certificate) *int64 {
	days := int64(time.Until(cert.NotAfter).Hours() / 24)
	if days < 0 {
		return nil
	}
	return new(days)
}

func failed(reason string, err error, responseTime int64) Result {
	msg := fmt.Sprintf("%s: %v", reason, err)
	return failedMsg(msg, responseTime)
}

func failedMsg(msg string, responseTime int64) Result {
	return Result{
		IsUp:         false,
		Error:        &msg,
		ResponseTime: responseTime,
	}
}
