package session

import (
	"testing"
	"time"

	"github.com/richargh/clamp/console/internal/config"
	"github.com/richargh/clamp/console/internal/kernel/environment"
	"github.com/stretchr/testify/require"
)

func TestContainerCommand(t *testing.T) {
	t.Run("Should assemble the complete Pi command for Docker in execution order", func(t *testing.T) {
		// given
		memory := environment.NewMemory()
		options := launchOptions("/projects/my-app")
		harness, err := ParseHarness("pi")
		require.NoError(t, err)
		options.Harness = harness
		workspace, err := ParseWorkspace("/projects/my-app")
		require.NoError(t, err)
		project := newProjectIdentity(harness, workspace)
		now := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
		session, err := newContainerSession(project, ProjectAuthPolicy(), now)
		require.NoError(t, err)
		testee := New(memory.Env, launchConfig(t))

		// when
		actual := testee.containerArgs(options, session)

		// then
		require.Equal(t, []string{
			"run", "-it", "--rm",
			"--name", "pi-20261027-120000",
			"--user", "root",
			"--security-opt", "no-new-privileges",
			"--cap-add=NET_ADMIN",
			"-e", "HTTP_PROXY=http://127.0.0.1:8888",
			"-e", "HTTPS_PROXY=http://127.0.0.1:8888",
			"-e", "NO_PROXY=",
			"-v", "/projects/my-app:/workspace:delegated",
			"--tmpfs", "/workspace/.clamp:rw,noexec,nosuid,nodev,mode=700",
			"-v", "/projects/my-app/.clamp/sessions:/workspace/.clamp/sessions",
			"-v", "my-app-node-modules:/workspace/node_modules",
			"-v", "my-app-gradle-build:/workspace/build",
			"-v", "my-app-gradle-cache:/home/dev/.gradle",
			"-v", "pi-my-app-3670971996:/home/dev/.pi",
			"-e", "CLAUDE_CONFIG_DIR=/home/dev/.pi",
			"-e", "CLAMP_CONFIG_DIR=/home/dev/.pi",
			"-e", "CLAMP_CONTAINER_USER=dev",
			"-e", "CLAMP_NETWORK_LOG_FILE=/workspace/.clamp/sessions/20261027-120000-network.log",
			"clamp-pi-3670971996",
			"/opt/clamp/startup/container-startup.sh", "--harness=pi",
			"--", "bash", "-c", "pi",
		}, actual)
	})
	t.Run("Should assemble Apple container options before the image and startup flags after it", func(t *testing.T) {
		// given
		memory := environment.NewMemory()
		raw := launchConfigInput()
		raw.ContainerDNS = "9.9.9.9"
		raw.ContainerTimezone = "Europe/Berlin"
		settings, err := config.Parse(raw)
		require.NoError(t, err)
		options := launchOptions("/projects/my-app")
		options.Runtime, err = ParseRuntime("container")
		require.NoError(t, err)
		options.NoFirewall = true
		options.Danger = true
		workspace, err := ParseWorkspace("/projects/my-app")
		require.NoError(t, err)
		project := newProjectIdentity(options.Harness, workspace)
		now := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
		session, err := newContainerSession(project, GlobalAuthPolicy(), now)
		require.NoError(t, err)
		testee := New(memory.Env, settings)

		// when
		actual := testee.containerArgs(options, session)

		// then
		require.Equal(t, []string{
			"run", "-it", "--rm",
			"--name", "claude-clamp-20261027-120000",
			"--user", "root",
			"--dns", "9.9.9.9",
			"--cap-add=NET_ADMIN",
			"-e", "HTTP_PROXY=http://127.0.0.1:8888",
			"-e", "HTTPS_PROXY=http://127.0.0.1:8888",
			"-e", "NO_PROXY=",
			"-e", "TZ=Europe/Berlin",
			"-v", "/projects/my-app:/workspace",
			"--tmpfs", "/workspace/.clamp:rw,noexec,nosuid,nodev,mode=700",
			"-v", "/projects/my-app/.clamp/sessions:/workspace/.clamp/sessions",
			"-v", "my-app-node-modules:/workspace/node_modules",
			"-v", "my-app-gradle-build:/workspace/build",
			"-v", "my-app-gradle-cache:/home/dev/.gradle",
			"-v", "claude-clamp:/home/dev/.claude",
			"-e", "CLAUDE_CONFIG_DIR=/home/dev/.claude",
			"-e", "CLAMP_CONFIG_DIR=/home/dev/.claude",
			"-e", "CLAMP_CONTAINER_USER=dev",
			"-e", "CLAMP_NETWORK_LOG_FILE=/workspace/.clamp/sessions/20261027-120000-network.log",
			"clamp-claude-3670971996",
			"/opt/clamp/startup/container-startup.sh", "--harness=claude",
			"--no-firewall",
			"--", "bash", "-c", "claude --dangerously-skip-permissions",
		}, actual)
	})
}
