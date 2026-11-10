package session

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProjectIdentity(t *testing.T) {
	t.Run("Should derive shell-compatible image names from the workspace", func(t *testing.T) {
		// given
		workspace, err := ParseWorkspace("/workspace")
		require.NoError(t, err)
		harness, err := ParseHarness("claude")
		require.NoError(t, err)
		testee := newProjectIdentity

		// when
		actual := testee(harness, workspace)

		// then
		require.Equal(t, workspace, actual.workspace)
		require.Equal(t, "clamp-claude", actual.toolImage.String())
		require.Equal(t, "clamp-claude-306203429", actual.image.String())
	})
	t.Run("Should distinguish projects with the same folder name", func(t *testing.T) {
		// given
		first, err := ParseWorkspace("/first/my-app")
		require.NoError(t, err)
		second, err := ParseWorkspace("/second/my-app")
		require.NoError(t, err)
		harness, err := ParseHarness("pi")
		require.NoError(t, err)
		testee := newProjectIdentity

		// when
		firstProject := testee(harness, first)
		secondProject := testee(harness, second)

		// then
		require.NotEqual(t, firstProject.image, secondProject.image)
	})
}
