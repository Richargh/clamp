package session

import (
	"context"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/richargh/clamp/console/internal/config"
	"github.com/richargh/clamp/console/internal/kernel/environment"
	"github.com/richargh/clamp/console/internal/kernel/process"
	"github.com/stretchr/testify/require"
)

func newLaunchEnv(t *testing.T) (environment.Env, string, *process.Memory) {
	t.Helper()
	memory := environment.NewMemory()
	memory.ProcessMemory.OnRun = withProxy(nil)
	assets := launchConfig(t).AssetsDir().String()
	workspace := "/projects/project with spaces"
	require.NoError(t, memory.Files.MkdirAll(workspace, 0700))
	domains := filepath.Join(assets, "clamp-shared", "allowed-domains.d")
	require.NoError(t, memory.Files.MkdirAll(domains, 0700))
	require.NoError(t, memory.Files.WriteFile(filepath.Join(domains, "example.txt"), []byte("example.com\n"), 0600))
	for _, name := range []string{"clamp-base", "clamp-pi", "clamp-claude", "clamp-opencode"} {
		require.NoError(t, memory.Files.WriteFile(filepath.Join(assets, name+".Dockerfile"), []byte("FROM scratch\n"), 0600))
	}
	memory.ClockMemory.Time = time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
	for _, name := range []string{"docker", "container"} {
		executable, err := process.ParseExecutable("/bin/" + name)
		require.NoError(t, err)
		memory.ProcessMemory.Executables[name] = executable
	}
	return memory.Env, workspace, memory.ProcessMemory
}

func withProxy(next func(context.Context, process.Process) error) func(context.Context, process.Process) error {
	return func(ctx context.Context, p process.Process) error {
		if p.Executable.String() == "/clamp/bin/clamp-mitm-proxy" {
			if _, err := io.WriteString(p.Stdout, "{\"schema\":1,\"address\":\"0.0.0.0:18888\"}\n"); err != nil {
				return err
			}
			<-ctx.Done()
			return ctx.Err()
		}
		if next != nil {
			return next(ctx, p)
		}
		return nil
	}
}

func launchConfigInput() config.Raw {
	return config.Raw{AssetsDir: "/clamp", AuthStore: "/config/clamp/auth-projects.tsv"}
}

func launchConfig(t *testing.T) config.Config {
	t.Helper()
	settings, err := config.Parse(launchConfigInput())
	require.NoError(t, err)
	return settings
}

func launchOptions(workspace string) Options {
	harness, _ := ParseHarness("claude")
	runtime, _ := ParseRuntime("docker")
	return Options{Harness: harness, Runtime: runtime, Folder: workspace, PerProjectAuth: true}
}
