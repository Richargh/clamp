package acceptance_test

import (
	"crypto/x509"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"sync"
	"testing"

	"github.com/richargh/clamp/mitm-proxy/internal/kernel/environment"
	"github.com/stretchr/testify/require"
)

type destinationSpy struct {
	t        *testing.T
	baseURL  *url.URL
	server   *httptest.Server
	mu       sync.Mutex
	requests []receivedRequest
	status   int
	headers  http.Header
	body     string
}

type receivedRequest struct {
	Method string
	URL    url.URL
	Header http.Header
	Body   string
	Err    error
}

func (d *destinationSpy) URLFor(reference string) string {
	return d.resolve(d.baseURL, reference)
}

func (d *destinationSpy) resolve(base *url.URL, reference string) string {
	d.t.Helper()
	relative, err := url.Parse(reference)
	require.NoError(d.t, err)
	return base.ResolveReference(relative).String()
}

func (d *destinationSpy) Certificate() *x509.Certificate {
	return d.server.Certificate()
}

func (d *destinationSpy) RespondWith(status int, headers http.Header, body string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.status, d.headers, d.body = status, headers.Clone(), body
}

func (d *destinationSpy) ReceivedRequests() []receivedRequest {
	d.mu.Lock()
	defer d.mu.Unlock()
	return slices.Clone(d.requests)
}

func (d *destinationSpy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	d.mu.Lock()
	d.requests = append(d.requests, receivedRequest{
		Method: r.Method, URL: *r.URL, Header: r.Header.Clone(), Body: string(body), Err: err,
	})
	status, headers, responseBody := d.status, d.headers.Clone(), d.body
	d.mu.Unlock()
	for name, values := range headers {
		w.Header()[name] = values
	}
	w.WriteHeader(status)
	_, _ = io.WriteString(w, responseBody)
}

func NewHTTPDestinationSpy(t *testing.T, env *environment.Env) *destinationSpy {
	t.Helper()
	return newDestinationSpy(t, env, false)
}

func NewHTTPSDestinationSpy(t *testing.T, env *environment.Env) *destinationSpy {
	t.Helper()
	return newDestinationSpy(t, env, true)
}

func newDestinationSpy(t *testing.T, env *environment.Env, encrypted bool) *destinationSpy {
	t.Helper()
	addresses, err := env.Network.LookupIP(t.Context(), MustHost(t, "example.com"))
	require.NoError(t, err)
	require.NotEmpty(t, addresses)
	spy := &destinationSpy{t: t, status: http.StatusNoContent}
	spy.server = Server(t, env.Network, net.JoinHostPort(addresses[0].String(), "0"), encrypted, spy)
	localURL, err := url.Parse(spy.server.URL)
	require.NoError(t, err)
	baseURL := *localURL
	baseURL.Host = net.JoinHostPort("example.com", baseURL.Port())
	spy.baseURL = &baseURL
	return spy
}
