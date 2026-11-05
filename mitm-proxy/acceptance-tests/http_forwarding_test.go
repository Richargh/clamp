package acceptance_test

import (
	"net/http"
	"testing"

	"github.com/richargh/clamp/mitm-proxy/internal/alloweddomains"
	"github.com/richargh/clamp/mitm-proxy/internal/forwardproxy"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/environment"
	"github.com/stretchr/testify/require"
)

func TestHTTPForwarding(t *testing.T) {
	for _, setup := range []struct {
		name           string
		newEnvironment func(*testing.T) *environment.Env
	}{
		{"Should forward HTTP requests unchanged with an in-memory environment", NewMemoryEnvironment},
		{"Should forward HTTP requests unchanged with an OS environment", NewOSEnvironment},
		{"Should forward HTTP unchanged to 8.8.8.8 with an in-memory environment", func(t *testing.T) *environment.Env {
			return MemoryEnvironmentAtIP(t, "8.8.8.8")
		}},
		{"Should forward HTTP unchanged to 10.0.0.1 with an in-memory environment", func(t *testing.T) *environment.Env {
			return MemoryEnvironmentAtIP(t, "10.0.0.1")
		}},
	} {
		t.Run(setup.name, func(t *testing.T) {
			// given
			env := setup.newEnvironment(t)
			destination := NewHTTPDestinationSpy(t, env)
			request := NewPOSTRequest(destination.URLFor("/path?query=kept"), "payload")
			request.Header.Set("Content-Type", "text/plain")
			testee := forwardproxy.NewHandler(env, forwardproxy.Options{Proxy: DirectProxy, AllowedDomains: alloweddomains.AllowAll()})

			// when
			ExecuteHTTPRequest(testee.ServeHTTP, request)

			// then
			received := RequireSingle(t, destination.ReceivedRequests())
			require.NoError(t, received.Err)
			require.Equal(t, http.MethodPost, received.Method)
			require.Equal(t, "/path", received.URL.Path)
			require.Equal(t, "query=kept", received.URL.RawQuery)
			require.Equal(t, "text/plain", received.Header.Get("Content-Type"))
			require.Equal(t, "payload", received.Body)
		})
	}
	ForEachEnvironment(t, "Should return HTTP response status headers and body unchanged", func(t *testing.T, env *environment.Env) {
		// given
		destination := NewHTTPDestinationSpy(t, env)
		destination.RespondWith(http.StatusCreated, http.Header{"X-Destination": {"unchanged"}}, "destination response")
		request := NewPOSTRequest(destination.URLFor("/"), "payload")
		testee := forwardproxy.NewHandler(env, forwardproxy.Options{Proxy: DirectProxy, AllowedDomains: alloweddomains.AllowAll()})

		// when
		response := RecordHTTPResponse(testee.ServeHTTP, request)

		// then
		require.Equal(t, http.StatusCreated, response.StatusCode)
		require.Equal(t, "unchanged", response.Header.Get("X-Destination"))
		require.Equal(t, "destination response", ReadAndCloseResponseBody(t, response))
	})
}
