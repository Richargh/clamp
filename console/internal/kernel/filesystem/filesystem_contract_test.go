package filesystem

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFileSystemContract(t *testing.T) {
	for _, adapter := range []struct {
		name string
		new  func(*testing.T) (FileSystem, string)
	}{
		{"memory", newMemoryFixture},
		{"OS", newOSFixture},
	} {
		t.Run(adapter.name, func(t *testing.T) {
			t.Run("Should read file contents", func(t *testing.T) {
				// given
				testee, root := adapter.new(t)
				path := filepath.Join(root, "rules.txt")
				require.NoError(t, testee.WriteFile(path, []byte("example.com:443"), 0600))

				// when
				contents, err := testee.ReadFile(path)

				// then
				require.NoError(t, err)
				require.Equal(t, "example.com:443", string(contents))
			})
			t.Run("Should copy contents supplied to WriteFile", func(t *testing.T) {
				// given
				testee, root := adapter.new(t)
				path := filepath.Join(root, "rules.txt")
				contents := []byte("example.com:443")
				require.NoError(t, testee.WriteFile(path, contents, 0600))

				// when
				contents[0] = 'X'
				stored, err := testee.ReadFile(path)

				// then
				require.NoError(t, err)
				require.Equal(t, "example.com:443", string(stored))
			})
			t.Run("Should copy contents returned by ReadFile", func(t *testing.T) {
				// given
				testee, root := adapter.new(t)
				path := filepath.Join(root, "rules.txt")
				require.NoError(t, testee.WriteFile(path, []byte("example.com:443"), 0600))
				contents, err := testee.ReadFile(path)
				require.NoError(t, err)

				// when
				contents[0] = 'X'
				stored, err := testee.ReadFile(path)

				// then
				require.NoError(t, err)
				require.Equal(t, "example.com:443", string(stored))
			})
			t.Run("Should preserve an existing file mode when overwriting", func(t *testing.T) {
				// given
				testee, root := adapter.new(t)
				path := filepath.Join(root, "rules.txt")
				require.NoError(t, testee.WriteFile(path, []byte("old"), 0600))

				// when
				require.NoError(t, testee.WriteFile(path, []byte("new"), 0644))
				info, err := testee.Stat(path)

				// then
				require.NoError(t, err)
				require.Equal(t, Mode(0600), info.Mode())
			})
			t.Run("Should create empty directories and their parents", func(t *testing.T) {
				// given
				testee, root := adapter.new(t)
				path := filepath.Join(root, "nested", "empty")

				// when
				require.NoError(t, testee.MkdirAll(path, 0700))
				info, err := testee.Stat(path)

				// then
				require.NoError(t, err)
				require.True(t, info.IsDir())
			})
			t.Run("Should report a missing file", func(t *testing.T) {
				// given
				testee, root := adapter.new(t)

				// when
				_, err := testee.ReadFile(filepath.Join(root, "missing"))

				// then
				require.ErrorIs(t, err, ErrNotExist)
			})
			t.Run("Should reject writing without a parent directory", func(t *testing.T) {
				// given
				testee, root := adapter.new(t)

				// when
				err := testee.WriteFile(filepath.Join(root, "missing", "file"), nil, 0600)

				// then
				require.ErrorIs(t, err, ErrNotExist)
			})
			t.Run("Should reject creating a directory below a file", func(t *testing.T) {
				// given
				testee, root := adapter.new(t)
				path := filepath.Join(root, "file")
				require.NoError(t, testee.WriteFile(path, nil, 0600))

				// when
				err := testee.MkdirAll(filepath.Join(path, "child"), 0700)

				// then
				require.ErrorIs(t, err, ErrNotDirectory)
			})
			t.Run("Should reject reading a directory as a file", func(t *testing.T) {
				// given
				testee, root := adapter.new(t)

				// when
				_, err := testee.ReadFile(root)

				// then
				require.ErrorIs(t, err, ErrIsDirectory)
			})
			t.Run("Should remove a file", func(t *testing.T) {
				// given
				testee, root := adapter.new(t)
				path := filepath.Join(root, "file")
				require.NoError(t, testee.WriteFile(path, nil, 0600))

				// when
				require.NoError(t, testee.Remove(path))
				_, err := testee.Stat(path)

				// then
				require.ErrorIs(t, err, ErrNotExist)
			})
			t.Run("Should reject removing a nonempty directory", func(t *testing.T) {
				// given
				testee, root := adapter.new(t)
				require.NoError(t, testee.WriteFile(filepath.Join(root, "file"), nil, 0600))

				// when
				err := testee.Remove(root)

				// then
				require.ErrorIs(t, err, ErrNotEmpty)
			})
			t.Run("Should replace an existing file by renaming", func(t *testing.T) {
				// given
				testee, root := adapter.new(t)
				source, target := filepath.Join(root, "new"), filepath.Join(root, "current")
				require.NoError(t, testee.WriteFile(source, []byte("new contents"), 0600))
				require.NoError(t, testee.WriteFile(target, []byte("old contents"), 0600))

				// when
				require.NoError(t, testee.Rename(source, target))
				contents, err := testee.ReadFile(target)

				// then
				require.NoError(t, err)
				require.Equal(t, "new contents", string(contents))
			})
			t.Run("Should remove the old name after renaming", func(t *testing.T) {
				// given
				testee, root := adapter.new(t)
				source := filepath.Join(root, "old")
				require.NoError(t, testee.WriteFile(source, nil, 0600))

				// when
				require.NoError(t, testee.Rename(source, filepath.Join(root, "new")))
				_, err := testee.Stat(source)

				// then
				require.ErrorIs(t, err, ErrNotExist)
			})
			t.Run("Should rename a directory with its contents", func(t *testing.T) {
				// given
				testee, root := adapter.new(t)
				source, target := filepath.Join(root, "old"), filepath.Join(root, "new")
				require.NoError(t, testee.MkdirAll(filepath.Join(source, "nested"), 0700))
				require.NoError(t, testee.WriteFile(filepath.Join(source, "nested", "file"), []byte("contents"), 0600))

				// when
				require.NoError(t, testee.Rename(source, target))
				contents, err := testee.ReadFile(filepath.Join(target, "nested", "file"))

				// then
				require.NoError(t, err)
				require.Equal(t, "contents", string(contents))
			})
			t.Run("Should copy nested directory contents", func(t *testing.T) {
				// given
				testee, root := adapter.new(t)
				source, target := filepath.Join(root, "source"), filepath.Join(root, "target")
				require.NoError(t, testee.MkdirAll(filepath.Join(source, "nested"), 0700))
				require.NoError(t, testee.WriteFile(filepath.Join(source, "nested", "file"), []byte("contents"), 0600))

				// when
				require.NoError(t, testee.CopyDirectory(source, target))
				contents, err := testee.ReadFile(filepath.Join(target, "nested", "file"))

				// then
				require.NoError(t, err)
				require.Equal(t, "contents", string(contents))
			})
			t.Run("Should copy empty directories", func(t *testing.T) {
				// given
				testee, root := adapter.new(t)
				source, target := filepath.Join(root, "source"), filepath.Join(root, "target")
				require.NoError(t, testee.MkdirAll(filepath.Join(source, "empty"), 0700))

				// when
				require.NoError(t, testee.CopyDirectory(source, target))
				info, err := testee.Stat(filepath.Join(target, "empty"))

				// then
				require.NoError(t, err)
				require.True(t, info.IsDir())
			})
			t.Run("Should refuse to overwrite files during a directory copy", func(t *testing.T) {
				// given
				testee, root := adapter.new(t)
				source, target := filepath.Join(root, "source"), filepath.Join(root, "target")
				require.NoError(t, testee.MkdirAll(source, 0700))
				require.NoError(t, testee.MkdirAll(target, 0700))
				require.NoError(t, testee.WriteFile(filepath.Join(source, "file"), []byte("new"), 0600))
				require.NoError(t, testee.WriteFile(filepath.Join(target, "file"), []byte("existing"), 0600))

				// when
				err := testee.CopyDirectory(source, target)

				// then
				require.ErrorIs(t, err, ErrExist)
			})
			t.Run("Should resolve an existing path", func(t *testing.T) {
				// given
				testee, root := adapter.new(t)
				path := filepath.Join(root, "file")
				require.NoError(t, testee.WriteFile(path, nil, 0600))

				// when
				resolved, err := testee.Resolve(root + string(filepath.Separator) + "." + string(filepath.Separator) + "file")

				// then
				require.NoError(t, err)
				require.Equal(t, path, resolved)
			})
			t.Run("Should reject resolving a missing path", func(t *testing.T) {
				// given
				testee, root := adapter.new(t)

				// when
				_, err := testee.Resolve(filepath.Join(root, "missing"))

				// then
				require.ErrorIs(t, err, ErrNotExist)
			})
			t.Run("Should create unique temporary files matching the pattern", func(t *testing.T) {
				// given
				testee, root := adapter.new(t)
				first, err := testee.TempFile(root, "auth-*.tmp")
				require.NoError(t, err)

				// when
				second, err := testee.TempFile(root, "auth-*.tmp")

				// then
				require.NoError(t, err)
				require.NotEqual(t, first, second)
				require.Equal(t, root, filepath.Dir(second))
				require.True(t, strings.HasPrefix(filepath.Base(second), "auth-"))
				require.True(t, strings.HasSuffix(second, ".tmp"))
			})
			t.Run("Should create a private temporary file", func(t *testing.T) {
				// given
				testee, root := adapter.new(t)

				// when
				path, err := testee.TempFile(root, "auth-*")
				require.NoError(t, err)
				info, err := testee.Stat(path)

				// then
				require.NoError(t, err)
				require.Equal(t, Mode(0600), info.Mode())
			})
		})
	}
}

func newMemoryFixture(t *testing.T) (FileSystem, string) {
	t.Helper()
	files := NewMemory()
	root := filepath.Join(string(filepath.Separator), "contract")
	require.NoError(t, files.MkdirAll(root, 0700))
	return files, root
}

func newOSFixture(t *testing.T) (FileSystem, string) {
	t.Helper()
	// macOS temporary directories can be below /var, which links to /private/var.
	// Use a canonical fixture root without relying on the Resolve method under test.
	root, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	return NewOS(), root
}
