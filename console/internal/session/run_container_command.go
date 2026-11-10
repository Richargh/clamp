package session

import (
	"path/filepath"
	"slices"
	"strconv"
)

// containerArgs shows the complete command in execution order. Optional groups
// add arguments directly; no host shell parses or expands these values.
func (l *Runner) containerArgs(o Options, session containerSession) []string {
	return slices.Concat(
		[]string{"run", "-it", "--rm",
			"--name", session.name.String(),
			"--user", "root"},
		l.runtimeArgs(o.Runtime),
		[]string{
			"--cap-add=NET_ADMIN",
			"-e", "CLAMP_HOST_PROXY_PORT=" + strconv.Itoa(int(session.proxyPort)),
			"-e", "CLAMP_HOST_PROXY_HOST=" + hostProxyName(o.Runtime),
			"-e", "NO_PROXY=",
		},
		l.timezoneArgs(),
		[]string{
			"-v", filepath.Join(l.config.AssetsDir().String(), "startup-scripts") + ":/opt/clamp/startup:ro",
			"-v", workspaceMount(o.Runtime, session.workspace),
			"--tmpfs", "/workspace/.clamp:rw,noexec,nosuid,nodev,mode=700",
			"-v", session.logDirectory.String() + ":/workspace/.clamp/sessions",
			"-v", session.project.NodeModulesVolume().String() + ":/workspace/node_modules",
			"-v", session.project.GradleBuildVolume().String() + ":/workspace/build",
			"-v", session.project.GradleCacheVolume().String() + ":/home/dev/.gradle",
			"-v", session.harnessStorage.Volume().String() + ":" + o.Harness.configDir,
			"-e", o.Harness.configEnv + "=" + o.Harness.configDir,
			"-e", "CLAMP_CONFIG_DIR=" + o.Harness.configDir,
			"-e", "CLAMP_CONTAINER_USER=dev",
			"-e", "CLAMP_NETWORK_LOG_FILE=/workspace/.clamp/sessions/" + session.logName.String(),
			session.image.String(),
			"/opt/clamp/startup/container-startup.sh", "--harness=" + o.Harness.name,
		},
		[]string{"--", "bash", "-c", agentCommand(o)},
	)
}

func (l *Runner) runtimeArgs(runtime Runtime) []string {
	if runtime.name == "docker" {
		return []string{"--security-opt", "no-new-privileges", "--add-host", "host.docker.internal:host-gateway"}
	}
	return []string{"--dns", l.config.User().ContainerDNS().String()}
}

func (l *Runner) timezoneArgs() []string {
	if zone := l.config.User().ContainerTimezone().String(); zone != "" {
		return []string{"-e", "TZ=" + zone}
	}
	return nil
}

func hostProxyName(runtime Runtime) string {
	if runtime.name == "docker" {
		return "host.docker.internal"
	}
	return "host.container.internal"
}

func workspaceMount(runtime Runtime, workspace Workspace) string {
	mount := workspace.String() + ":/workspace"
	if runtime.name == "docker" {
		mount += ":delegated"
	}
	return mount
}

func agentCommand(o Options) string {
	if o.Shell {
		return "bash"
	}
	if o.Danger && o.Harness.danger != "" {
		return o.Harness.name + " " + o.Harness.danger
	}
	return o.Harness.name
}
