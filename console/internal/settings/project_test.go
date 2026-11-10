package settings

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProjectSettings(t *testing.T) {
	t.Run("Should resolve project files without user settings", func(t *testing.T) {
		// given
		workspace := "/work/project"

		// when
		testee, err := ParseProject(workspace)

		// then
		require.NoError(t, err)
		require.Equal(t, "/work/project/.clamp", testee.Directory())
		require.Equal(t, "/work/project/.clamp/clamp.Dockerfile", testee.Dockerfile())
		require.Equal(t, "/work/project/.clamp/clamp.Dockerfile.dockerignore", testee.Dockerignore())
		require.Equal(t, "/work/project/.clamp/allowed-domains.d", testee.AllowedDomains())
	})
	t.Run("Should reject a relative workspace", func(t *testing.T) {
		// given
		workspace := "project"

		// when
		_, err := ParseProject(workspace)

		// then
		require.ErrorContains(t, err, "project workspace")
	})
}
