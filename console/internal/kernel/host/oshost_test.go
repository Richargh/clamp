package host

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAssetDirectory(t *testing.T) {
	t.Run("Should ignore project assets in the working directory", func(t *testing.T) {
		// given
		root := t.TempDir()
		assets := filepath.Join(root, "installation")
		project := filepath.Join(root, "project")
		marker := "asset-directory-test.Dockerfile"
		for _, dir := range []string{assets, project} {
			require.NoError(t, os.MkdirAll(dir, 0700))
			require.NoError(t, os.WriteFile(filepath.Join(dir, marker), []byte("FROM scratch\n"), 0600))
		}
		t.Chdir(project)
		testee := AssetDirectory

		// when
		actual, err := testee(marker, filepath.Join(assets, "cmd", "main.go"))

		// then
		require.NoError(t, err)
		require.Equal(t, assets, actual)
	})
	t.Run("Should reject project assets reached through a relative source path", func(t *testing.T) {
		// given
		project := t.TempDir()
		marker := "relative-source-test.Dockerfile"
		require.NoError(t, os.WriteFile(filepath.Join(project, marker), []byte("FROM scratch\n"), 0600))
		t.Chdir(project)
		testee := AssetDirectory

		// when
		_, err := testee(marker, "cmd/main.go")

		// then
		require.ErrorContains(t, err, "cannot find asset directory")
	})
	t.Run("Should reject a directory used as an asset marker", func(t *testing.T) {
		// given
		assets := t.TempDir()
		marker := "directory-marker-test.Dockerfile"
		require.NoError(t, os.Mkdir(filepath.Join(assets, marker), 0700))
		testee := AssetDirectory

		// when
		_, err := testee(marker, filepath.Join(assets, "cmd", "main.go"))

		// then
		require.ErrorContains(t, err, "cannot find asset directory")
	})
}

func TestConfigDirectory(t *testing.T) {
	t.Run("Should use the XDG configuration directory", func(t *testing.T) {
		// given
		t.Setenv("XDG_CONFIG_HOME", "/preferences")

		// when
		testee, err := ConfigDirectory()

		// then
		require.NoError(t, err)
		require.Equal(t, "/preferences", testee)
	})
	t.Run("Should fall back to the home configuration directory", func(t *testing.T) {
		// given
		t.Setenv("XDG_CONFIG_HOME", "")
		t.Setenv("HOME", "/home/test-user")

		// when
		testee, err := ConfigDirectory()

		// then
		require.NoError(t, err)
		require.Equal(t, "/home/test-user/.config", testee)
	})
}

func TestHostTimezone(t *testing.T) {
	t.Run("Should use the explicit TZ setting", func(t *testing.T) {
		// given
		t.Setenv("TZ", "Europe/Berlin")

		// when
		testee := Timezone()

		// then
		require.Equal(t, "Europe/Berlin", testee)
	})
}
