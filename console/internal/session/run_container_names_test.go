package session

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContainerSessionValueParsing(t *testing.T) {
	for _, kind := range []struct {
		name    string
		parse   func(string) (fmt.Stringer, error)
		valid   []string
		invalid []string
	}{
		{
			name:    "container name",
			parse:   func(raw string) (fmt.Stringer, error) { return ParseContainerName(raw) },
			valid:   []string{"pi-20261027-120000", "container_1.2"},
			invalid: []string{"", "a", "-container", "container name", "container/name", "container\x00name"},
		},
		{
			name:    "workspace",
			parse:   func(raw string) (fmt.Stringer, error) { return ParseWorkspace(raw) },
			valid:   []string{"/projects/my-app", "/projects/project with spaces", "/"},
			invalid: []string{"", "relative", "/project:other", "/project\tname", "/project\nname", "/project\x00name"},
		},
		{
			name:    "project name",
			parse:   func(raw string) (fmt.Stringer, error) { return ParseProjectName(raw) },
			valid:   []string{"my-app", "a", "Project_1.2"},
			invalid: []string{"", ".", "..", "-project", "project name", "project/name", "project\x00name"},
		},
		{
			name:    "local image name",
			parse:   func(raw string) (fmt.Stringer, error) { return ParseImageName(raw) },
			valid:   []string{"clamp-pi-1234", "clamp__pi", "clamp.pi", "clamp--pi", strings.Repeat("a", 255)},
			invalid: []string{"", "Clamp", "-clamp", "clamp-", "clamp pi", "clamp:latest", "registry/clamp", strings.Repeat("a", 256)},
		},
		{
			name:    "session log directory",
			parse:   func(raw string) (fmt.Stringer, error) { return ParseSessionLogDirectory(raw) },
			valid:   []string{"/project/.clamp/sessions", "/project with spaces/.clamp/sessions"},
			invalid: []string{"", "relative", "/logs:other", "/logs\rname", "/logs\x00name"},
		},
		{
			name:    "session log filename",
			parse:   func(raw string) (fmt.Stringer, error) { return ParseSessionLogName(raw) },
			valid:   []string{"20261027-120000-network.log", "20240229-235959-network.log"},
			invalid: []string{"", "../20261027-120000-network.log", "20260229-120000-network.log", "20261027-240000-network.log", "20261027-120000.log"},
		},
		{
			name:    "volume name",
			parse:   func(raw string) (fmt.Stringer, error) { return ParseVolumeName(raw) },
			valid:   []string{"pi", "pi-my-app-1234", "Project_1.2"},
			invalid: []string{"", "/host/path", "-volume", "volume name", "volume:other", "volume\x00name"},
		},
	} {
		t.Run(kind.name, func(t *testing.T) {
			for _, raw := range kind.valid {
				t.Run(fmt.Sprintf("Should parse %q", raw), func(t *testing.T) {
					// given
					testee := kind.parse

					// when
					value, err := testee(raw)

					// then
					require.NoError(t, err)
					require.Equal(t, raw, value.String())
				})
			}
			for _, raw := range kind.invalid {
				t.Run(fmt.Sprintf("Should reject %q", raw), func(t *testing.T) {
					// given
					testee := kind.parse

					// when
					_, err := testee(raw)

					// then
					require.Error(t, err)
				})
			}
		})
	}
}

func TestProjectVolumeNames(t *testing.T) {
	for _, scenario := range []struct {
		workspace string
		project   string
	}{
		{workspace: "/projects/my-app", project: "my-app"},
		{workspace: "/projects/project with spaces", project: "project-with-spaces"},
		{workspace: "/projects/.hidden", project: "hidden"},
		{workspace: "/projects/项目", project: "project"},
		{workspace: "/", project: "project"},
	} {
		t.Run("Should derive a safe project name from "+scenario.workspace, func(t *testing.T) {
			// given
			workspace, err := ParseWorkspace(scenario.workspace)
			require.NoError(t, err)

			// when
			testee := ProjectNameFor(workspace)

			// then
			require.Equal(t, scenario.project, testee.String())
		})
	}
	t.Run("Should construct the project cache volume names", func(t *testing.T) {
		// given
		testee, err := ParseProjectName("my-app")
		require.NoError(t, err)

		// when
		actual := []VolumeName{testee.NodeModulesVolume(), testee.GradleBuildVolume(), testee.GradleCacheVolume()}

		// then
		require.Equal(t, []string{"my-app-node-modules", "my-app-gradle-build", "my-app-gradle-cache"},
			[]string{actual[0].String(), actual[1].String(), actual[2].String()})
	})
	t.Run("Should use a safe cache volume prefix for the zero project value", func(t *testing.T) {
		// given
		var testee ProjectName

		// when
		actual := testee.NodeModulesVolume()

		// then
		require.Equal(t, "project-node-modules", actual.String())
	})
}
