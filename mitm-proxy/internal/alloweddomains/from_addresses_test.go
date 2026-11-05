package alloweddomains

import (
	"testing"

	"github.com/richargh/clamp/mitm-proxy/internal/kernel/network"
	"github.com/stretchr/testify/require"
)

func TestFromAddresses(t *testing.T) {
	t.Run("Should block all destinations when no addresses are supplied", func(t *testing.T) {
		// given
		address := mustAddress(t, "example.com:443")

		// when
		testee, err := FromAddresses()

		// then
		require.NoError(t, err)
		require.False(t, testee.IsAllowed(address))
	})
	t.Run("Should keep the supplied addresses independent of later slice changes", func(t *testing.T) {
		// given
		original := mustAddress(t, "example.com:443")
		replacement := mustAddress(t, "other.example:443")
		addresses := []network.Address{original}
		testee, err := FromAddresses(addresses...)
		require.NoError(t, err)

		// when
		addresses[0] = replacement

		// then
		require.True(t, testee.IsAllowed(original))
		require.False(t, testee.IsAllowed(replacement))
	})
	for _, target := range []string{"", "127.0.0.1:443", "[::1]:443", "example.com:0"} {
		t.Run("Should reject a non-domain endpoint "+target, func(t *testing.T) {
			// given
			var address network.Address
			if target != "" {
				address = mustAddress(t, target)
			}

			// when
			testee, err := FromAddresses(address)

			// then
			require.Error(t, err)
			require.Nil(t, testee)
		})
	}
}
