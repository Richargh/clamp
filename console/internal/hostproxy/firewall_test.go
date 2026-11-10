package hostproxy

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContainerFirewall(t *testing.T) {
	t.Run("Should permit only TCP to the host proxy", func(t *testing.T) {
		// given
		dir := t.TempDir()
		log := filepath.Join(dir, "rules")
		for _, name := range []string{"iptables", "ip6tables"} {
			require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\nprintf '%s %s\\n' \"$(basename \"$0\")\" \"$*\" >> \"$RULE_LOG\"\n"), 0700))
		}
		require.NoError(t, os.WriteFile(filepath.Join(dir, "getent"), []byte("#!/bin/sh\necho '192.0.2.1 STREAM host'\n"), 0700))
		testee := exec.CommandContext(t.Context(), "bash", "../../../startup-scripts/init-host-proxy-firewall.sh")
		testee.Env = append(os.Environ(), "PATH="+dir+":"+os.Getenv("PATH"), "RULE_LOG="+log, "CLAMP_HOST_PROXY_HOST=host.test", "CLAMP_HOST_PROXY_PORT=18888")
		// when
		output, err := testee.CombinedOutput()
		// then
		require.NoError(t, err, string(output))
		data, err := os.ReadFile(log)
		require.NoError(t, err)
		var rules []string
		for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			if !strings.Contains(line, "-L OUTPUT") {
				rules = append(rules, line)
			}
		}
		require.Equal(t, []string{
			"iptables -P OUTPUT DROP",
			"iptables -F OUTPUT",
			"ip6tables -P OUTPUT DROP",
			"ip6tables -F OUTPUT",
			"iptables -A OUTPUT -p tcp -d 192.0.2.1 --dport 18888 -j ACCEPT",
		}, rules)
	})
}
