package cli

import (
	"testing"

	"github.com/richargh/clamp/console/internal/config"
	"github.com/richargh/clamp/console/internal/kernel/environment"
	"github.com/stretchr/testify/require"
)

func TestVersionFromMemory(t *testing.T) {
	t.Run("Should read the version from the injected file system", func(t *testing.T) {
		// given
		memory := environment.NewMemory()
		settings, err := config.Parse(config.Raw{AssetsDir: "/assets", AuthStore: "/config/auth.tsv"})
		require.NoError(t, err)
		require.NoError(t, memory.Files.MkdirAll("/assets", 0700))
		require.NoError(t, memory.Files.WriteFile("/assets/VERSION", []byte("2.3.4\n"), 0600))
		testee := New(memory.Env, settings)

		// when
		err = testee.Run(t.Context(), []string{"--version"})

		// then
		require.NoError(t, err)
		require.Equal(t, "Clamp 2.3.4\n", memory.Output.String())
	})
}
