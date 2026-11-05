package forwardproxy

import (
	"io"
	"testing"

	"github.com/elazarl/goproxy"
	"github.com/stretchr/testify/require"
)

func TestConnectLoggingDecorator(t *testing.T) {
	for _, scenario := range []struct {
		name   string
		action *goproxy.ConnectAction
	}{
		{name: "Should preserve a CONNECT acceptance", action: goproxy.OkConnect},
		{name: "Should preserve a CONNECT rejection", action: goproxy.RejectConnect},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// given
			const destination = "example.com:8443"
			policy := goproxy.FuncHttpsHandler(func(string, *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
				return scenario.action, destination
			})
			testee := withConnectLogging(policy, newActionLogger(io.Discard))
			ctx := &goproxy.ProxyCtx{}

			// when
			action, address := testee.HandleConnect("EXAMPLE.COM:8443", ctx)

			// then
			require.Same(t, scenario.action, action)
			require.Equal(t, destination, address)
		})
	}
	t.Run("Should pass the original CONNECT input to the policy", func(t *testing.T) {
		// given
		const authority = "example.com:443"
		ctx := &goproxy.ProxyCtx{}
		var receivedAuthority string
		var receivedContext *goproxy.ProxyCtx
		policy := goproxy.FuncHttpsHandler(func(authority string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
			receivedAuthority, receivedContext = authority, ctx
			return goproxy.RejectConnect, authority
		})
		testee := withConnectLogging(policy, newActionLogger(io.Discard))

		// when
		testee.HandleConnect(authority, ctx)

		// then
		require.Equal(t, authority, receivedAuthority)
		require.Same(t, ctx, receivedContext)
	})
}
