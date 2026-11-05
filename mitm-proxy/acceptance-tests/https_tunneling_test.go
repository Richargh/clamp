package acceptance_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/richargh/clamp/mitm-proxy/internal/alloweddomains"
	"github.com/richargh/clamp/mitm-proxy/internal/forwardproxy"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/environment"
	"github.com/stretchr/testify/require"
)

func TestHTTPSTunneling(t *testing.T) {
	for _, setup := range []struct {
		name           string
		newEnvironment func(*testing.T) *environment.Env
	}{
		{"in-memory environment", NewMemoryEnvironment},
		{"OS environment", NewOSEnvironment},
		{"in-memory environment at 8.8.8.8", func(t *testing.T) *environment.Env { return MemoryEnvironmentAtIP(t, "8.8.8.8") }},
		{"in-memory environment at 10.0.0.1", func(t *testing.T) *environment.Env { return MemoryEnvironmentAtIP(t, "10.0.0.1") }},
	} {
		t.Run("Should tunnel HTTPS requests unchanged with an "+setup.name, func(t *testing.T) {
			// given
			env := setup.newEnvironment(t)
			destination := NewHTTPSDestinationSpy(t, env)
			request := NewPOSTRequest(destination.URLFor("/path?query=kept"), "payload")
			request.Header.Set("Content-Type", "text/plain")
			testee := forwardproxy.NewHandler(env, forwardproxy.Options{Proxy: DirectProxy, AllowedDomains: alloweddomains.AllowAll()})

			// when
			ExecuteHTTPSRequest(testee.Serve, t, env, request, destination)

			// then
			received := RequireSingle(t, destination.ReceivedRequests())
			require.NoError(t, received.Err)
			require.Equal(t, http.MethodPost, received.Method)
			require.Equal(t, "/path", received.URL.Path)
			require.Equal(t, "query=kept", received.URL.RawQuery)
			require.Equal(t, "text/plain", received.Header.Get("Content-Type"))
			require.Equal(t, "payload", received.Body)
		})
		t.Run("Should tunnel HTTPS response status headers and body unchanged with an "+setup.name, func(t *testing.T) {
			// given
			env := setup.newEnvironment(t)
			destination := NewHTTPSDestinationSpy(t, env)
			destination.RespondWith(http.StatusCreated, http.Header{"X-Destination": {"unchanged"}}, "destination response")
			request := NewPOSTRequest(destination.URLFor("/"), "")
			testee := forwardproxy.NewHandler(env, forwardproxy.Options{Proxy: DirectProxy, AllowedDomains: alloweddomains.AllowAll()})

			// when
			response := RecordHTTPSResponse(testee.Serve, t, env, request, destination)

			// then
			defer func() { _ = response.Body.Close() }()
			require.Equal(t, http.StatusCreated, response.StatusCode)
			require.Equal(t, "unchanged", response.Header.Get("X-Destination"))
			require.Equal(t, "destination response", ReadAndCloseResponseBody(t, response))
		})
		t.Run("Should preserve the HTTPS destination certificate with an "+setup.name, func(t *testing.T) {
			// given
			env := setup.newEnvironment(t)
			destination := NewHTTPSDestinationSpy(t, env)
			request := NewPOSTRequest(destination.URLFor("/"), "")
			testee := forwardproxy.NewHandler(env, forwardproxy.Options{Proxy: DirectProxy, AllowedDomains: alloweddomains.AllowAll()})

			// when
			response := RecordHTTPSResponse(testee.Serve, t, env, request, destination)

			// then
			defer func() { _ = response.Body.Close() }()
			require.NotNil(t, response.TLS)
			require.NotEmpty(t, response.TLS.PeerCertificates)
			require.True(t, response.TLS.PeerCertificates[0].Equal(destination.Certificate()), "client must receive the destination certificate")
		})
		t.Run("Should reject an untrusted HTTPS destination with an "+setup.name, func(t *testing.T) {
			// given
			env := setup.newEnvironment(t)
			destination := NewHTTPSDestinationSpy(t, env)
			request := NewPOSTRequest(destination.URLFor("/"), "payload")
			testee := forwardproxy.NewHandler(env, forwardproxy.Options{Proxy: DirectProxy, AllowedDomains: alloweddomains.AllowAll()})

			// when
			response, err := ExchangeHTTPSRequest(testee.Serve, t, env, request, nil)
			if response != nil {
				_ = response.Body.Close()
			}

			// then
			require.Error(t, err)
			require.Empty(t, destination.ReceivedRequests())
		})
	}
	for _, authority := range []string{"example.com:0", "example.com:65536"} {
		ForEachEnvironment(t, "Should reject CONNECT to invalid destination "+authority, func(t *testing.T, env *environment.Env) {
			// given
			request := httptest.NewRequest(http.MethodConnect, authority, nil)
			testee := forwardproxy.NewHandler(env, forwardproxy.Options{Proxy: DirectProxy, AllowedDomains: alloweddomains.AllowAll()})
			proxy := ServeHandler(t, env, testee)
			conn := Connection(t, env, proxy)

			// when
			response, err := ExchangeRequest(conn, request)

			// then
			if err != nil {
				// GoProxy can close rejected CONNECTs without an HTTP response.
				require.True(t, err == io.EOF || err == io.ErrUnexpectedEOF, "unexpected rejection error: %v", err)
				return
			}
			defer func() { _ = response.Body.Close() }()
			require.NotEqual(t, http.StatusOK, response.StatusCode)
		})
	}
}
