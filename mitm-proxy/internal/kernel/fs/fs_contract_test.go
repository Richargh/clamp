package fs

import (
	stdfs "io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFileSystemContract(t *testing.T) {
	for _, adapter := range []struct {
		name string
		new  func(*testing.T) (FileSystem, Path)
	}{
		{"memory", newMemoryFixture},
		{"OS", newOSFixture},
	} {
		t.Run(adapter.name, func(t *testing.T) {
			t.Run("Should list immediate children sorted by name", func(t *testing.T) {
				// given
				testee, path := adapter.new(t)

				// when
				entries, err := testee.ListDirectory(path)

				// then
				require.NoError(t, err)
				require.Equal(t, []Entry{{Name: "nested", IsDirectory: true}, {Name: "rules.txt"}}, entries)
			})
			t.Run("Should read file contents", func(t *testing.T) {
				// given
				testee, directory := adapter.new(t)
				path := mustJoin(t, directory, "rules.txt")

				// when
				data, err := testee.ReadFile(path)

				// then
				require.NoError(t, err)
				require.Equal(t, "example.com:443\n", string(data))
			})
			t.Run("Should list a nested directory", func(t *testing.T) {
				// given
				testee, directory := adapter.new(t)
				path := mustJoin(t, directory, "nested")

				// when
				entries, err := testee.ListDirectory(path)

				// then
				require.NoError(t, err)
				require.Equal(t, []Entry{{Name: "info.txt"}}, entries)
			})
			t.Run("Should identify a directory", func(t *testing.T) {
				// given
				testee, path := adapter.new(t)

				// when
				isDirectory, err := testee.IsDirectory(path)

				// then
				require.NoError(t, err)
				require.True(t, isDirectory)
			})
			t.Run("Should identify a file as not a directory", func(t *testing.T) {
				// given
				testee, directory := adapter.new(t)
				path := mustJoin(t, directory, "rules.txt")

				// when
				isDirectory, err := testee.IsDirectory(path)

				// then
				require.NoError(t, err)
				require.False(t, isDirectory)
			})
			t.Run("Should report a missing directory when listing", func(t *testing.T) {
				// given
				testee, directory := adapter.new(t)
				path := mustJoin(t, directory, "missing")

				// when
				_, err := testee.ListDirectory(path)

				// then
				require.ErrorIs(t, err, stdfs.ErrNotExist)
			})
			t.Run("Should report a missing path when checking its type", func(t *testing.T) {
				// given
				testee, directory := adapter.new(t)
				path := mustJoin(t, directory, "missing")

				// when
				_, err := testee.IsDirectory(path)

				// then
				require.ErrorIs(t, err, stdfs.ErrNotExist)
			})
			t.Run("Should report a missing file", func(t *testing.T) {
				// given
				testee, directory := adapter.new(t)
				path := mustJoin(t, directory, "missing.txt")

				// when
				_, err := testee.ReadFile(path)

				// then
				require.ErrorIs(t, err, stdfs.ErrNotExist)
			})
			t.Run("Should reject listing a file", func(t *testing.T) {
				// given
				testee, directory := adapter.new(t)
				path := mustJoin(t, directory, "rules.txt")

				// when
				_, err := testee.ListDirectory(path)

				// then
				require.ErrorIs(t, err, ErrNotDirectory)
			})
			t.Run("Should reject reading a directory as a file", func(t *testing.T) {
				// given
				testee, path := adapter.new(t)

				// when
				_, err := testee.ReadFile(path)

				// then
				require.ErrorIs(t, err, ErrIsDirectory)
			})
			t.Run("Should reject a zero-value path in each read operation", func(t *testing.T) {
				// given
				testee, _ := adapter.new(t)
				var path Path

				// when
				_, readErr := testee.ReadFile(path)
				_, listErr := testee.ListDirectory(path)
				_, typeErr := testee.IsDirectory(path)

				// then
				require.ErrorIs(t, readErr, stdfs.ErrInvalid)
				require.ErrorIs(t, listErr, stdfs.ErrInvalid)
				require.ErrorIs(t, typeErr, stdfs.ErrInvalid)
			})
			t.Run("Should keep stored contents independent of returned bytes", func(t *testing.T) {
				// given
				testee, directory := adapter.new(t)
				path := mustJoin(t, directory, "rules.txt")
				data, err := testee.ReadFile(path)
				require.NoError(t, err)

				// when
				data[0] = 'X'
				stored, err := testee.ReadFile(path)

				// then
				require.NoError(t, err)
				require.Equal(t, "example.com:443\n", string(stored))
			})
		})
	}
}

func newMemoryFixture(t *testing.T) (FileSystem, Path) {
	t.Helper()
	return NewMemory(MemoryOptions{Files: map[Path][]byte{
		mustPath(t, "rules/rules.txt"):        []byte("example.com:443\n"),
		mustPath(t, "rules/nested/info.txt"):  []byte("nested"),
		mustPath(t, "rules-other/hidden.txt"): []byte("outside"),
	}}), mustPath(t, "rules")
}

func newOSFixture(t *testing.T) (FileSystem, Path) {
	t.Helper()
	base := t.TempDir()
	path := filepath.Join(base, "rules")
	require.NoError(t, os.MkdirAll(filepath.Join(path, "nested"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(path, "rules.txt"), []byte("example.com:443\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(path, "nested", "info.txt"), []byte("nested"), 0600))
	require.NoError(t, os.Mkdir(filepath.Join(base, "rules-other"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(base, "rules-other", "hidden.txt"), []byte("outside"), 0600))
	return NewOS(), mustPath(t, path)
}

func mustPath(t *testing.T, value string) Path {
	t.Helper()
	path, err := ParsePath(value)
	require.NoError(t, err)
	return path
}

func mustJoin(t *testing.T, path Path, name string) Path {
	t.Helper()
	child, err := path.Join(name)
	require.NoError(t, err)
	return child
}
