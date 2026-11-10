package settings

import (
	"testing"

	"github.com/richargh/clamp/console/internal/kernel/environment"
	"github.com/richargh/clamp/console/internal/kernel/filesystem"
	"github.com/stretchr/testify/require"
)

func newProjectBootstrapEnv(t *testing.T) (environment.Env, string, filesystem.Path) {
	t.Helper()
	memory := environment.NewMemory()
	workspace := "/projects/project with spaces"
	require.NoError(t, memory.Files.MkdirAll(workspace, 0700))
	assets, err := filesystem.ParsePath("/clamp")
	require.NoError(t, err)
	return memory.Env, workspace, assets
}
