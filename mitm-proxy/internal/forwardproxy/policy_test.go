package forwardproxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/elazarl/goproxy"
	"github.com/richargh/clamp/mitm-proxy/internal/alloweddomains"
	"github.com/stretchr/testify/require"
)

func TestDestinationPolicy(t *testing.T) {
	for _, scenario := range []struct {
		name   string
		method string
		target string
	}{
		{name: "Should reject HTTP without logging hooks", method: http.MethodGet, target: "http://example.com/"},
		{name: "Should reject CONNECT without logging hooks", method: http.MethodConnect, target: "example.com:443"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// given
			allowed, err := alloweddomains.FromAddresses()
			require.NoError(t, err)
			policy := destinationPolicy{allowed: allowed}
			testee := goproxy.NewProxyHttpServer()
			testee.OnRequest().Do(policy)
			testee.OnRequest().HandleConnect(policy)
			request := httptest.NewRequest(scenario.method, scenario.target, nil)
			response := httptest.NewRecorder()

			// when
			testee.ServeHTTP(response, request)

			// then
			require.Equal(t, http.StatusForbidden, response.Code)
		})
	}
	t.Run("Should reject CONNECT without a destination policy", func(t *testing.T) {
		// given
		policy := destinationPolicy{}
		testee := goproxy.NewProxyHttpServer()
		testee.OnRequest().HandleConnect(policy)
		request := httptest.NewRequest(http.MethodConnect, "example.com:443", nil)
		response := httptest.NewRecorder()

		// when
		testee.ServeHTTP(response, request)

		// then
		require.Equal(t, http.StatusForbidden, response.Code)
	})
	t.Run("Should record a blocked CONNECT decision", func(t *testing.T) {
		// given
		allowed, err := alloweddomains.FromAddresses()
		require.NoError(t, err)
		testee := destinationPolicy{allowed: allowed}
		request := httptest.NewRequest(http.MethodConnect, "example.com:443", nil)
		ctx := &goproxy.ProxyCtx{Req: request}

		// when
		testee.HandleConnect("example.com:443", ctx)

		// then
		decision, ok := ctx.UserData.(destinationPolicyDecision)
		require.True(t, ok)
		require.False(t, decision.allowed)
		require.Equal(t, "example.com:443", decision.address.String())
	})
}
