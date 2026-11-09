package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfiguration(t *testing.T) {
	t.Run("Should use the default container DNS when none is supplied", func(t *testing.T) {
		// given
		raw := Raw{AssetsDir: "/assets", AuthStore: "/preferences/auth.tsv"}

		// when
		testee, err := Parse(raw)

		// then
		require.NoError(t, err)
		require.Equal(t, "1.1.1.1", testee.ContainerDNS().String())
	})
	t.Run("Should retain an explicit IPv6 DNS server", func(t *testing.T) {
		// given
		raw := Raw{AssetsDir: "/assets", AuthStore: "/preferences/auth.tsv", ContainerDNS: "2606:4700:4700::1111"}

		// when
		testee, err := Parse(raw)

		// then
		require.NoError(t, err)
		require.Equal(t, raw.ContainerDNS, testee.ContainerDNS().String())
	})
	t.Run("Should reject an invalid container DNS server", func(t *testing.T) {
		// given
		raw := Raw{AssetsDir: "/assets", AuthStore: "/preferences/auth.tsv", ContainerDNS: "not-an-address"}

		// when
		_, err := Parse(raw)

		// then
		require.ErrorContains(t, err, "container DNS")
	})
	for _, scenario := range []struct {
		name string
		raw  Raw
	}{
		{"assets", Raw{AssetsDir: "relative", AuthStore: "/preferences/auth.tsv"}},
		{"auth store", Raw{AssetsDir: "/assets", AuthStore: "relative"}},
	} {
		t.Run("Should reject a relative "+scenario.name+" path", func(t *testing.T) {
			// given
			raw := scenario.raw

			// when
			_, err := Parse(raw)

			// then
			require.ErrorContains(t, err, "absolute file path")
		})
	}
	t.Run("Should retain the configured asset directory", func(t *testing.T) {
		// given
		raw := Raw{AssetsDir: "/assets/custom", AuthStore: "/preferences/auth.tsv"}

		// when
		testee, err := Parse(raw)

		// then
		require.NoError(t, err)
		require.Equal(t, raw.AssetsDir, testee.AssetsDir().String())
	})
	t.Run("Should retain the configured auth store path", func(t *testing.T) {
		// given
		raw := Raw{AssetsDir: "/assets", AuthStore: "/preferences/custom.tsv"}

		// when
		testee, err := Parse(raw)

		// then
		require.NoError(t, err)
		require.Equal(t, raw.AuthStore, testee.AuthStore().String())
	})
}

func TestTimezone(t *testing.T) {
	for _, raw := range []string{"", "UTC", "Europe/Berlin", "Etc/GMT+2"} {
		t.Run("Should accept timezone "+raw, func(t *testing.T) {
			// given
			// when
			testee, err := ParseTimezone(raw)

			// then
			require.NoError(t, err)
			require.Equal(t, raw, testee.String())
		})
	}
	t.Run("Should reject an invalid timezone setting", func(t *testing.T) {
		// given
		raw := "Europe/Berlin\nextra"

		// when
		_, err := Parse(Raw{AssetsDir: "/assets", AuthStore: "/preferences/auth.tsv", ContainerTimezone: raw})

		// then
		require.ErrorContains(t, err, "invalid container timezone")
	})
}
