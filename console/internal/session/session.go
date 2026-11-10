package session

import (
	"context"
	"fmt"
	"io"

	"github.com/richargh/clamp/console/internal/config"
	"github.com/richargh/clamp/console/internal/kernel/environment"
	"github.com/richargh/clamp/console/internal/kernel/process"
)

type Runner struct {
	env    environment.Env
	config config.Config
}

func New(env environment.Env, settings config.Config) *Runner {
	return &Runner{env: env, config: settings}
}

// Launch is the non-interactive entry point for sessions with a saved or explicit
// volume choice. Interactive callers prepare a Session before launching it.
func (l *Runner) Launch(ctx context.Context, o Options) error {
	prepared, err := l.Prepare(o)
	if err != nil {
		return err
	}
	return prepared.Launch(ctx)
}

// Prepare resolves the session without building images or running processes.
// A new project may still need a harness volume selection from the caller.
func (l *Runner) Prepare(o Options) (*Session, error) {
	if o.Harness.name == "" || o.Runtime.name == "" {
		return nil, fmt.Errorf("harness and runtime are required")
	}
	executable, err := l.env.Processes.LookPath(o.Runtime.name)
	if err != nil {
		return nil, fmt.Errorf("container runtime not found: %s: %w", o.Runtime.name, err)
	}
	o.Runtime.executable = executable
	workspace, err := l.resolveWorkspace(o.Folder)
	if err != nil {
		return nil, err
	}
	choice, err := l.loadHarnessStorage(workspace.String(), o.PerProjectAuth)
	if err != nil {
		return nil, err
	}
	project := newProjectIdentity(o.Harness, workspace)
	container, err := newContainerSession(project, l.env.Clock.Now())
	if err != nil {
		return nil, err
	}
	prepared := &Session{runner: l, options: o, project: project, container: container}
	if err := prepared.SetHarnessVolume(choice); err != nil {
		return nil, err
	}
	return prepared, nil
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
