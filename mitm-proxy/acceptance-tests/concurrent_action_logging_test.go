package acceptance_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/richargh/clamp/mitm-proxy/internal/forwardproxy"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/environment"
	"github.com/stretchr/testify/require"
)

func TestConcurrentActionLogging(t *testing.T) {
	ForEachEnvironment(t, "Should write one complete log line per concurrent HTTP request", func(t *testing.T, env *environment.Env) {
		// given
		const requests = 16
		destination := NewHTTPDestinationSpy(t, env)
		destination.RespondWith(http.StatusNoContent, nil, "")
		var output LogBuffer
		env.Stdout = &output
		testee := forwardproxy.NewHandler(env, forwardproxy.Options{Proxy: DirectProxy})
		var done sync.WaitGroup

		// when
		for range requests {
			done.Go(func() {
				request := httptest.NewRequest(http.MethodGet, destination.URLFor("/"), nil)
				ExecuteHTTPRequest(testee.ServeHTTP, request)
			})
		}
		done.Wait()

		// then
		lines := strings.Split(strings.TrimSpace(output.String()), "\n")
		require.Len(t, lines, requests)
		for _, line := range lines {
			require.Equal(t, 1, strings.Count(line, "HTTP method=\"GET\""))
			require.True(t, strings.HasSuffix(line, "status=204"), "incomplete log line: %q", line)
		}
	})
}
