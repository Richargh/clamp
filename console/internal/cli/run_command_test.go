package cli

import (
	"context"
	"testing"

	"github.com/richargh/clamp/console/internal/kernel/process"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	t.Run("Should describe the harness as a positional argument", func(t *testing.T) {
		// given
		memory, settings := cliMemory(t)
		testee := New(memory.Env, settings)

		// when
		err := testee.Run(t.Context(), []string{"run", "--help"})

		// then
		require.NoError(t, err)
		require.Contains(t, memory.Output.String(), "clamp run <harness> <folder> [flags]")
		require.NotContains(t, memory.Output.String(), "Available Commands:")
	})
	t.Run("Should reject an unsupported harness value", func(t *testing.T) {
		// given
		testee := New(cliEnv(t))

		// when
		err := testee.Run(t.Context(), []string{"run", "unsupported", "."})

		// then
		require.ErrorContains(t, err, "unsupported harness: unsupported")
	})
	for _, scenario := range []struct {
		name                      string
		beforeFolder, afterFolder []string
	}{
		{name: "Should accept flags before the harness", beforeFolder: []string{"run", "--runtime=container", "--per-project-auth", "pi"}},
		{name: "Should accept flags between the harness and folder", beforeFolder: []string{"run", "pi", "--runtime=container", "--per-project-auth"}},
		{name: "Should accept flags after the folder", beforeFolder: []string{"run", "pi"}, afterFolder: []string{"--runtime=container", "--per-project-auth"}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// given
			memory, settings := cliMemory(t)
			memory.ProcessMemory.OnRun = nil
			testee := New(memory.Env, settings)
			args := append(append([]string{}, scenario.beforeFolder...), "/project")
			args = append(args, scenario.afterFolder...)

			// when
			err := testee.Run(t.Context(), args)

			// then
			require.NoError(t, err)
			for _, call := range memory.ProcessMemory.Calls {
				if len(call.Args) > 0 && call.Args[0] == "run" {
					require.Equal(t, "/bin/container", call.Executable.String())
					return
				}
			}
			t.Fatal("expected a container run command")
		})
	}
	for _, harness := range []string{"pi", "claude", "opencode"} {
		t.Run("Should select "+harness+" through the run command", func(t *testing.T) {
			// given
			memory, settings := cliMemory(t)
			memory.ProcessMemory.OnRun = nil
			testee := New(memory.Env, settings)
			args := []string{"run", harness, "/project", "--per-project-auth"}

			// when
			err := testee.Run(t.Context(), args)

			// then
			require.NoError(t, err)
			for _, call := range memory.ProcessMemory.Calls {
				if len(call.Args) > 0 && call.Args[0] == "run" {
					require.Contains(t, call.Args, "--harness="+harness)
					return
				}
			}
			t.Fatal("expected a container run command")
		})
	}
	t.Run("Should not share parsed flags between command instances", func(t *testing.T) {
		// given
		memory, settings := cliMemory(t)
		require.NoError(t, New(memory.Env, settings).Run(t.Context(), []string{"--version"}))
		memory.Output.Reset()
		testee := New(memory.Env, settings)

		// when
		err := testee.Run(t.Context(), nil)

		// then
		require.NoError(t, err)
		require.Contains(t, memory.Output.String(), "Usage:")
		require.NotContains(t, memory.Output.String(), "Clamp 1.2.3")
	})
	t.Run("Should preserve the container process exit code", func(t *testing.T) {
		// given
		memory, settings := cliMemory(t)
		failure := &processExitError{code: 42}
		memory.ProcessMemory.OnRun = func(_ context.Context, p process.Process) error {
			if len(p.Args) > 0 && p.Args[0] == "run" {
				return failure
			}
			return nil
		}
		testee := New(memory.Env, settings)
		args := []string{"run", "pi", "/project", "--per-project-auth"}

		// when
		err := testee.Run(t.Context(), args)

		// then
		var exit interface{ ExitCode() int }
		require.ErrorAs(t, err, &exit)
		require.Equal(t, 42, exit.ExitCode())
	})
}

type processExitError struct{ code int }

func (*processExitError) Error() string   { return "container process failed" }
func (e *processExitError) ExitCode() int { return e.code }
