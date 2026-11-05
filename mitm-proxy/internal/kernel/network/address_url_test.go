package network_test

import (
	"net/url"
	"testing"

	"github.com/richargh/clamp/mitm-proxy/internal/kernel/network"
	"github.com/stretchr/testify/require"
)

func TestAddressFromURL(t *testing.T) {
	for _, scenario := range []struct {
		name  string
		value string
		want  string
	}{
		{name: "Should use the default HTTP port", value: "http://example.com/path", want: "example.com:80"},
		{name: "Should use the default HTTPS port", value: "https://example.com/path", want: "example.com:443"},
		{name: "Should use an explicit port instead of the default", value: "http://example.com:8443/path", want: "example.com:8443"},
		{name: "Should use an explicit port without a scheme", value: "//example.com:8443", want: "example.com:8443"},
		{name: "Should use an explicit port with an unknown scheme", value: "custom://example.com:8443", want: "example.com:8443"},
		{name: "Should normalize the hostname", value: "https://EXAMPLE.COM./", want: "example.com:443"},
		{name: "Should ignore non-address URL fields", value: "https://user:password@example.com/path?query=value#fragment", want: "example.com:443"},
		{name: "Should construct an IPv4 address", value: "http://127.0.0.1/", want: "127.0.0.1:80"},
		{name: "Should construct an IPv6 address with a default port", value: "https://[2001:db8::1]/", want: "[2001:db8::1]:443"},
		{name: "Should construct an IPv6 address with an explicit port", value: "https://[2001:db8::1]:8443/", want: "[2001:db8::1]:8443"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// given
			value, err := url.Parse(scenario.value)
			require.NoError(t, err)

			// when
			testee, err := network.AddressFromURL(value)

			// then
			require.NoError(t, err)
			require.Equal(t, scenario.want, testee.String())
		})
	}
}

func TestAddressFromURLRejectsInvalidDestination(t *testing.T) {
	for _, host := range []string{"", "user@example.com:443", "example.com:0", "example.com:65536", "example.com:", "example.com:https", "example.com:-1", "example.com/path", "example.com\n", "[example.com:443", "[::1", "fe80::1%lo0"} {
		t.Run("Should reject invalid URL host "+host, func(t *testing.T) {
			// given
			value := &url.URL{Scheme: "https", Host: host}

			// when
			testee, err := network.AddressFromURL(value)

			// then
			require.Error(t, err)
			require.False(t, testee.IsValid())
		})
	}
	t.Run("Should reject a missing URL", func(t *testing.T) {
		// given
		var value *url.URL

		// when
		testee, err := network.AddressFromURL(value)

		// then
		require.Error(t, err)
		require.False(t, testee.IsValid())
	})
	for _, scheme := range []string{"", "custom"} {
		t.Run("Should reject a missing port without a known scheme "+scheme, func(t *testing.T) {
			// given
			value := &url.URL{Scheme: scheme, Host: "example.com"}

			// when
			testee, err := network.AddressFromURL(value)

			// then
			require.Error(t, err)
			require.False(t, testee.IsValid())
		})
	}
}
