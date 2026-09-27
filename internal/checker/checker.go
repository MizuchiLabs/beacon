// Package checker checks endpoints over HTTP, plain TCP, TLS, DNS and ICMP.
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
	"strings"
	"time"

	"github.com/caarlos0/env/v11"

	"github.com/mizuchilabs/beacon/internal/db"
)

// Check types, stored in the monitors table and derived from the URL scheme.
const (
	TypeHTTP = "http"
	TypeTCP  = "tcp"
	TypeSSL  = "ssl"
	TypeDNS  = "dns"
	TypePing = "ping"
	// TypePush monitors are not checked at all, they wait for heartbeats.
	TypePush = "push"

	// CertWarnDays is how many days before certificate expiry an ssl monitor
	// counts as degraded and subscribers get a warning.
	CertWarnDays = 30

	// maxKeywordBody caps how much of a response is searched for a keyword.
	maxKeywordBody = 1 << 20
)

// Schemes are the URL schemes a monitor may use.
var Schemes = []string{"http", "https", "tcp", "ssl", "dns", "ping", "push"}

// ValidScheme reports whether scheme is one a monitor may use.
func ValidScheme(scheme string) bool {
	return slices.Contains(Schemes, scheme)
}

// TypeForScheme maps a URL scheme to its check type, defaulting to http.
func TypeForScheme(scheme string) string {
	switch scheme {
	case "tcp", "ssl", "dns", "ping", "push":
		return scheme
	default:
		return TypeHTTP
	}
}

type Checker struct {
	client *http.Client
	dialer *net.Dialer

	Timeout  time.Duration `env:"BEACON_TIMEOUT"  envDefault:"30s"`
	Insecure bool          `env:"BEACON_INSECURE" envDefault:"false"`
}

type Result struct {
	StatusCode   int64  // HTTP status, zero for every other check
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
	return newChecker(c.Timeout, c.Insecure), nil
}

func newChecker(timeout time.Duration, insecure bool) *Checker {
	c := &Checker{Timeout: timeout, Insecure: insecure}
	c.dialer = &net.Dialer{Timeout: timeout}
	c.client = &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext:       c.dialer.DialContext,
			TLSClientConfig:   c.tlsConfig(),
			DisableKeepAlives: true,
		},
	}
	return c
}

func (c *Checker) tlsConfig() *tls.Config {
	return &tls.Config{InsecureSkipVerify: c.Insecure} // #nosec G402
}

// Check dispatches on the URL scheme of the monitor.
func (c *Checker) Check(ctx context.Context, m *db.Monitor) Result {
	u, err := url.Parse(m.Url)
	if err != nil {
		return failed("invalid url", err, 0)
	}

	switch u.Scheme {
	case "http", "https":
		return c.checkHTTP(ctx, m)
	case "tcp":
		if u.Port() == "" {
			return failedMsg("connection failed: tcp url requires an explicit port, e.g. tcp://host:5432", 0)
		}
		return c.checkTCP(ctx, u.Host)
	case "ssl":
		addr := u.Host
		if u.Port() == "" {
			addr = net.JoinHostPort(u.Hostname(), "443")
		}
		return c.checkSSL(ctx, addr)
	case "dns":
		return c.checkDNS(ctx, u.Hostname(), u.Query().Get("server"))
	case "ping":
		return c.checkPing(ctx, u.Hostname())
	default:
		return failedMsg(fmt.Sprintf("connection failed: unsupported scheme %q", u.Scheme), 0)
	}
}

func (c *Checker) checkHTTP(ctx context.Context, m *db.Monitor) Result {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.Url, nil)
	if err != nil {
		return failed("request failed", err, 0)
	}
	req.Header.Set("User-Agent", "Beacon/1.0")
	req.Header.Set("Accept", "*/*")

	resp, err := c.client.Do(req)
	if err != nil {
		return failed("request failed", err, time.Since(start).Milliseconds())
	}
	defer func() { _ = resp.Body.Close() }()

	var body []byte
	if m.Keyword != "" {
		body, err = io.ReadAll(io.LimitReader(resp.Body, maxKeywordBody))
		if err != nil {
			return failed("reading body failed", err, time.Since(start).Milliseconds())
		}
	}

	result := Result{
		StatusCode:   int64(resp.StatusCode),
		ResponseTime: time.Since(start).Milliseconds(),
	}
	// https responses carry the peer certificate from the handshake, so every
	// https monitor tracks expiry for free.
	if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		result.DaysLeft = certDaysLeft(resp.TLS.PeerCertificates[0])
	}

	switch {
	case m.ExpectedStatus != 0 && result.StatusCode != m.ExpectedStatus:
		result.Error = new(fmt.Sprintf("HTTP %d, expected %d", result.StatusCode, m.ExpectedStatus))
	case m.ExpectedStatus == 0 && (resp.StatusCode < 200 || resp.StatusCode >= 400):
		result.Error = new(fmt.Sprintf("HTTP %d", result.StatusCode))
	case m.Keyword != "" && !strings.Contains(string(body), m.Keyword):
		result.Error = new(fmt.Sprintf("keyword %q not found in response", m.Keyword))
	default:
		result.IsUp = true
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

// checkDNS resolves host, through server when one is given. Pointing it at
// your own resolver is the usual way to watch a Pi-hole or AdGuard.
func (c *Checker) checkDNS(ctx context.Context, host, server string) Result {
	ctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	resolver := net.DefaultResolver
	if server != "" {
		if _, _, err := net.SplitHostPort(server); err != nil {
			server = net.JoinHostPort(server, "53")
		}
		resolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
				return c.dialer.DialContext(ctx, network, server)
			},
		}
	}

	start := time.Now()
	addrs, err := resolver.LookupHost(ctx, host)
	ms := time.Since(start).Milliseconds()
	if err != nil {
		return failed("dns lookup failed", err, ms)
	}
	if len(addrs) == 0 {
		return failedMsg("dns lookup failed: no addresses", ms)
	}
	return Result{IsUp: true, ResponseTime: ms}
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
	return failedMsg(fmt.Sprintf("%s: %v", reason, err), responseTime)
}

func failedMsg(msg string, responseTime int64) Result {
	return Result{
		IsUp:         false,
		Error:        &msg,
		ResponseTime: responseTime,
	}
}
