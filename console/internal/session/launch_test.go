package session

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/richargh/clamp/console/internal/config"
	"github.com/richargh/clamp/console/internal/kernel/process"
	"github.com/stretchr/testify/require"
)

func TestLaunch(t *testing.T) {
	for _, runtime := range []string{"docker", "container"} {
		for _, name := range []string{"pi", "claude", "opencode"} {
			t.Run("Should launch "+name+" with "+runtime, func(t *testing.T) {
				// given
				env, workspace, processes := newLaunchEnv(t)
				options := launchOptions(workspace)
				var err error
				options.Harness, err = ParseHarness(name)
				require.NoError(t, err)
				options.Runtime, err = ParseRuntime(runtime)
				require.NoError(t, err)
				testee := New(env, launchConfig(t))

				// when
				err = testee.Launch(t.Context(), options)

				// then
				require.NoError(t, err)
				run := requireContainerRun(t, processes.Calls)
				require.Equal(t, "/bin/"+runtime, run.Executable.String())
				require.Contains(t, run.Args, "--harness="+name)
			})
		}
	}
}

func TestProjectConfiguration(t *testing.T) {
	for _, scenario := range []struct {
		harness, prefix, configDir, configEnv string
	}{
		{"pi", "pi", "/home/dev/.pi", "CLAUDE_CONFIG_DIR"},
		{"claude", "claude-clamp", "/home/dev/.claude", "CLAUDE_CONFIG_DIR"},
		{"opencode", "opencode-clamp", "/home/dev/.local/share/opencode", "OPENCODE_CONFIG_DIR"},
	} {
		t.Run("Should isolate project credentials for "+scenario.harness, func(t *testing.T) {
			// given
			env, workspace, processes := newLaunchEnv(t)
			options := launchOptions(workspace)
			var err error
			options.Harness, err = ParseHarness(scenario.harness)
			require.NoError(t, err)
			options.PerProjectAuth = true
			volume := fmt.Sprintf("%s-project-with-spaces-%d", scenario.prefix, projectChecksum([]byte(workspace)))
			testee := New(env, launchConfig(t))

			// when
			err = testee.Launch(t.Context(), options)

			// then
			require.NoError(t, err)
			run := requireContainerRun(t, processes.Calls)
			require.Contains(t, run.Args, volume+":"+scenario.configDir)
			require.Contains(t, run.Args, "CLAMP_CONFIG_DIR="+scenario.configDir)
			require.Contains(t, run.Args, scenario.configEnv+"="+scenario.configDir)
		})
	}
	t.Run("Should save an explicit per-project authentication preference", func(t *testing.T) {
		// given
		env, workspace, _ := newLaunchEnv(t)
		options := launchOptions(workspace)
		options.PerProjectAuth = true
		settings := launchConfig(t)
		testee := New(env, settings)

		// when
		err := testee.Launch(t.Context(), options)

		// then
		require.NoError(t, err)
		data, err := env.Files.ReadFile(settings.AuthStore().String())
		require.NoError(t, err)
		require.Equal(t, workspace+"\tproject\n", string(data))
	})
	t.Run("Should hide host Clamp configuration behind a private temporary mount", func(t *testing.T) {
		// given
		env, workspace, processes := newLaunchEnv(t)
		testee := New(env, launchConfig(t))

		// when
		err := testee.Launch(t.Context(), launchOptions(workspace))

		// then
		require.NoError(t, err)
		run := requireContainerRun(t, processes.Calls)
		require.Contains(t, strings.Join(run.Args, "\n"), "--tmpfs\n/workspace/.clamp:rw,noexec,nosuid,nodev,mode=700")
	})
}

func TestContainerNetworking(t *testing.T) {
	t.Run("Should route HTTP traffic through the container proxy", func(t *testing.T) {
		// given
		env, workspace, processes := newLaunchEnv(t)
		testee := New(env, launchConfig(t))

		// when
		err := testee.Launch(t.Context(), launchOptions(workspace))

		// then
		require.NoError(t, err)
		run := requireContainerRun(t, processes.Calls)
		require.Contains(t, run.Args, "HTTP_PROXY=http://127.0.0.1:8888")
		require.Contains(t, run.Args, "HTTPS_PROXY=http://127.0.0.1:8888")
		require.Contains(t, run.Args, "NO_PROXY=")
	})
	t.Run("Should grant network administration for the firewall", func(t *testing.T) {
		// given
		env, workspace, processes := newLaunchEnv(t)
		testee := New(env, launchConfig(t))

		// when
		err := testee.Launch(t.Context(), launchOptions(workspace))

		// then
		require.NoError(t, err)
		require.Contains(t, requireContainerRun(t, processes.Calls).Args, "--cap-add=NET_ADMIN")
	})
}

func TestRuntimeConfiguration(t *testing.T) {
	t.Run("Should prevent privilege escalation with Docker", func(t *testing.T) {
		// given
		env, workspace, processes := newLaunchEnv(t)
		testee := New(env, launchConfig(t))

		// when
		err := testee.Launch(t.Context(), launchOptions(workspace))

		// then
		require.NoError(t, err)
		run := requireContainerRun(t, processes.Calls)
		require.Contains(t, strings.Join(run.Args, "\n"), "--security-opt\nno-new-privileges")
	})
	t.Run("Should use a delegated workspace mount with Docker", func(t *testing.T) {
		// given
		env, workspace, processes := newLaunchEnv(t)
		testee := New(env, launchConfig(t))

		// when
		err := testee.Launch(t.Context(), launchOptions(workspace))

		// then
		require.NoError(t, err)
		require.Contains(t, requireContainerRun(t, processes.Calls).Args, workspace+":/workspace:delegated")
	})
	t.Run("Should use Apple container runtime options", func(t *testing.T) {
		// given
		env, workspace, processes := newLaunchEnv(t)
		raw := launchConfigInput()
		raw.ContainerDNS = "9.9.9.9"
		settings, err := config.Parse(raw)
		require.NoError(t, err)
		options := launchOptions(workspace)
		options.Runtime, err = ParseRuntime("container")
		require.NoError(t, err)
		testee := New(env, settings)

		// when
		err = testee.Launch(t.Context(), options)

		// then
		require.NoError(t, err)
		run := requireContainerRun(t, processes.Calls)
		require.Contains(t, strings.Join(run.Args, "\n"), "--dns\n9.9.9.9")
		require.NotContains(t, run.Args, "--security-opt")
		require.Contains(t, run.Args, workspace+":/workspace")
		require.NotContains(t, run.Args, workspace+":/workspace:delegated")
	})
	t.Run("Should pass the configured timezone to the container", func(t *testing.T) {
		// given
		env, workspace, processes := newLaunchEnv(t)
		raw := launchConfigInput()
		raw.ContainerTimezone = "Europe/Berlin"
		settings, err := config.Parse(raw)
		require.NoError(t, err)
		testee := New(env, settings)

		// when
		err = testee.Launch(t.Context(), launchOptions(workspace))

		// then
		require.NoError(t, err)
		require.Contains(t, requireContainerRun(t, processes.Calls).Args, "TZ=Europe/Berlin")
	})
}

func TestStartupOptions(t *testing.T) {
	for _, disabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("Should pass the firewall override only when disabled is %t", disabled), func(t *testing.T) {
			// given
			env, workspace, processes := newLaunchEnv(t)
			options := launchOptions(workspace)
			options.NoFirewall = disabled
			testee := New(env, launchConfig(t))

			// when
			err := testee.Launch(t.Context(), options)

			// then
			require.NoError(t, err)
			run := requireContainerRun(t, processes.Calls)
			require.Equal(t, disabled, slices.Contains(run.Args, "--no-firewall"))
		})
	}
	for _, scenario := range []struct {
		name, harness string
		danger, shell bool
		command       string
	}{
		{name: "Should start Claude normally by default", harness: "claude", command: "claude"},
		{name: "Should skip Claude permission confirmations in danger mode", harness: "claude", danger: true, command: "claude --dangerously-skip-permissions"},
		{name: "Should leave Pi unchanged in danger mode", harness: "pi", danger: true, command: "pi"},
		{name: "Should leave OpenCode unchanged in danger mode", harness: "opencode", danger: true, command: "opencode"},
		{name: "Should open a shell instead of the agent in danger mode", harness: "claude", danger: true, shell: true, command: "bash"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// given
			env, workspace, processes := newLaunchEnv(t)
			options := launchOptions(workspace)
			var err error
			options.Harness, err = ParseHarness(scenario.harness)
			require.NoError(t, err)
			options.Danger = scenario.danger
			options.Shell = scenario.shell
			testee := New(env, launchConfig(t))

			// when
			err = testee.Launch(t.Context(), options)

			// then
			require.NoError(t, err)
			run := requireContainerRun(t, processes.Calls)
			require.Equal(t, scenario.command, run.Args[len(run.Args)-1])
		})
	}
}

func requireContainerRun(t *testing.T, calls []process.Process) process.Process {
	t.Helper()
	var runs []process.Process
	for _, call := range calls {
		if len(call.Args) > 0 && call.Args[0] == "run" {
			runs = append(runs, call)
		}
	}
	require.Len(t, runs, 1, "expected one container run command")
	return runs[0]
}
