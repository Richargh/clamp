package settings

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserSettings(t *testing.T) {
	t.Run("Should parse user defaults without installation or project paths", func(t *testing.T) {
		// given
		raw := RawUser{AuthStore: "/preferences/auth-projects.tsv"}

		// when
		testee, err := ParseUser(raw)

		// then
		require.NoError(t, err)
		require.Equal(t, raw.AuthStore, testee.AuthStore().String())
		require.Equal(t, "1.1.1.1", testee.ContainerDNS().String())
		require.Empty(t, testee.ContainerTimezone().String())
	})
}
