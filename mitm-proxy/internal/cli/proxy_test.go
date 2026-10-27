package cli

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProxySelection(t *testing.T) {
	for _, scenario := range []struct {
		name       string
		target     string
		httpProxy  string
		httpsProxy string
		noProxy    string
		ignore     bool
		want       string
	}{
		{
			name: "Should select the HTTP host proxy", target: "http://example.com/path",
			httpProxy: "http://proxy.example:3128", want: "http://proxy.example:3128",
		},
		{
			name: "Should select the HTTPS host proxy", target: "https://example.com/path",
			httpsProxy: "http://proxy.example:3128", want: "http://proxy.example:3128",
		},
		{
			name: "Should connect directly for HTTP when host proxy settings are ignored", target: "http://example.com/path",
			httpProxy: "http://127.0.0.1:1", ignore: true,
		},
		{
			name: "Should connect directly for HTTPS when host proxy settings are ignored", target: "https://example.com/path",
			httpsProxy: "http://127.0.0.1:1", ignore: true,
		},
		{
			name: "Should connect directly when NO_PROXY matches", target: "http://example.com/path",
			httpProxy: "http://proxy.example:3128", noProxy: "example.com",
		},
		{
			name: "Should connect directly when no host proxy is configured", target: "http://example.com/path",
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// given
			for _, variable := range []string{"http_proxy", "https_proxy", "no_proxy", "REQUEST_METHOD"} {
				t.Setenv(variable, "")
			}
			t.Setenv("HTTP_PROXY", scenario.httpProxy)
			t.Setenv("HTTPS_PROXY", scenario.httpsProxy)
			t.Setenv("NO_PROXY", scenario.noProxy)
			target, err := url.Parse(scenario.target)
			require.NoError(t, err)
			testee := proxyFromEnvironment(scenario.ignore)

			// when
			proxy, err := testee(target)

			// then
			require.NoError(t, err)
			if scenario.want == "" {
				require.Nil(t, proxy)
			} else {
				require.NotNil(t, proxy)
				require.Equal(t, scenario.want, proxy.String())
			}
		})
	}
	t.Run("Should keep host proxy settings captured when the selector is constructed", func(t *testing.T) {
		// given
		for _, variable := range []string{"http_proxy", "NO_PROXY", "no_proxy", "REQUEST_METHOD"} {
			t.Setenv(variable, "")
		}
		t.Setenv("HTTP_PROXY", "http://original.example:3128")
		target, err := url.Parse("http://example.com/path")
		require.NoError(t, err)
		testee := proxyFromEnvironment(false)
		t.Setenv("HTTP_PROXY", "http://changed.example:3128")

		// when
		proxy, err := testee(target)

		// then
		require.NoError(t, err)
		require.NotNil(t, proxy)
		require.Equal(t, "http://original.example:3128", proxy.String())
	})
}
