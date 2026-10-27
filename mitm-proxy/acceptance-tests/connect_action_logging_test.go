package acceptance_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/richargh/clamp/mitm-proxy/internal/forwardproxy"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/environment"
	"github.com/stretchr/testify/require"
)

func TestCONNECTActionLogging(t *testing.T) {
	ForEachEnvironment(t, "Should log CONNECT setup without HTTPS request details or secrets", func(t *testing.T, env *environment.Env) {
		// given
		const secret = "secret-canary"
		destination := NewHTTPSDestinationSpy(t, env)
		destination.RespondWith(http.StatusCreated, http.Header{"X-Secret": {secret}}, secret)
		request := NewPOSTRequest(destination.URLFor("/"+secret+"?token="+secret+"#"+secret), secret)
		request.Header.Set("Authorization", "Bearer "+secret)
		request.Header.Set("Cookie", "session="+secret)
		var output LogBuffer
		env.Stdout = &output
		testee := forwardproxy.NewHandler(env, forwardproxy.Options{Proxy: DirectProxy})

		// when
		ExecuteHTTPSRequest(testee.Serve, t, env, request, destination)

		// then
		logs := output.String()
		require.NotContains(t, logs, secret)
		require.Equal(t, 1, strings.Count(logs, "\n"))
		require.Contains(t, logs, "HTTPS CONNECT destination=\"example.com:")
		require.Contains(t, logs, "outcome=connected")
		require.NotContains(t, logs, "status=")
		require.NotContains(t, logs, "method=")
	})
	ForEachEnvironment(t, "Should log CONNECT rejection without the invalid authority", func(t *testing.T, env *environment.Env) {
		// given
		const secret = "secret-canary"
		request := httptest.NewRequest(http.MethodConnect, secret+":0", nil)
		var output LogBuffer
		env.Stdout = &output
		testee := forwardproxy.NewHandler(env, forwardproxy.Options{Proxy: DirectProxy})
		proxy := ServeHandler(t, env, testee)
		conn := Connection(t, env, proxy)

		// when
		err := request.WriteProxy(conn)
		require.NoError(t, err)
		_, err = io.ReadAll(conn)

		// then
		require.NoError(t, err)
		require.Contains(t, output.String(), "HTTPS CONNECT outcome=rejected")
		require.NotContains(t, output.String(), secret)
	})
}
