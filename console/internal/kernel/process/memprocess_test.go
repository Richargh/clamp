package process

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMemoryProcesses(t *testing.T) {
	t.Run("Should record a process without starting a program", func(t *testing.T) {
		// given
		testee := NewMemory()
		executable, err := ParseExecutable("not-a-real-program")
		require.NoError(t, err)
		process := Process{Executable: executable, Args: []string{"argument"}}

		// when
		err = testee.Run(t.Context(), process)

		// then
		require.NoError(t, err)
		require.Equal(t, []Process{process}, testee.Calls)
	})
	t.Run("Should retain arguments after the caller changes its slice", func(t *testing.T) {
		// given
		testee := NewMemory()
		executable, err := ParseExecutable("program")
		require.NoError(t, err)
		arguments := []string{"original"}
		require.NoError(t, testee.Run(t.Context(), Process{Executable: executable, Args: arguments}))

		// when
		arguments[0] = "changed"

		// then
		require.Equal(t, []string{"original"}, testee.Calls[0].Args)
	})
	t.Run("Should return a simulated process error", func(t *testing.T) {
		// given
		testee := NewMemory()
		executable, err := ParseExecutable("program")
		require.NoError(t, err)
		failure := errors.New("simulated exit")
		testee.OnRun = func(context.Context, Process) error { return failure }

		// when
		err = testee.Run(t.Context(), Process{Executable: executable})

		// then
		require.ErrorIs(t, err, failure)
	})
	t.Run("Should reject a canceled process before recording it", func(t *testing.T) {
		// given
		testee := NewMemory()
		executable, err := ParseExecutable("program")
		require.NoError(t, err)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		// when
		err = testee.Run(ctx, Process{Executable: executable})

		// then
		require.ErrorIs(t, err, context.Canceled)
		require.Empty(t, testee.Calls)
	})
	t.Run("Should capture simulated process output", func(t *testing.T) {
		// given
		testee := NewMemory()
		executable, err := ParseExecutable("program")
		require.NoError(t, err)
		var output bytes.Buffer
		testee.OnRun = func(_ context.Context, process Process) error {
			_, err := fmt.Fprint(process.Stdout, "simulated output")
			return err
		}

		// when
		err = testee.Run(t.Context(), Process{Executable: executable, Stdout: &output})

		// then
		require.NoError(t, err)
		require.Equal(t, "simulated output", output.String())
	})
	t.Run("Should reject an uninitialized executable", func(t *testing.T) {
		// given
		testee := NewMemory()

		// when
		err := testee.Run(t.Context(), Process{})

		// then
		require.ErrorContains(t, err, "invalid executable")
		require.Empty(t, testee.Calls)
	})
}

func TestMemoryExecutableLookup(t *testing.T) {
	t.Run("Should return a configured executable", func(t *testing.T) {
		// given
		testee := NewMemory()
		executable, err := ParseExecutable("/virtual/bin/docker")
		require.NoError(t, err)
		testee.Executables["docker"] = executable

		// when
		actual, err := testee.LookPath("docker")

		// then
		require.NoError(t, err)
		require.Equal(t, executable, actual)
	})
	t.Run("Should reject an executable that is not configured", func(t *testing.T) {
		// given
		testee := NewMemory()

		// when
		_, err := testee.LookPath("sh")

		// then
		require.ErrorContains(t, err, "executable not found in memory: sh")
	})
}

func TestExecutableParsing(t *testing.T) {
	for _, raw := range []string{"", "program\x00argument"} {
		t.Run(fmt.Sprintf("Should reject invalid executable %q", raw), func(t *testing.T) {
			// given
			// when
			testee, err := ParseExecutable(raw)

			// then
			require.Error(t, err)
			require.Empty(t, testee.String())
		})
	}
}
