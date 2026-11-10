package filesystem

import (
	"io"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAppendFileContract(t *testing.T) {
	for _, adapter := range []struct {
		name string
		new  func(*testing.T) (FileSystem, string)
	}{
		{"memory", newMemoryFixture}, {"OS", newOSFixture},
	} {
		t.Run(adapter.name, func(t *testing.T) {
			t.Run("Should create a log file", func(t *testing.T) {
				// given
				testee, root := adapter.new(t)
				path := filepath.Join(root, "network.log")
				// when
				writer, err := testee.OpenAppend(path, 0600)
				require.NoError(t, err)
				_, err = io.WriteString(writer, "entry\n")
				require.NoError(t, err)
				require.NoError(t, writer.Close())
				// then
				data, err := testee.ReadFile(path)
				require.NoError(t, err)
				require.Equal(t, "entry\n", string(data))
				info, err := testee.Stat(path)
				require.NoError(t, err)
				require.Equal(t, Mode(0600), info.Mode())
			})
			t.Run("Should append without truncating an existing log", func(t *testing.T) {
				// given
				testee, root := adapter.new(t)
				path := filepath.Join(root, "network.log")
				require.NoError(t, testee.WriteFile(path, []byte("old\n"), 0600))
				// when
				writer, err := testee.OpenAppend(path, 0644)
				require.NoError(t, err)
				_, err = io.WriteString(writer, "new\n")
				require.NoError(t, err)
				require.NoError(t, writer.Close())
				// then
				data, err := testee.ReadFile(path)
				require.NoError(t, err)
				require.Equal(t, "old\nnew\n", string(data))
			})
			t.Run("Should support concurrent writes through one log handle", func(t *testing.T) {
				// given
				testee, root := adapter.new(t)
				path := filepath.Join(root, "network.log")
				writer, err := testee.OpenAppend(path, 0600)
				require.NoError(t, err)
				var group sync.WaitGroup
				failures := make(chan error, 10)
				// when
				for range 10 {
					group.Go(func() { _, err := io.WriteString(writer, "entry\n"); failures <- err })
				}
				group.Wait()
				require.NoError(t, writer.Close())
				// then
				for range 10 {
					require.NoError(t, <-failures)
				}
				data, err := testee.ReadFile(path)
				require.NoError(t, err)
				require.Equal(t, strings.Repeat("entry\n", 10), string(data))
			})
			t.Run("Should reject a directory as a log file", func(t *testing.T) {
				// given
				testee, root := adapter.new(t)
				// when
				_, err := testee.OpenAppend(root, 0600)
				// then
				require.ErrorIs(t, err, ErrIsDirectory)
			})
		})
	}
}
