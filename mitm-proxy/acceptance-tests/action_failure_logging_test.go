package acceptance_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/richargh/clamp/mitm-proxy/internal/alloweddomains"
	"github.com/richargh/clamp/mitm-proxy/internal/forwardproxy"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/environment"
	"github.com/stretchr/testify/require"
)

func TestActionFailureLogging(t *testing.T) {
	for _, scenario := range []struct {
		name, method, target string
	}{
		{"Should log HTTP failures without raw error text", http.MethodGet, "http://example.com:443/upstream-error-canary"},
		{"Should log CONNECT failures without raw error text", http.MethodConnect, "example.com:443"},
	} {
		ForEachEnvironment(t, scenario.name, func(t *testing.T, env *environment.Env) {
			// given
			const secret = "upstream-error-canary"
			FailDial(env, MustAddress(t, "127.0.0.1:443"), errors.New(secret))
			request := httptest.NewRequest(scenario.method, scenario.target, nil)
			request.Close = true
			var output LogBuffer
			env.Stdout = &output
			testee := forwardproxy.NewHandler(env, forwardproxy.Options{Proxy: DirectProxy, AllowedDomains: alloweddomains.AllowAll()})

			// when
			if scenario.method == http.MethodConnect {
				ExecuteCONNECTRequest(testee.Serve, t, env, request)
			} else {
				ExecuteHTTPRequest(testee.ServeHTTP, request)
			}

			// then
			require.Contains(t, output.String(), "outcome=failed")
			require.NotContains(t, output.String(), secret)
		})
	}
}
