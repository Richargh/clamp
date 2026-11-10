package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStartupConfiguration(t *testing.T) {
	t.Run("Should resolve environment settings once before services run", func(t *testing.T) {
		// given
		t.Setenv("XDG_CONFIG_HOME", "/preferences")
		t.Setenv("CLAMP_CONTAINER_DNS", "9.9.9.9")
		t.Setenv("TZ", "Europe/Berlin")

		// when
		testee, err := startupConfig()
		t.Setenv("CLAMP_CONTAINER_DNS", "1.0.0.1")
		t.Setenv("TZ", "UTC")

		// then
		require.NoError(t, err)
		require.Equal(t, "/preferences/clamp/auth-projects.tsv", testee.User().AuthStore().String())
		require.Equal(t, "9.9.9.9", testee.User().ContainerDNS().String())
		require.Equal(t, "Europe/Berlin", testee.User().ContainerTimezone().String())
	})
	t.Run("Should reject invalid DNS before services run", func(t *testing.T) {
		// given
		t.Setenv("XDG_CONFIG_HOME", "/preferences")
		t.Setenv("CLAMP_CONTAINER_DNS", "invalid")
		t.Setenv("TZ", "UTC")

		// when
		_, err := startupConfig()

		// then
		require.ErrorContains(t, err, "container DNS")
	})
}
