package cli

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/richargh/clamp/console/internal/config"
	"github.com/richargh/clamp/console/internal/kernel/environment"
	"github.com/richargh/clamp/console/internal/kernel/process"
	"github.com/stretchr/testify/require"
)

func cliEnv(t *testing.T) (environment.Env, config.Config) {
	t.Helper()
	memory, settings := cliMemory(t)
	return memory.Env, settings
}

func cliMemory(t *testing.T) (*environment.Memory, config.Config) {
	t.Helper()
	settings, err := config.Parse(config.Raw{AssetsDir: "/clamp", AuthStore: "/config/clamp/auth-projects.tsv"})
	require.NoError(t, err)
	memory := environment.NewMemory()
	assets := settings.AssetsDir().String()
	require.NoError(t, memory.Files.MkdirAll(assets, 0700))
	require.NoError(t, memory.Files.MkdirAll("/project", 0700))
	for _, name := range []string{"clamp-base.Dockerfile", "clamp-pi.Dockerfile", "clamp-claude.Dockerfile", "clamp-opencode.Dockerfile", "VERSION"} {
		require.NoError(t, memory.Files.WriteFile(filepath.Join(assets, name), []byte("1.2.3\n"), 0600))
	}
	for _, name := range []string{"docker", "container"} {
		executable, err := process.ParseExecutable("/bin/" + name)
		require.NoError(t, err)
		memory.ProcessMemory.Executables[name] = executable
	}
	memory.ProcessMemory.OnRun = func(context.Context, process.Process) error {
		t.Fatal("unexpected process execution")
		return nil
	}
	return memory, settings
}
