package acceptance_test

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/richargh/clamp/mitm-proxy/internal/kernel/environment"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/network"
)

func ProxyClient(t *testing.T, env *environment.Env, proxy *url.URL, trustedOrigin *httptest.Server) *http.Client {
	t.Helper()
	roots := x509.NewCertPool()
	if trustedOrigin != nil && trustedOrigin.Certificate() != nil {
		roots.AddCert(trustedOrigin.Certificate())
	}
	transport := &http.Transport{
		Proxy:           http.ProxyURL(proxy),
		DialContext:     testDialContext(env),
		TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12},
	}
	t.Cleanup(transport.CloseIdleConnections)
	return &http.Client{Transport: transport, Timeout: 5 * time.Second}
}

func ExchangeRequest(conn net.Conn, request *http.Request) (*http.Response, error) {
	if err := request.WriteProxy(conn); err != nil {
		return nil, err
	}
	return http.ReadResponse(bufio.NewReader(conn), request)
}

func testDialContext(env *environment.Env) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, _, address string) (net.Conn, error) {
		target, err := network.ParseAddress(address)
		if err != nil {
			return nil, err
		}
		return env.Network.Dial(ctx, target)
	}
}

func Connection(t *testing.T, env *environment.Env, proxy *url.URL) net.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	conn, err := env.Network.Dial(ctx, MustAddress(t, proxy.Host))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	return conn
}
