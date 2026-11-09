package session

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProjectSetup(t *testing.T) {
	for _, scenario := range []struct {
		name       string
		dockerfile string
		warn       bool
	}{
		{name: "without a mise install", dockerfile: "RUN echo custom\n", warn: true},
		{name: "with only a commented mise example", dockerfile: projectTemplate, warn: true},
		{name: "with a mise install", dockerfile: "RUN mise install -C /tmp\n", warn: false},
	} {
		t.Run("Should report mise setup correctly for a Dockerfile "+scenario.name, func(t *testing.T) {
			// given
			env, workspace, _ := newLaunchEnv(t)
			var output bytes.Buffer
			env.Stdout = &output
			require.NoError(t, env.Files.WriteFile(filepath.Join(workspace, "mise.toml"), []byte("[tools]\n"), 0600))
			dir := filepath.Join(workspace, ".clamp")
			require.NoError(t, env.Files.MkdirAll(dir, 0700))
			require.NoError(t, env.Files.WriteFile(filepath.Join(dir, "clamp.Dockerfile"), []byte(scenario.dockerfile), 0600))
			resolved, err := ParseWorkspace(workspace)
			require.NoError(t, err)
			testee := New(env, launchConfig(t))

			// when
			err = testee.setupProject(resolved)

			// then
			require.NoError(t, err)
			if scenario.warn {
				require.Contains(t, output.String(), "mise.toml exists, but this Dockerfile does not install mise tools.")
			} else {
				require.NotContains(t, output.String(), "does not install mise tools")
			}
		})
	}
	t.Run("Should preserve existing project additions", func(t *testing.T) {
		// given
		env, workspace, _ := newLaunchEnv(t)
		dir := filepath.Join(workspace, ".clamp")
		require.NoError(t, env.Files.MkdirAll(dir, 0700))
		files := map[string]string{
			"clamp.Dockerfile":              "RUN echo custom\n",
			"clamp.Dockerfile.dockerignore": "!custom.txt\n",
		}
		for name, content := range files {
			require.NoError(t, env.Files.WriteFile(filepath.Join(dir, name), []byte(content), 0600))
		}
		resolved, err := ParseWorkspace(workspace)
		require.NoError(t, err)
		testee := New(env, launchConfig(t))

		// when
		err = testee.setupProject(resolved)

		// then
		require.NoError(t, err)
		for name, expected := range files {
			actual, err := env.Files.ReadFile(filepath.Join(dir, name))
			require.NoError(t, err)
			require.Equal(t, expected, string(actual), name)
		}
	})
}
