package environment

import (
	"fmt"
	"testing"

	"github.com/richargh/clamp/console/internal/kernel/filesystem"
	"github.com/richargh/clamp/console/internal/kernel/process"
	"github.com/stretchr/testify/require"
)

func TestMemoryEnvironment(t *testing.T) {
	t.Run("Should keep file changes inside each memory environment", func(t *testing.T) {
		// given
		testee := NewMemory()
		other := NewMemory()
		path := "/tmp/memory-only.txt"

		// when
		err := testee.Files.WriteFile(path, []byte("memory only"), 0600)

		// then
		require.NoError(t, err)
		contents, err := testee.Files.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, "memory only", string(contents))
		_, err = other.Files.Stat(path)
		require.ErrorIs(t, err, filesystem.ErrNotExist)
	})
	t.Run("Should supply the memory process adapter through the environment port", func(t *testing.T) {
		// given
		testee := NewMemory()
		executable, err := process.ParseExecutable("not-a-real-program")
		require.NoError(t, err)
		command := process.Process{Executable: executable}

		// when
		err = testee.Processes.Run(t.Context(), command)

		// then
		require.NoError(t, err)
		require.Equal(t, []process.Process{command}, testee.ProcessMemory.Calls)
	})
	t.Run("Should capture output through the environment streams", func(t *testing.T) {
		// given
		testee := NewMemory()

		// when
		_, err := fmt.Fprint(testee.Stdout, "memory output")

		// then
		require.NoError(t, err)
		require.Equal(t, "memory output", testee.Output.String())
	})
	t.Run("Should supply the memory clock through the environment port", func(t *testing.T) {
		// given
		testee := NewMemory()
		testee.ClockMemory.Time = testee.ClockMemory.Time.AddDate(1, 0, 0)

		// when
		actual := testee.Clock.Now()

		// then
		require.Equal(t, testee.ClockMemory.Time, actual)
	})
}
