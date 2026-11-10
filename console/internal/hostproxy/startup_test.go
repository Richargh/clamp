package hostproxy

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLegacyLaunchers(t *testing.T) {
	for _, script := range []string{"clamp-lib.sh", "clamp-pi.sh", "clamp-claude.sh", "clamp-open-code.sh"} {
		t.Run("Should reject the deprecated "+script, func(t *testing.T) {
			// given
			testee := exec.CommandContext(t.Context(), "bash", "../../../"+script)
			// when
			output, err := testee.CombinedOutput()
			// then
			var exit *exec.ExitError
			require.ErrorAs(t, err, &exit)
			require.Equal(t, 1, exit.ExitCode())
			require.Contains(t, string(output), "clamp-lib.sh is deprecated")
			require.Contains(t, string(output), "bin/clamp run <harness> <folder>")
		})
	}
	t.Run("Should return failure when the old library is sourced", func(t *testing.T) {
		// given
		testee := exec.CommandContext(t.Context(), "bash", "-c", `source ../../../clamp-lib.sh; status=$?; if declare -F clamp_main; then exit 99; fi; exit "$status"`)
		// when
		_, err := testee.CombinedOutput()
		// then
		var exit *exec.ExitError
		require.ErrorAs(t, err, &exit)
		require.Equal(t, 1, exit.ExitCode())
	})
}

func TestHostProxyStartupEnvironment(t *testing.T) {
	t.Run("Should expose proxy host and port variables to the agent", func(t *testing.T) {
		// given
		script, err := os.ReadFile("../../../startup-scripts/container-startup.sh")
		require.NoError(t, err)
		_, block, found := strings.Cut(string(script), "HARNESS_ENV=(\n")
		require.True(t, found)
		block, _, found = strings.Cut(block, "\n)")
		require.True(t, found)
		testee := exec.CommandContext(t.Context(), "bash", "-c", "HARNESS_ENV=(\n"+block+"\n)\nenv -i \"${HARNESS_ENV[@]}\"")
		testee.Env = []string{
			"PATH=" + os.Getenv("PATH"),
			"CLAMP_CONTAINER_HOME=/home/dev",
			"CLAMP_CONTAINER_USER=dev",
			"CLAMP_HOST_PROXY_HOST=192.0.2.10",
			"CLAMP_HOST_PROXY_PORT=18888",
			"NO_PROXY=",
		}

		// when
		output, err := testee.CombinedOutput()

		// then
		require.NoError(t, err, string(output))
		variables := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, protocol := range []string{"HTTP", "HTTPS"} {
			require.Contains(t, variables, protocol+"_PROXY=http://192.0.2.10:18888")
			require.Contains(t, variables, protocol+"_PROXY_HOST=192.0.2.10")
			require.Contains(t, variables, protocol+"_PROXY_PORT=18888")
		}
	})
}

func TestHostProxyStartupRequirements(t *testing.T) {
	for _, scenario := range []struct {
		name    string
		env     []string
		message string
	}{
		{"Should stop without a host proxy", nil, "Error: CLAMP_HOST_PROXY_HOST must be set"},
		{"Should stop without a host proxy port", []string{"CLAMP_HOST_PROXY_HOST=host.test"}, "Error: CLAMP_HOST_PROXY_PORT must be set"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// given
			args := []string{"../../../startup-scripts/container-startup.sh", "--harness=pi", "--", "true"}
			testee := exec.CommandContext(t.Context(), "bash", args...)
			testee.Env = append([]string{"PATH=" + os.Getenv("PATH")}, scenario.env...)
			// when
			output, err := testee.CombinedOutput()
			// then
			require.Error(t, err)
			require.Contains(t, string(output), scenario.message)
		})
	}
}
