package acceptance_test

import (
	"testing"

	"github.com/richargh/clamp/mitm-proxy/internal/alloweddomains"
	"github.com/richargh/clamp/mitm-proxy/internal/forwardproxy"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/environment"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http/httpproxy"
)

func TestHTTPHostProxyRouting(t *testing.T) {
	for _, scenario := range []struct {
		behavior         string
		configure        bool
		noProxy          string
		upstreamRequests int32
	}{
		{behavior: "Should use the configured host proxy for HTTP", configure: true, upstreamRequests: 1},
		{behavior: "Should connect directly for HTTP when NO_PROXY matches", configure: true, noProxy: "example.com"},
		{behavior: "Should connect directly for HTTP when no host proxy is configured"},
	} {
		ForEachEnvironment(t, scenario.behavior, func(t *testing.T, env *environment.Env) {
			// given
			destination := NewHTTPDestinationSpy(t, env)
			config := httpproxy.Config{}
			var upstream *upstreamProxySpy
			if scenario.configure {
				upstream = NewUpstreamProxySpy(t, env)
				config = httpproxy.Config{HTTPProxy: upstream.URL, NoProxy: scenario.noProxy}
			}
			request := NewPOSTRequest(destination.URLFor("/"), "")
			testee := forwardproxy.NewHandler(env, forwardproxy.Options{Proxy: config.ProxyFunc(), AllowedDomains: alloweddomains.AllowAll()})

			// when
			ExecuteHTTPRequest(testee.ServeHTTP, request)

			// then
			received := RequireSingle(t, destination.ReceivedRequests())
			require.NoError(t, received.Err)
			if upstream != nil {
				require.Equal(t, scenario.upstreamRequests, upstream.RequestCount())
			}
		})
	}
}

func TestHTTPSHostProxyRouting(t *testing.T) {
	for _, scenario := range []struct {
		behavior         string
		configure        bool
		noProxy          string
		upstreamRequests int32
	}{
		{behavior: "Should use the configured host proxy for HTTPS", configure: true, upstreamRequests: 1},
		{behavior: "Should connect directly for HTTPS when NO_PROXY matches", configure: true, noProxy: "example.com"},
		{behavior: "Should connect directly for HTTPS when no host proxy is configured"},
	} {
		ForEachEnvironment(t, scenario.behavior, func(t *testing.T, env *environment.Env) {
			// given
			destination := NewHTTPSDestinationSpy(t, env)
			config := httpproxy.Config{}
			var upstream *upstreamProxySpy
			if scenario.configure {
				upstream = NewUpstreamProxySpy(t, env)
				config = httpproxy.Config{HTTPSProxy: upstream.URL, NoProxy: scenario.noProxy}
			}
			request := NewPOSTRequest(destination.URLFor("/"), "")
			testee := forwardproxy.NewHandler(env, forwardproxy.Options{Proxy: config.ProxyFunc(), AllowedDomains: alloweddomains.AllowAll()})

			// when
			ExecuteHTTPSRequest(testee.Serve, t, env, request, destination)

			// then
			received := RequireSingle(t, destination.ReceivedRequests())
			require.NoError(t, received.Err)
			if upstream != nil {
				require.Equal(t, scenario.upstreamRequests, upstream.RequestCount())
			}
		})
	}
}
