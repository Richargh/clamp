package filesystem

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOSPathResolution(t *testing.T) {
	t.Run("Should resolve a symbolic link in an ancestor directory", func(t *testing.T) {
		// given
		root, err := filepath.EvalSymlinks(t.TempDir())
		require.NoError(t, err)
		directory := filepath.Join(root, "actual")
		require.NoError(t, os.Mkdir(directory, 0700))
		path := filepath.Join(directory, "file")
		require.NoError(t, os.WriteFile(path, []byte("contents"), 0600))
		link := filepath.Join(root, "alias")
		require.NoError(t, os.Symlink(directory, link))
		testee := NewOS()

		// when
		resolved, err := testee.Resolve(filepath.Join(link, "file"))

		// then
		require.NoError(t, err)
		require.Equal(t, path, resolved)
	})
}
