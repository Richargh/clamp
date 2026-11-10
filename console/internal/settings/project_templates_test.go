package settings

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

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
