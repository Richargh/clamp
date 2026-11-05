package acceptance_test

import (
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/richargh/clamp/mitm-proxy/internal/alloweddomains"
	"github.com/richargh/clamp/mitm-proxy/internal/forwardproxy"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/environment"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http/httpproxy"
)

func TestHTTPUpstreamProxyLogging(t *testing.T) {
	for _, scenario := range []struct {
		name    string
		noProxy string
	}{
		{name: "Should log the selected HTTP upstream proxy without credentials or URL details"},
		{name: "Should not log HTTP upstream proxy use when NO_PROXY matches", noProxy: "example.com"},
	} {
		ForEachEnvironment(t, scenario.name, func(t *testing.T, env *environment.Env) {
			// given
			const secret = "proxy-secret-canary"
			var output LogBuffer
			env.Stdout = &output
			destination := NewHTTPDestinationSpy(t, env)
			upstream := NewUpstreamProxySpy(t, env)
			proxyURL := upstream.URLWithDetails(t, url.URL{
				User: url.UserPassword(secret, secret),
				Path: "/" + secret, RawQuery: "token=" + secret, Fragment: secret,
			})
			config := httpproxy.Config{HTTPProxy: proxyURL.String(), NoProxy: scenario.noProxy}
			request := NewPOSTRequest(destination.URLFor("/"), "")
			testee := forwardproxy.NewHandler(env, forwardproxy.Options{Proxy: config.ProxyFunc(), AllowedDomains: alloweddomains.AllowAll()})

			// when
			ExecuteHTTPRequest(testee.ServeHTTP, request)

			// then
			logs := output.String()
			require.NotContains(t, logs, secret)
			if scenario.noProxy == "" {
				require.Contains(t, logs, fmt.Sprintf("UPSTREAM proxy=%q", proxyURL.Host))
				require.Equal(t, 1, strings.Count(logs, "UPSTREAM proxy="))
			} else {
				require.NotContains(t, logs, "UPSTREAM proxy=")
				require.NotContains(t, logs, "warning=")
			}
		})
	}
}

func TestHTTPSUpstreamProxyLogging(t *testing.T) {
	for _, scenario := range []struct {
		name    string
		noProxy string
	}{
		{name: "Should log the selected HTTPS upstream proxy without credentials or URL details"},
		{name: "Should not log HTTPS upstream proxy use when NO_PROXY matches", noProxy: "example.com"},
	} {
		ForEachEnvironment(t, scenario.name, func(t *testing.T, env *environment.Env) {
			// given
			const secret = "proxy-secret-canary"
			var output LogBuffer
			env.Stdout = &output
			destination := NewHTTPSDestinationSpy(t, env)
			upstream := NewUpstreamProxySpy(t, env)
			proxyURL := upstream.URLWithDetails(t, url.URL{
				User: url.UserPassword(secret, secret),
				Path: "/" + secret, RawQuery: "token=" + secret, Fragment: secret,
			})
			config := httpproxy.Config{HTTPSProxy: proxyURL.String(), NoProxy: scenario.noProxy}
			request := NewPOSTRequest(destination.URLFor("/"), "")
			testee := forwardproxy.NewHandler(env, forwardproxy.Options{Proxy: config.ProxyFunc(), AllowedDomains: alloweddomains.AllowAll()})

			// when
			ExecuteHTTPSRequest(testee.Serve, t, env, request, destination)

			// then
			logs := output.String()
			require.NotContains(t, logs, secret)
			if scenario.noProxy == "" {
				require.Contains(t, logs, fmt.Sprintf("UPSTREAM proxy=%q", proxyURL.Host))
				require.Equal(t, 1, strings.Count(logs, "UPSTREAM proxy="))
			} else {
				require.NotContains(t, logs, "UPSTREAM proxy=")
				require.NotContains(t, logs, "warning=")
			}
		})
	}
}
