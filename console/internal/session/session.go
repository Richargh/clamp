package session

import (
	"context"
	"fmt"
	"io"

	"github.com/richargh/clamp/console/internal/config"
	"github.com/richargh/clamp/console/internal/kernel/environment"
	"github.com/richargh/clamp/console/internal/kernel/process"
	"github.com/richargh/clamp/console/internal/settings"
)

type Runner struct {
	env    environment.Env
	config config.Config
}

func New(env environment.Env, settings config.Config) *Runner {
	return &Runner{env: env, config: settings}
}

// Launch starts a session. It calls selectHarnessVolume only when no saved or
// explicit volume choice exists. A nil selector requires an existing choice.
func Launch(ctx context.Context, env environment.Env, settings config.Config, o Options, selectHarnessVolume func(Workspace) (HarnessVolumeChoice, error)) error {
	return New(env, settings).launch(ctx, o, selectHarnessVolume)
}

// Launch starts a session that already has a saved or explicit volume choice.
func (l *Runner) Launch(ctx context.Context, o Options) error {
	return l.launch(ctx, o, nil)
}

func (l *Runner) launch(ctx context.Context, o Options, selectHarnessVolume func(Workspace) (HarnessVolumeChoice, error)) error {
	if o.Harness.name == "" || o.Runtime.name == "" {
		return fmt.Errorf("harness and runtime are required")
	}
	executable, err := l.env.Processes.LookPath(o.Runtime.name)
	if err != nil {
		return fmt.Errorf("container runtime not found: %s: %w", o.Runtime.name, err)
	}
	o.Runtime.executable = executable
	workspace, err := l.resolveWorkspace(o.Folder)
	if err != nil {
		return err
	}
	project := newProjectIdentity(o.Harness, workspace)
	storage, err := l.selectStorage(project, o.PerProjectAuth, selectHarnessVolume)
	if err != nil {
		return err
	}
	container, err := newContainerSession(project, l.env.Clock.Now())
	if err != nil {
		return err
	}
	container.harnessStorage = storage
	if err := l.rememberHarnessStorage(workspace.String(), storage.Choice()); err != nil {
		return err
	}
	l.printConfig(o, storage.Choice())
	projectSettings, err := settings.ParseProject(workspace.String())
	if err != nil {
		return err
	}
	if err := settings.BootstrapProject(l.env, projectSettings, l.config.AssetsDir()); err != nil {
		return err
	}
	if err := l.build(ctx, o, project); err != nil {
		return err
	}
	return l.run(ctx, o, container)
}

func (l *Runner) selectStorage(project projectIdentity, explicit bool, selectHarnessVolume func(Workspace) (HarnessVolumeChoice, error)) (HarnessVolume, error) {
	choice, err := l.loadHarnessStorage(project.workspace.String(), explicit)
	if err != nil {
		return HarnessVolume{}, err
	}
	if choice == NoChoiceMade && selectHarnessVolume != nil {
		choice, err = selectHarnessVolume(project.workspace)
		if err != nil {
			return HarnessVolume{}, err
		}
	}
	if choice == NoChoiceMade {
		return HarnessVolume{}, fmt.Errorf("harness storage selection is required for %s", project.workspace.String())
	}
	return resolveHarnessStorage(choice, project)
}

func (l *Runner) printConfig(o Options, choice HarnessVolumeChoice) {
	auth := "Global"
	if choice == ProjectVolume {
		auth = "Project"
	}
	firewall := "On"
	if o.NoFirewall {
		firewall = "Off"
	}
	_, _ = fmt.Fprintf(l.env.Stdout, "[Clamp User Config]\n  Auth: %s\n  Auth Config: %s\n\n[Clamp CLI Config]\n  Coding Agent: %s\n  Runtime: %s\n  Firewall: %s\n\n", auth, l.authPath(), o.Harness.display, o.Runtime.name, firewall)
}

func (l *Runner) execute(ctx context.Context, executable process.Executable, args []string, quiet bool) error {
	out, stderr := l.env.Stdout, l.env.Stderr
	if quiet {
		out = io.Discard
		stderr = io.Discard
	}
	return l.env.Processes.Run(ctx, process.Process{Executable: executable, Args: args, Stdin: l.env.Stdin, Stdout: out, Stderr: stderr})
}

// projectChecksum matches POSIX cksum, including its length suffix.
func projectChecksum(data []byte) uint32 {
	var crc uint32
	update := func(b byte) {
		crc ^= uint32(b) << 24
		for range 8 {
			if crc&0x80000000 != 0 {
				crc = crc<<1 ^ 0x04c11db7
			} else {
				crc <<= 1
			}
		}
	}
	for _, b := range data {
		update(b)
	}
	for n := len(data); n != 0; n >>= 8 {
		update(byte(n))
	}
	return ^crc
}
