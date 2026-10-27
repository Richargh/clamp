package acceptance_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/richargh/clamp/mitm-proxy/internal/forwardproxy"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/environment"
	"github.com/stretchr/testify/require"
)

func TestHTTPActionLogging(t *testing.T) {
	ForEachEnvironment(t, "Should log HTTP response metadata without secrets", func(t *testing.T, env *environment.Env) {
		// given
		const secret = "secret-canary"
		destination := NewHTTPDestinationSpy(t, env)
		destination.RespondWith(http.StatusCreated, http.Header{"X-Secret": {secret}}, secret)
		request := NewPOSTRequest(destination.URLFor("/"+secret+"?token="+secret+"#"+secret), secret)
		request.Header.Set("Authorization", "Bearer "+secret)
		request.Header.Set("Cookie", "session="+secret)
		var output LogBuffer
		env.Stdout = &output
		testee := forwardproxy.NewHandler(env, forwardproxy.Options{Proxy: DirectProxy})

		// when
		ExecuteHTTPRequest(testee.ServeHTTP, request)

		// then
		logs := output.String()
		require.NotContains(t, logs, secret)
		require.Equal(t, 1, strings.Count(logs, "\n"))
		require.Contains(t, logs, "HTTP method=\"POST\" destination=\"example.com:")
		require.Contains(t, logs, "status=201")
	})
}
