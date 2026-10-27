package acceptance_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/richargh/clamp/mitm-proxy/internal/forwardproxy"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/environment"
	"github.com/stretchr/testify/require"
)

func TestProxySelection(t *testing.T) {
	ForEachEnvironment(t, "Should use the supplied direct proxy selector for HTTP", func(t *testing.T, env *environment.Env) {
		// given
		destination := NewHTTPDestinationSpy(t, env)
		request := NewPOSTRequest(destination.URLFor("/"), "")
		var selections atomic.Int32
		testee := forwardproxy.NewHandler(env, forwardproxy.Options{
			Proxy: func(*url.URL) (*url.URL, error) {
				selections.Add(1)
				return nil, nil
			},
		})

		// when
		ExecuteHTTPRequest(testee.ServeHTTP, request)

		// then
		received := RequireSingle(t, destination.ReceivedRequests())
		require.NoError(t, received.Err)
		require.Equal(t, int32(1), selections.Load())
	})
	ForEachEnvironment(t, "Should use the supplied direct proxy selector for CONNECT", func(t *testing.T, env *environment.Env) {
		// given
		destination := NewHTTPDestinationSpy(t, env)
		destinationURL, err := url.Parse(destination.URLFor("/"))
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodConnect, destinationURL.Host, nil)
		var selections atomic.Int32
		testee := forwardproxy.NewHandler(env, forwardproxy.Options{
			Proxy: func(*url.URL) (*url.URL, error) {
				selections.Add(1)
				return nil, nil
			},
		})
		proxy := ServeHandler(t, env, testee)
		conn := Connection(t, env, proxy)

		// when
		response, err := ExchangeRequest(conn, request)

		// then
		require.NoError(t, err)
		defer func() {
			_ = conn.Close()
			_ = response.Body.Close()
		}()
		require.Equal(t, http.StatusOK, response.StatusCode)
		require.Equal(t, int32(1), selections.Load())
	})
}
