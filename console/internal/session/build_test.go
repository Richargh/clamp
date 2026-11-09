package session

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/richargh/clamp/console/internal/kernel/filesystem"
	"github.com/richargh/clamp/console/internal/kernel/process"
	"github.com/stretchr/testify/require"
)

func TestProjectImageBuild(t *testing.T) {
	t.Run("Should explain why the project image is built", func(t *testing.T) {
		// given
		env, workspace, _ := newLaunchEnv(t)
		var output bytes.Buffer
		env.Stdout = &output
		testee := New(env, launchConfig(t))

		// when
		err := testee.Launch(t.Context(), launchOptions(workspace))

		// then
		require.NoError(t, err)
		require.Contains(t, output.String(), "Why: Clamp builds a per-project image so project Dockerfile changes, allowed-domain files, and tool setup are applied before the agent starts.")
	})
	t.Run("Should remove generated build files when the project build fails", func(t *testing.T) {
		// given
		env, workspace, processes := newLaunchEnv(t)
		failure := errors.New("project build failed")
		var generated string
		processes.OnRun = func(_ context.Context, p process.Process) error {
			if p.Args[0] == "build" && p.Args[len(p.Args)-1] == workspace {
				for i, arg := range p.Args {
					if arg == "-f" {
						generated = p.Args[i+1]
					}
				}
				return failure
			}
			return nil
		}
		testee := New(env, launchConfig(t))

		// when
		err := testee.Launch(t.Context(), launchOptions(workspace))

		// then
		require.ErrorIs(t, err, failure)
		require.NotEmpty(t, generated)
		for _, path := range []string{generated, generated + ".dockerignore"} {
			_, err := env.Files.Stat(path)
			require.ErrorIs(t, err, filesystem.ErrNotExist, path)
		}
	})
	t.Run("Should create project files before building images", func(t *testing.T) {
		// given
		env, workspace, processes := newLaunchEnv(t)
		failure := errors.New("base build failed")
		processes.OnRun = func(context.Context, process.Process) error { return failure }
		testee := New(env, launchConfig(t))

		// when
		err := testee.Launch(t.Context(), launchOptions(workspace))

		// then
		require.ErrorIs(t, err, failure)
		_, err = env.Files.Stat(filepath.Join(workspace, ".clamp", "clamp.Dockerfile"))
		require.NoError(t, err)
	})
}
