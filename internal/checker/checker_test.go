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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newTestChecker(insecure bool) *Checker {
	return &Checker{
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure}, // #nosec G402
			},
		},
		dialer:   &net.Dialer{Timeout: 2 * time.Second},
		Insecure: insecure,
	}
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

	result := newTestChecker(false).Check(t.Context(), server.URL)
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

	result := newTestChecker(true).Check(t.Context(), server.URL)
	require.True(t, result.IsUp)
	require.NotNil(t, result.DaysLeft, "https checks must carry the certificate lifetime")
	require.GreaterOrEqual(t, *result.DaysLeft, int64(0))
}

func TestCheckTCP(t *testing.T) {
	t.Parallel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	result := newTestChecker(false).Check(t.Context(), "tcp://"+listener.Addr().String())
	require.True(t, result.IsUp)
	require.Nil(t, result.Error)
	require.Zero(t, result.StatusCode)
}

func TestCheckTCPRefused(t *testing.T) {
	t.Parallel()

	result := newTestChecker(false).Check(t.Context(), "tcp://127.0.0.1:1")
	require.False(t, result.IsUp)
	require.NotNil(t, result.Error)
}

func TestCheckTCPRequiresPort(t *testing.T) {
	t.Parallel()

	result := newTestChecker(false).Check(t.Context(), "tcp://localhost")
	require.False(t, result.IsUp)
	require.Contains(t, *result.Error, "port")
}

func TestCheckUnsupportedScheme(t *testing.T) {
	t.Parallel()

	result := newTestChecker(false).Check(t.Context(), "ftp://example.com")
	require.False(t, result.IsUp)
	require.Contains(t, *result.Error, "unsupported scheme")
}

func TestCheckSSLValidCertificate(t *testing.T) {
	t.Parallel()

	addr := startTLSServer(t, selfSignedCert(t, time.Now().Add(40*24*time.Hour)))

	result := newTestChecker(true).Check(t.Context(), "ssl://"+addr)
	require.True(t, result.IsUp)
	require.Nil(t, result.Error)
	require.NotNil(t, result.DaysLeft)
	require.GreaterOrEqual(t, *result.DaysLeft, int64(38))
	require.LessOrEqual(t, *result.DaysLeft, int64(40))
}

func TestCheckSSLExpiredCertificate(t *testing.T) {
	t.Parallel()

	addr := startTLSServer(t, selfSignedCert(t, time.Now().Add(-24*time.Hour)))

	result := newTestChecker(true).Check(t.Context(), "ssl://"+addr)
	require.False(t, result.IsUp)
	require.NotNil(t, result.Error)
	require.Contains(t, *result.Error, "expired")
}

func TestCheckSSLUntrustedCertificate(t *testing.T) {
	t.Parallel()

	addr := startTLSServer(t, selfSignedCert(t, time.Now().Add(40*24*time.Hour)))

	result := newTestChecker(false).Check(t.Context(), "ssl://"+addr)
	require.False(t, result.IsUp)
	require.NotNil(t, result.Error)
	require.Contains(t, *result.Error, "handshake")
}
