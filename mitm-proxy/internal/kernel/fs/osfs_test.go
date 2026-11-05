package fs

import (
	stdfs "io/fs"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOSFileSystem(t *testing.T) {
	t.Run("Should retain the standard error and path for a missing file", func(t *testing.T) {
		// given
		testee := NewOS()
		path := mustPath(t, filepath.Join(t.TempDir(), "missing.txt"))

		// when
		_, err := testee.ReadFile(path)

		// then
		require.ErrorIs(t, err, stdfs.ErrNotExist)
		var pathError *stdfs.PathError
		require.ErrorAs(t, err, &pathError)
		require.Equal(t, path.String(), pathError.Path)
	})
	t.Run("Should list an empty directory", func(t *testing.T) {
		// given
		testee := NewOS()
		path := mustPath(t, t.TempDir())

		// when
		entries, err := testee.ListDirectory(path)

		// then
		require.NoError(t, err)
		require.Empty(t, entries)
	})
}
