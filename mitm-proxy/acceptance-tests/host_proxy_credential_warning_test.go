package acceptance_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/richargh/clamp/mitm-proxy/internal/alloweddomains"
	"github.com/richargh/clamp/mitm-proxy/internal/forwardproxy"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/environment"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http/httpproxy"
)

func TestHostProxyCredentialWarnings(t *testing.T) {
	ForEachEnvironment(t, "Should not warn about host proxy credentials when constructing the handler", func(t *testing.T, env *environment.Env) {
		// given
		hostProxy := &url.URL{
			Scheme: "http", Host: "127.0.0.1:3128",
			User: url.UserPassword("username-canary", "password-canary"),
		}
		config := httpproxy.Config{HTTPProxy: hostProxy.String(), HTTPSProxy: hostProxy.String()}
		var output LogBuffer
		env.Stdout = &output

		// when
		testee := forwardproxy.NewHandler(env, forwardproxy.Options{Proxy: config.ProxyFunc(), AllowedDomains: alloweddomains.AllowAll()})

		// then
		require.NotNil(t, testee)
		require.Empty(t, output.String())
	})
	for _, exchange := range []struct {
		name, method, target string
	}{
		{"HTTP requests", http.MethodGet, "http://example.com:443/path"},
		{"CONNECT tunnels", http.MethodConnect, "example.com:443"},
	} {
		for _, scenario := range []struct {
			name        string
			scheme      string
			credentials *url.Userinfo
			warn        bool
		}{
			{
				name: "Should warn about credentials sent to an HTTP host proxy", scheme: "http",
				credentials: url.UserPassword("username-canary", "password-canary"), warn: true,
			},
			{
				name: "Should warn about a username sent to an HTTP host proxy", scheme: "http",
				credentials: url.User("username-canary"), warn: true,
			},
			{
				name: "Should not warn about an HTTP host proxy without credentials", scheme: "http",
			},
			{
				name: "Should not warn about credentials sent to an HTTPS host proxy", scheme: "https",
				credentials: url.UserPassword("username-canary", "password-canary"),
			},
		} {
			ForEachEnvironment(t, scenario.name+" for "+exchange.name, func(t *testing.T, env *environment.Env) {
				// given
				hostProxy := &url.URL{Scheme: scenario.scheme, Host: "127.0.0.1:3128", User: scenario.credentials}
				FailDial(env, MustAddress(t, hostProxy.Host), errors.New("host proxy unavailable"))
				config := httpproxy.Config{HTTPProxy: hostProxy.String(), HTTPSProxy: hostProxy.String()}
				request := httptest.NewRequest(exchange.method, exchange.target, nil)
				request.Close = true
				var output LogBuffer
				env.Stdout = &output
				testee := forwardproxy.NewHandler(env, forwardproxy.Options{Proxy: config.ProxyFunc(), AllowedDomains: alloweddomains.AllowAll()})

				// when
				if exchange.method == http.MethodConnect {
					ExecuteCONNECTRequest(testee.Serve, t, env, request)
				} else {
					ExecuteHTTPRequest(testee.ServeHTTP, request)
				}

				// then
				logs := output.String()
				require.Contains(t, logs, "UPSTREAM proxy=\"127.0.0.1:3128\"")
				require.NotContains(t, logs, "username-canary")
				require.NotContains(t, logs, "password-canary")
				if scenario.warn {
					require.Contains(t, logs, "warning=\"proxy credentials are sent over unencrypted HTTP\"")
				} else {
					require.NotContains(t, logs, "warning=")
				}
			})
		}
	}
}
