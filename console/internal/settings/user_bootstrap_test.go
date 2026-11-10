package settings

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserBootstrap(t *testing.T) {
	t.Run("Should resolve user preferences from host inputs", func(t *testing.T) {
		// given
		testee := BootstrapUser

		// when
		actual, err := testee("/preferences", "9.9.9.9", "UTC")

		// then
		require.NoError(t, err)
		require.Equal(t, "/preferences/clamp/auth-projects.tsv", actual.AuthStore().String())
		require.Equal(t, "9.9.9.9", actual.ContainerDNS().String())
		require.Equal(t, "UTC", actual.ContainerTimezone().String())
	})
	t.Run("Should ignore an invalid detected timezone", func(t *testing.T) {
		// given
		testee := BootstrapUser

		// when
		actual, err := testee("/preferences", "", "invalid\ntimezone")

		// then
		require.NoError(t, err)
		require.Empty(t, actual.ContainerTimezone().String())
	})
	t.Run("Should reject invalid DNS before services run", func(t *testing.T) {
		// given
		testee := BootstrapUser

		// when
		_, err := testee("/preferences", "invalid", "UTC")

		// then
		require.ErrorContains(t, err, "container DNS")
	})
}
