package acceptance_test

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/richargh/clamp/mitm-proxy/internal/alloweddomains"
	"github.com/richargh/clamp/mitm-proxy/internal/forwardproxy"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/environment"
	"github.com/stretchr/testify/require"
)

func TestDomainRules(t *testing.T) {
	ForEachEnvironment(t, "Should block HTTP with no destination policy", func(t *testing.T, env *environment.Env) {
		// given
		testee := forwardproxy.NewHandler(env, forwardproxy.Options{Proxy: DirectProxy})
		request := NewPOSTRequest("http://example.com/", "")

		// when
		response := RecordHTTPResponse(testee.ServeHTTP, request)
		defer func() { require.NoError(t, response.Body.Close()) }()

		// then
		require.Equal(t, http.StatusForbidden, response.StatusCode)
	})
	ForEachEnvironment(t, "Should block CONNECT with no destination policy", func(t *testing.T, env *environment.Env) {
		// given
		testee := forwardproxy.NewHandler(env, forwardproxy.Options{Proxy: DirectProxy})
		proxy := ServeHandler(t, env, testee)
		conn := Connection(t, env, proxy)
		request := httptest.NewRequest(http.MethodConnect, "example.com:443", nil)

		// when
		response, err := ExchangeRequest(conn, request)
		require.NoError(t, err)
		defer func() { require.NoError(t, response.Body.Close()) }()

		// then
		require.Equal(t, http.StatusForbidden, response.StatusCode)
	})
	ForEachEnvironment(t, "Should forward an allowed HTTP destination", func(t *testing.T, env *environment.Env) {
		// given
		destination := NewHTTPDestinationSpy(t, env)
		target, err := url.Parse(destination.URLFor("/allowed"))
		require.NoError(t, err)
		allowed, err := alloweddomains.FromAddresses(MustAddress(t, target.Host))
		require.NoError(t, err)
		testee := forwardproxy.NewHandler(env, forwardproxy.Options{Proxy: DirectProxy, AllowedDomains: allowed})
		request := NewPOSTRequest(target.String(), "payload")

		// when
		ExecuteHTTPRequest(testee.ServeHTTP, request)

		// then
		require.Len(t, destination.ReceivedRequests(), 1)
	})
	for _, target := range []string{"http://blocked.example/", "http://sub.example.com/", "http://example.com:81/", "http://127.0.0.1/"} {
		ForEachEnvironment(t, "Should block HTTP destination "+target, func(t *testing.T, env *environment.Env) {
			// given
			allowed, err := alloweddomains.FromAddresses(MustAddress(t, "example.com:80"))
			require.NoError(t, err)
			testee := forwardproxy.NewHandler(env, forwardproxy.Options{Proxy: DirectProxy, AllowedDomains: allowed})
			request := NewPOSTRequest(target, "")

			// when
			response := RecordHTTPResponse(testee.ServeHTTP, request)
			defer func() { require.NoError(t, response.Body.Close()) }()

			// then
			require.Equal(t, http.StatusForbidden, response.StatusCode)
		})
	}
	ForEachEnvironment(t, "Should block a CONNECT destination", func(t *testing.T, env *environment.Env) {
		// given
		allowed, err := alloweddomains.FromAddresses()
		require.NoError(t, err)
		testee := forwardproxy.NewHandler(env, forwardproxy.Options{Proxy: DirectProxy, AllowedDomains: allowed})
		proxy := ServeHandler(t, env, testee)
		conn := Connection(t, env, proxy)
		request := httptest.NewRequest(http.MethodConnect, "example.com:443", nil)

		// when
		require.NoError(t, request.WriteProxy(conn))
		response, err := http.ReadResponse(bufio.NewReader(conn), request)
		require.NoError(t, err)
		defer func() { require.NoError(t, response.Body.Close()) }()

		// then
		require.Equal(t, http.StatusForbidden, response.StatusCode)
	})
	ForEachEnvironment(t, "Should establish an allowed CONNECT tunnel", func(t *testing.T, env *environment.Env) {
		// given
		destination := NewHTTPSDestinationSpy(t, env)
		target, err := url.Parse(destination.URLFor("/allowed"))
		require.NoError(t, err)
		allowed, err := alloweddomains.FromAddresses(MustAddress(t, target.Host))
		require.NoError(t, err)
		testee := forwardproxy.NewHandler(env, forwardproxy.Options{Proxy: DirectProxy, AllowedDomains: allowed})
		request := NewPOSTRequest(target.String(), "payload")

		// when
		ExecuteHTTPSRequest(testee.Serve, t, env, request, destination)

		// then
		require.Len(t, destination.ReceivedRequests(), 1)
	})
	ForEachEnvironment(t, "Should allow CONNECT with no firewall", func(t *testing.T, env *environment.Env) {
		// given
		destination := NewHTTPSDestinationSpy(t, env)
		testee := forwardproxy.NewHandler(env, forwardproxy.Options{Proxy: DirectProxy, AllowedDomains: alloweddomains.AllowAll()})
		request := NewPOSTRequest(destination.URLFor("/unrestricted"), "")

		// when
		ExecuteHTTPSRequest(testee.Serve, t, env, request, destination)

		// then
		require.Len(t, destination.ReceivedRequests(), 1)
	})
	ForEachEnvironment(t, "Should allow HTTP with no firewall", func(t *testing.T, env *environment.Env) {
		// given
		destination := NewHTTPDestinationSpy(t, env)
		testee := forwardproxy.NewHandler(env, forwardproxy.Options{Proxy: DirectProxy, AllowedDomains: alloweddomains.AllowAll()})
		request := NewPOSTRequest(destination.URLFor("/unrestricted"), "")

		// when
		ExecuteHTTPRequest(testee.ServeHTTP, request)

		// then
		require.Len(t, destination.ReceivedRequests(), 1)
	})
}
