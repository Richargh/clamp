package cli

import (
	"testing"

	filesystem "github.com/richargh/clamp/mitm-proxy/internal/kernel/fs"
	"github.com/stretchr/testify/require"
)

func TestFirewallFlags(t *testing.T) {
	for _, scenario := range []struct {
		name  string
		rules string
		flags []string
		want  string
	}{
		{name: "Should require a folder by default", want: "--allowed-domains-dir is required"},
		{name: "Should enable the firewall for an explicit false flag", flags: []string{"--no-firewall=false"}, want: "--allowed-domains-dir is required"},
		{name: "Should disable the firewall for an explicit true flag", flags: []string{"--no-firewall=true"}, want: "listen:"},
		{name: "Should use the last firewall flag value", flags: []string{"--no-firewall", "--no-firewall=false"}, want: "--allowed-domains-dir is required"},
		{name: "Should reject an invalid firewall flag value", flags: []string{"--no-firewall=invalid"}, want: "invalid syntax"},
		{name: "Should load the folder before listening", rules: "example.com:443", flags: []string{"--allowed-domains-dir"}, want: "listen:"},
		{name: "Should reject invalid rules before listening", rules: "invalid", flags: []string{"--allowed-domains-dir"}, want: "invalid allowed endpoint"},
		{name: "Should permit an empty rules file", flags: []string{"--allowed-domains-dir"}, want: "listen:"},
		{name: "Should reject a missing folder", flags: []string{"--allowed-domains-dir=missing-folder"}, want: "allowed domains:"},
		{name: "Should bypass folder loading with no firewall", flags: []string{"--allowed-domains-dir=missing-folder", "--no-firewall"}, want: "listen:"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// given
			const directory = "allowed-domains"
			env := listenFailureEnvironment(t)
			path, err := filesystem.ParsePath(directory + "/rules.txt")
			require.NoError(t, err)
			env.FileSystem = filesystem.NewMemory(filesystem.MemoryOptions{
				Files: map[filesystem.Path][]byte{path: []byte(scenario.rules)},
			})
			flags := append([]string{"serve", "--listen=127.0.0.1:0"}, scenario.flags...)
			if len(scenario.flags) == 1 && scenario.flags[0] == "--allowed-domains-dir" {
				flags = append(flags, directory)
			}
			testee := New(*env)
			// when
			err = testee.Run(t.Context(), flags)
			// then
			require.ErrorContains(t, err, scenario.want)
		})
	}
}
