package session

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestContainerSession(t *testing.T) {
	t.Run("Should preserve workspace spaces while using safe project volume names", func(t *testing.T) {
		// given
		workspace, err := ParseWorkspace("/projects/project with spaces")
		require.NoError(t, err)
		harness, err := ParseHarness("claude")
		require.NoError(t, err)
		project := newProjectIdentity(harness, workspace)
		now := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
		testee := newContainerSession

		// when
		actual, err := testee(project, ProjectAuthPolicy(), now)

		// then
		require.NoError(t, err)
		require.Equal(t, "/projects/project with spaces", actual.workspace.String())
		require.Equal(t, "project-with-spaces", actual.project.String())
		require.Equal(t, "claude-clamp-project-with-spaces-1648402911", actual.credentialVolume.String())
	})
	t.Run("Should reject a clock value outside the session filename format", func(t *testing.T) {
		// given
		workspace, err := ParseWorkspace("/projects/my-app")
		require.NoError(t, err)
		harness, err := ParseHarness("claude")
		require.NoError(t, err)
		project := newProjectIdentity(harness, workspace)
		now := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)
		testee := newContainerSession

		// when
		_, err = testee(project, ProjectAuthPolicy(), now)

		// then
		require.ErrorContains(t, err, "invalid session log name")
	})
}
