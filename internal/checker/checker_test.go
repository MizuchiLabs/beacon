package checker

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/mizuchilabs/beacon/internal/db"
)

func newTestChecker(insecure bool) *Checker {
	return newChecker(2*time.Second, insecure)
}

func monitor(url string) *db.Monitor {
	return &db.Monitor{Url: url}
}

// selfSignedCert mints a one-off certificate. IsCA keeps the TLS server happy
// without a chain, verification is controlled per test through Insecure.
func selfSignedCert(t *testing.T, notAfter time.Time) tls.Certificate {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "beacon.test"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              notAfter,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:              []string{"localhost"},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	require.NoError(t, err)

	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

func startTLSServer(t *testing.T, cert tls.Certificate) string {
	t.Helper()

	listener, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{cert}}) // #nosec G402
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				if tlsConn, ok := conn.(*tls.Conn); ok {
					_ = tlsConn.HandshakeContext(context.Background())
				}
				_ = conn.Close()
			}()
		}
	}()

	return listener.Addr().String()
}

func TestCheckHTTP(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	result := newTestChecker(false).Check(t.Context(), monitor(server.URL))
	require.True(t, result.IsUp)
	require.Nil(t, result.Error)
	require.EqualValues(t, http.StatusOK, result.StatusCode)
}

func TestCheckHTTPCapturesCertificateExpiry(t *testing.T) {
	t.Parallel()

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	result := newTestChecker(true).Check(t.Context(), monitor(server.URL))
	require.True(t, result.IsUp)
	require.NotNil(t, result.DaysLeft, "https checks must carry the certificate lifetime")
	require.GreaterOrEqual(t, *result.DaysLeft, int64(0))
}

func TestCheckTCP(t *testing.T) {
	t.Parallel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	result := newTestChecker(false).Check(t.Context(), monitor("tcp://"+listener.Addr().String()))
	require.True(t, result.IsUp)
	require.Nil(t, result.Error)
	require.Zero(t, result.StatusCode)
}

func TestCheckTCPRefused(t *testing.T) {
	t.Parallel()

	result := newTestChecker(false).Check(t.Context(), monitor("tcp://127.0.0.1:1"))
	require.False(t, result.IsUp)
	require.NotNil(t, result.Error)
}

func TestCheckTCPRequiresPort(t *testing.T) {
	t.Parallel()

	result := newTestChecker(false).Check(t.Context(), monitor("tcp://localhost"))
	require.False(t, result.IsUp)
	require.Contains(t, *result.Error, "port")
}

func TestCheckUnsupportedScheme(t *testing.T) {
	t.Parallel()

	result := newTestChecker(false).Check(t.Context(), monitor("ftp://example.com"))
	require.False(t, result.IsUp)
	require.Contains(t, *result.Error, "unsupported scheme")
}

func TestCheckSSLValidCertificate(t *testing.T) {
	t.Parallel()

	addr := startTLSServer(t, selfSignedCert(t, time.Now().Add(40*24*time.Hour)))

	result := newTestChecker(true).Check(t.Context(), monitor("ssl://"+addr))
	require.True(t, result.IsUp)
	require.Nil(t, result.Error)
	require.NotNil(t, result.DaysLeft)
	require.GreaterOrEqual(t, *result.DaysLeft, int64(38))
	require.LessOrEqual(t, *result.DaysLeft, int64(40))
}

func TestCheckSSLExpiredCertificate(t *testing.T) {
	t.Parallel()

	addr := startTLSServer(t, selfSignedCert(t, time.Now().Add(-24*time.Hour)))

	result := newTestChecker(true).Check(t.Context(), monitor("ssl://"+addr))
	require.False(t, result.IsUp)
	require.NotNil(t, result.Error)
	require.Contains(t, *result.Error, "expired")
}

func TestCheckSSLUntrustedCertificate(t *testing.T) {
	t.Parallel()

	addr := startTLSServer(t, selfSignedCert(t, time.Now().Add(40*24*time.Hour)))

	result := newTestChecker(false).Check(t.Context(), monitor("ssl://"+addr))
	require.False(t, result.IsUp)
	require.NotNil(t, result.Error)
	require.Contains(t, *result.Error, "handshake")
}

func TestCheckHTTPTimesOutOnHangingServer(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		<-release
	}))
	t.Cleanup(func() {
		close(release)
		server.Close()
	})

	start := time.Now()
	result := newChecker(200*time.Millisecond, false).Check(t.Context(), monitor(server.URL))
	require.False(t, result.IsUp)
	require.Less(t, time.Since(start), 2*time.Second, "a server that never answers must not block the check")
}

func TestCheckHTTPStatusAndKeyword(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("please log in"))
	}))
	t.Cleanup(server.Close)
	c := newTestChecker(false)

	result := c.Check(t.Context(), monitor(server.URL))
	require.False(t, result.IsUp, "4xx is down by default")
	require.Equal(t, "HTTP 401", *result.Error)

	result = c.Check(t.Context(), &db.Monitor{Url: server.URL, ExpectedStatus: 401})
	require.True(t, result.IsUp, "an expected 401 is up")

	result = c.Check(t.Context(), &db.Monitor{Url: server.URL, ExpectedStatus: 401, Keyword: "log in"})
	require.True(t, result.IsUp)

	result = c.Check(t.Context(), &db.Monitor{Url: server.URL, ExpectedStatus: 401, Keyword: "welcome"})
	require.False(t, result.IsUp)
	require.Contains(t, *result.Error, "keyword")
}

func TestCheckDNS(t *testing.T) {
	t.Parallel()

	require.True(t, newTestChecker(false).Check(t.Context(), monitor("dns://localhost")).IsUp)

	result := newTestChecker(false).Check(t.Context(), monitor("dns://beacon.invalid?server=127.0.0.1:1"))
	require.False(t, result.IsUp, "an unreachable resolver is down")
}

func TestCheckPing(t *testing.T) {
	t.Parallel()

	result := newTestChecker(false).Check(t.Context(), monitor("ping://127.0.0.1"))
	if !result.IsUp && strings.Contains(*result.Error, "ping_group_range") {
		t.Skip("unprivileged ICMP is not allowed on this host")
	}
	require.True(t, result.IsUp, "ping failed: %v", result.Error)
}
