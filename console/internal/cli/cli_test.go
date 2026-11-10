package cli

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCLI(t *testing.T) {
	for _, args := range [][]string{{"run", "pi", "--help"}, {"run", "claude", "--help"}, {"run", "opencode", "--help"}, {"run", "--help"}, {"--help"}} {
		t.Run("Should show help without a runtime for "+strings.Join(args, " "), func(t *testing.T) {
			// given
			memory, settings := cliMemory(t)
			testee := New(memory.Env, settings)

			// when
			err := testee.Run(t.Context(), args)

			// then
			require.NoError(t, err)
			require.Contains(t, memory.Output.String(), "Usage:")
		})
	}
	for _, args := range [][]string{{"--version"}, {"-v"}} {
		t.Run("Should show the version without a folder for "+strings.Join(args, " "), func(t *testing.T) {
			// given
			memory, settings := cliMemory(t)
			testee := New(memory.Env, settings)

			// when
			err := testee.Run(t.Context(), args)

			// then
			require.NoError(t, err)
			require.Contains(t, memory.Output.String(), "Clamp 1.2.3")
		})
	}
	for _, args := range [][]string{{"run"}, {"run", "pi"}, {"run", "claude", "one", "two"}, {"run", "opencode", "--unknown", "."}, {"run", "pi", "--runtime"}, {"run", "pi", "--runtime=invalid", "."}, {"run", "invalid", "."}, {"invalid", "."}, {"pi", "."}, {"claude", "."}, {"opencode", "."}} {
		t.Run("Should reject invalid arguments "+strings.Join(args, " "), func(t *testing.T) {
			// given
			testee := New(cliEnv(t))

			// when
			err := testee.Run(t.Context(), args)

			// then
			require.Error(t, err)
		})
	}
	t.Run("Should accept all legacy flags after the folder", func(t *testing.T) {
		// given
		memory, settings := cliMemory(t)
		memory.ProcessMemory.OnRun = withProxy(nil)
		testee := New(memory.Env, settings)
		args := []string{"run", "pi", "/project", "--runtime=container", "--rebuild", "--no-cache", "--per-project-auth", "--no-firewall", "--danger", "--shell"}

		// when
		err := testee.Run(t.Context(), args)

		// then
		require.NoError(t, err)
		calls := memory.ProcessMemory.Calls
		require.Len(t, calls, 6)
		for _, p := range calls[:3] {
			require.Equal(t, "/bin/container", p.Executable.String())
			require.Contains(t, p.Args, "--no-cache")
		}
		run := calls[4]
		require.Equal(t, "run", run.Args[0])
		require.Equal(t, "bash", run.Args[len(run.Args)-1])
		require.NotContains(t, run.Args, "--no-firewall")
	})
}
