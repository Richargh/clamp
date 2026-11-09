package session

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/richargh/clamp/console/internal/kernel/filesystem"
	"github.com/richargh/clamp/console/internal/kernel/process"
	"github.com/stretchr/testify/require"
)

func TestImageBuild(t *testing.T) {
	t.Run("Should always build the project with a restricted context and remove generated files", func(t *testing.T) {
		// given
		env, workspace, processes := newLaunchEnv(t)
		require.NoError(t, env.Files.WriteFile(filepath.Join(workspace, "mise.toml"), []byte("[tools]\n"), 0600))
		var generated, ignore string
		processes.OnRun = func(_ context.Context, p process.Process) error {
			if p.Args[0] == "build" && p.Args[len(p.Args)-1] == workspace {
				for i, arg := range p.Args {
					if arg == "-f" {
						generated = p.Args[i+1]
					}
				}
				data, err := env.Files.ReadFile(generated)
				require.NoError(t, err)
				require.Contains(t, string(data), "FROM ${CLAMP_TOOL_IMAGE}")
				require.Contains(t, string(data), "RUN mise install -C /tmp")
				data, err = env.Files.ReadFile(generated + ".dockerignore")
				require.NoError(t, err)
				ignore = string(data)
			}
			return nil
		}
		testee := New(env, launchConfig(t))

		// when
		err := testee.Launch(t.Context(), launchOptions(workspace))

		// then
		require.NoError(t, err)
		require.Contains(t, ignore, "*\n!.clamp/\n!.clamp/**\n")
		require.Contains(t, ignore, "!mise.toml\n")
		_, err = env.Files.Stat(generated)
		require.ErrorIs(t, err, filesystem.ErrNotExist)
		_, err = env.Files.Stat(generated + ".dockerignore")
		require.ErrorIs(t, err, filesystem.ErrNotExist)
		require.Equal(t, "image", processes.Calls[0].Args[0])
		require.Equal(t, "image", processes.Calls[1].Args[0])
		require.Equal(t, "build", processes.Calls[2].Args[0])
		data, err := env.Files.ReadFile(filepath.Join(workspace, ".clamp", "allowed-domains.d", "example.txt"))
		require.NoError(t, err)
		require.Equal(t, "example.com\n", string(data))
	})
	t.Run("Should rebuild all images without cache and preserve a legacy Dockerfile", func(t *testing.T) {
		// given
		env, workspace, processes := newLaunchEnv(t)
		dir := filepath.Join(workspace, ".clamp")
		require.NoError(t, env.Files.MkdirAll(dir, 0700))
		path := filepath.Join(dir, "clamp.Dockerfile")
		legacy := "FROM custom\nRUN true\n"
		require.NoError(t, env.Files.WriteFile(path, []byte(legacy), 0600))
		options := launchOptions(workspace)
		options.NoCache = true
		testee := New(env, launchConfig(t))

		// when
		err := testee.Launch(t.Context(), options)

		// then
		require.NoError(t, err)
		require.Len(t, processes.Calls, 5)
		for _, p := range processes.Calls[:3] {
			require.Equal(t, "build", p.Args[0])
			require.Contains(t, p.Args, "--no-cache")
		}
		require.Contains(t, processes.Calls[2].Args, path)
		data, err := env.Files.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, legacy, string(data))
	})
	t.Run("Should stop before launch when an image build fails", func(t *testing.T) {
		// given
		env, workspace, processes := newLaunchEnv(t)
		failure := errors.New("build failed")
		processes.OnRun = func(context.Context, process.Process) error { return failure }
		testee := New(env, launchConfig(t))

		// when
		err := testee.Launch(t.Context(), launchOptions(workspace))

		// then
		require.ErrorIs(t, err, failure)
		require.Len(t, processes.Calls, 2)
		require.Equal(t, "build", processes.Calls[1].Args[0])
	})
}

func TestAuthPolicy(t *testing.T) {
	t.Run("Should save the selected policy and reuse it without a prompt", func(t *testing.T) {
		// given
		env, workspace, _ := newLaunchEnv(t)
		env.Stdin = strings.NewReader("invalid\n1\n")
		options := launchOptions(workspace)
		options.PerProjectAuth = false
		settings := launchConfig(t)
		testee := New(env, settings)

		// when
		first := testee.Launch(t.Context(), options)
		env.Stdin = strings.NewReader("")
		testee = New(env, settings)
		second := testee.Launch(t.Context(), options)

		// then
		require.NoError(t, first)
		require.NoError(t, second)
		data, err := env.Files.ReadFile(settings.AuthStore().String())
		require.NoError(t, err)
		require.Equal(t, workspace+"\tglobal\n", string(data))
	})
	t.Run("Should fail without starting containers when the auth prompt reaches EOF", func(t *testing.T) {
		// given
		env, workspace, processes := newLaunchEnv(t)
		env.Stdin = strings.NewReader("")
		options := launchOptions(workspace)
		options.PerProjectAuth = false
		testee := New(env, launchConfig(t))

		// when
		err := testee.Launch(t.Context(), options)

		// then
		require.ErrorContains(t, err, "unable to read auth preference")
		require.Empty(t, processes.Calls)
	})
}

func TestContainerExit(t *testing.T) {
	t.Run("Should keep the container error when the shutdown script also fails", func(t *testing.T) {
		// given
		env, workspace, processes := newLaunchEnv(t)
		settings := launchConfig(t)
		failure := errors.New("agent failed")
		var shutdown process.Process
		processes.OnRun = func(_ context.Context, p process.Process) error {
			if p.Executable.String() == filepath.Join(settings.AssetsDir().String(), "shutdown-scripts", "clamp-shutdown.sh") {
				shutdown = p
				return errors.New("summary failed")
			}
			if p.Args[0] == "run" {
				return failure
			}
			return nil
		}
		options := launchOptions(workspace)
		options.Shell = true
		options.Danger = true
		testee := New(env, settings)

		// when
		err := testee.Launch(t.Context(), options)

		// then
		require.ErrorIs(t, err, failure)
		require.Equal(t, []string{workspace, "20261027-120000-network.log"}, shutdown.Args)
	})
}

func TestProjectChecksum(t *testing.T) {
	t.Run("Should match the POSIX checksum used by shell launchers", func(t *testing.T) {
		// given
		workspace := []byte("/workspace")

		// when
		checksum := projectChecksum(workspace)

		// then
		require.Equal(t, uint32(306203429), checksum)
	})
}
