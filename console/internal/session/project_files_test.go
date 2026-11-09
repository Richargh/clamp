package session

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProjectFileRendering(t *testing.T) {
	t.Run("Should prepend the dynamic header without changing project additions", func(t *testing.T) {
		// given
		tool, err := ParseImageName("clamp-pi")
		require.NoError(t, err)
		additions := []byte("RUN echo project\n")
		testee := renderProjectDockerfile

		// when
		actual := testee(tool, additions)

		// then
		require.Equal(t, "ARG CLAMP_TOOL_IMAGE=clamp-pi\nFROM ${CLAMP_TOOL_IMAGE}\n\nARG CONTAINER_USER=dev\n\nCOPY .clamp/allowed-domains.d/ /opt/clamp-shared/allowed-domains.d/\n\nUSER $CONTAINER_USER\nWORKDIR /workspace\n\nRUN echo project\n", string(actual))
	})
	t.Run("Should restrict the build context before applying project exceptions", func(t *testing.T) {
		// given
		additions := []byte("!src/\n!src/**\n")
		testee := renderProjectDockerignore

		// when
		actual := testee(additions)

		// then
		require.Equal(t, "*\n!.clamp/\n!.clamp/**\n!src/\n!src/**\n", string(actual))
	})
}

func TestProjectFileDefaults(t *testing.T) {
	for _, scenario := range []struct {
		name string
		mise bool
	}{
		{name: "with mise", mise: true},
		{name: "without mise", mise: false},
	} {
		t.Run("Should generate the Dockerfile "+scenario.name, func(t *testing.T) {
			// given
			testee := defaultProjectDockerfile

			// when
			actual := testee(scenario.mise)

			// then
			require.Contains(t, string(actual), "USER $CONTAINER_USER\n")
			require.Equal(t, scenario.mise, strings.Contains(string(actual), "\nRUN mise install -C /tmp\n"))
		})
		t.Run("Should generate the context exceptions "+scenario.name, func(t *testing.T) {
			// given
			testee := defaultProjectDockerignore

			// when
			actual := testee(scenario.mise)

			// then
			require.Contains(t, string(actual), "# Clamp already includes .clamp/**.\n")
			if scenario.mise {
				require.Contains(t, string(actual), "\n!mise.toml\n")
			} else {
				require.NotContains(t, string(actual), "\n!mise.toml\n")
			}
		})
	}
}
