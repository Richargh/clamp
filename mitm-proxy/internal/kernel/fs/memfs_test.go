package fs

import (
	stdfs "io/fs"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMemoryFileSystem(t *testing.T) {
	t.Run("Should copy input file contents", func(t *testing.T) {
		// given
		path := mustPath(t, "rules/rules.txt")
		contents := []byte("example.com:443")
		testee := NewMemory(MemoryOptions{Files: map[Path][]byte{path: contents}})

		// when
		contents[0] = 'X'
		stored, err := testee.ReadFile(path)

		// then
		require.NoError(t, err)
		require.Equal(t, "example.com:443", string(stored))
	})
	for _, directory := range []string{".", "rules", "rules/nested"} {
		t.Run("Should infer directory "+directory+" from a file path", func(t *testing.T) {
			// given
			testee := NewMemory(MemoryOptions{Files: map[Path][]byte{
				mustPath(t, "rules/nested/rules.txt"): []byte("example.com:443"),
			}})

			// when
			isDirectory, err := testee.IsDirectory(mustPath(t, directory))

			// then
			require.NoError(t, err)
			require.True(t, isDirectory)
		})
	}
	t.Run("Should report no directories when no files are stored", func(t *testing.T) {
		// given
		testee := NewMemory(MemoryOptions{})

		// when
		_, err := testee.ListDirectory(mustPath(t, "rules"))

		// then
		require.ErrorIs(t, err, stdfs.ErrNotExist)
	})
}
