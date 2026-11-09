package session

import (
	"testing"

	"github.com/richargh/clamp/console/internal/kernel/environment"
	"github.com/stretchr/testify/require"
)

func TestWorkspacePaths(t *testing.T) {
	t.Run("Should clean the workspace path without host file access", func(t *testing.T) {
		// given
		raw := "/projects/other/../my-app"

		// when
		testee, err := ParseWorkspace(raw)

		// then
		require.NoError(t, err)
		require.Equal(t, "/projects/my-app", testee.String())
	})
	t.Run("Should derive the session log directory from the workspace", func(t *testing.T) {
		// given
		testee, err := ParseWorkspace("/projects/my-app")
		require.NoError(t, err)

		// when
		actual, err := testee.SessionLogs()

		// then
		require.NoError(t, err)
		require.Equal(t, "/projects/my-app/.clamp/sessions", actual.String())
	})
	t.Run("Should reject deriving log paths from an uninitialized workspace", func(t *testing.T) {
		// given
		var testee Workspace

		// when
		_, err := testee.SessionLogs()

		// then
		require.Error(t, err)
	})
	t.Run("Should reject a workspace that is a file", func(t *testing.T) {
		// given
		memory := environment.NewMemory()
		require.NoError(t, memory.Files.WriteFile("/tmp/file", []byte("contents"), 0600))
		testee := New(memory.Env, launchConfig(t))

		// when
		_, err := testee.resolveWorkspace("/tmp/file")

		// then
		require.ErrorContains(t, err, "not a folder: /tmp/file")
	})
	t.Run("Should reject a workspace path that cannot be represented as a bind mount", func(t *testing.T) {
		// given
		memory := environment.NewMemory()
		require.NoError(t, memory.Files.MkdirAll("/project:other", 0700))
		testee := New(memory.Env, launchConfig(t))

		// when
		_, err := testee.resolveWorkspace("/project:other")

		// then
		require.ErrorContains(t, err, "mount path cannot contain colons")
	})
}
