package acceptance_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/richargh/clamp/mitm-proxy/internal/alloweddomains"
	"github.com/richargh/clamp/mitm-proxy/internal/forwardproxy"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/environment"
	"github.com/stretchr/testify/require"
)

func TestDomainPolicyLogging(t *testing.T) {
	ForEachEnvironment(t, "Should log an allowed HTTP destination", func(t *testing.T, env *environment.Env) {
		// given
		destination := NewHTTPDestinationSpy(t, env)
		target, err := url.Parse(destination.URLFor("/allowed"))
		require.NoError(t, err)
		allowed, err := alloweddomains.FromAddresses(MustAddress(t, target.Host))
		require.NoError(t, err)
		var output LogBuffer
		env.Stdout = &output
		testee := forwardproxy.NewHandler(env, forwardproxy.Options{Proxy: DirectProxy, AllowedDomains: allowed})
		request := NewPOSTRequest(target.String(), "")

		// when
		ExecuteHTTPRequest(testee.ServeHTTP, request)

		// then
		require.Contains(t, output.String(), "HTTP method=\"POST\" destination=\""+target.Host+"\"")
	})
	ForEachEnvironment(t, "Should log a blocked HTTP destination", func(t *testing.T, env *environment.Env) {
		// given
		var output LogBuffer
		env.Stdout = &output
		allowed, err := alloweddomains.FromAddresses()
		require.NoError(t, err)
		testee := forwardproxy.NewHandler(env, forwardproxy.Options{Proxy: DirectProxy, AllowedDomains: allowed})
		request := NewPOSTRequest("http://blocked.example/", "")

		// when
		ExecuteHTTPRequest(testee.ServeHTTP, request)

		// then
		require.Contains(t, output.String(), "destination=\"blocked.example:80\" status=403")
	})
	ForEachEnvironment(t, "Should log a blocked CONNECT destination", func(t *testing.T, env *environment.Env) {
		// given
		var output LogBuffer
		env.Stdout = &output
		allowed, err := alloweddomains.FromAddresses()
		require.NoError(t, err)
		testee := forwardproxy.NewHandler(env, forwardproxy.Options{Proxy: DirectProxy, AllowedDomains: allowed})
		request := httptest.NewRequest(http.MethodConnect, "example.com:443", nil)

		// when
		ExecuteCONNECTRequest(testee.Serve, t, env, request)

		// then
		require.Contains(t, output.String(), "destination=\"example.com:443\" outcome=rejected")
	})
}
