package hostproxy

import (
	"context"
	"io"
	"path/filepath"
	"testing"

	"github.com/richargh/clamp/console/internal/kernel/environment"
	"github.com/richargh/clamp/console/internal/kernel/filesystem"
	"github.com/richargh/clamp/console/internal/kernel/process"
	"github.com/stretchr/testify/require"
)

func TestHostProxy(t *testing.T) {
	for _, mode := range []string{"memory", "OS"} {
		t.Run(mode, func(t *testing.T) {
			for _, scenario := range []struct {
				name, record string
				port         uint16
				failure      string
			}{
				{"Should wait for readiness before returning the host port", `{"schema":1,"address":"0.0.0.0:23456"}`, 23456, ""},
				{"Should accept a dual-stack wildcard readiness address", `{"schema":1,"address":"[::]:23456"}`, 23456, ""},
				{"Should reject an invalid readiness port", `{"schema":1,"address":"0.0.0.0:0"}`, 0, "invalid proxy listen address"},
				{"Should reject an unsupported readiness schema", `{"schema":2,"address":"0.0.0.0:23456"}`, 0, "unsupported proxy readiness schema"},
			} {
				t.Run(scenario.name, func(t *testing.T) {
					// given
					env, options := proxyFixture(t, mode, scenario.record)
					testee := New(env)
					// when
					proxy, err := testee.Start(t.Context(), options)
					// then
					if scenario.failure != "" {
						require.ErrorContains(t, err, scenario.failure)
						return
					}
					require.NoError(t, err)
					defer proxy.Close()
					require.Equal(t, scenario.port, proxy.Port())
				})
			}
		})
	}
	t.Run("Should reject a proxy that exits before readiness", func(t *testing.T) {
		// given
		env, options := proxyFixture(t, "OS", "")
		require.NoError(t, env.Files.WriteFile(filepath.Join(options.AssetsDir.String(), "bin", "clamp-mitm-proxy"), []byte("#!/bin/sh\nexit 1\n"), 0700))
		testee := New(env)
		// when
		_, err := testee.Start(t.Context(), options)
		// then
		require.Error(t, err)
	})
}

func TestHostProxyCommand(t *testing.T) {
	for _, disabled := range []bool{false, true} {
		name := "Should launch the proxy with the project domain directory"
		if disabled {
			name = "Should disable only the proxy domain filter with no-firewall"
		}
		t.Run(name, func(t *testing.T) {
			// given
			env, options := proxyFixture(t, "memory", `{"schema":1,"address":"0.0.0.0:23456"}`)
			workspace, err := filesystem.ParsePath("/project with spaces")
			require.NoError(t, err)
			options.Workspace = workspace
			options.NoFirewall = disabled
			processes := env.Processes.(*process.Memory)
			testee := New(env)
			// when
			proxy, err := testee.Start(t.Context(), options)
			require.NoError(t, err)
			proxy.Close()
			// then
			require.Len(t, processes.Calls, 1)
			require.Equal(t, "/proxy/bin/clamp-mitm-proxy", processes.Calls[0].Executable.String())
			expected := []string{"serve", "--listen", "0.0.0.0:0"}
			if disabled {
				expected = append(expected, "--no-firewall")
			} else {
				expected = append(expected, "--allowed-domains-dir", "/project with spaces/.clamp/allowed-domains.d")
			}
			require.Equal(t, expected, processes.Calls[0].Args)
		})
	}
}

func proxyFixture(t *testing.T, mode, record string) (environment.Env, Options) {
	t.Helper()
	memory := environment.NewMemory()
	env := memory.Env
	dir := "/proxy"
	if mode == "OS" {
		env = environment.NewOS()
		dir = t.TempDir()
	}
	require.NoError(t, env.Files.MkdirAll(filepath.Join(dir, "bin"), 0700))
	if mode == "OS" {
		script := "#!/bin/sh\nprintf '%s\\n' '" + record + "'\nexec sleep 30\n"
		require.NoError(t, env.Files.WriteFile(filepath.Join(dir, "bin", "clamp-mitm-proxy"), []byte(script), 0700))
	} else {
		memory.ProcessMemory.OnRun = func(ctx context.Context, p process.Process) error {
			if _, err := io.WriteString(p.Stdout, record+"\n"); err != nil {
				return err
			}
			<-ctx.Done()
			return ctx.Err()
		}
	}
	path, err := filesystem.ParsePath(dir)
	require.NoError(t, err)
	return env, Options{AssetsDir: path, Workspace: path, LogDirectory: path, LogName: "network.log"}
}
