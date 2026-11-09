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

func (l *Runner) Launch(ctx context.Context, o Options) error {
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
	policy, err := l.authPolicy(workspace.String(), o.PerProjectAuth)
	if err != nil {
		return err
	}
	l.printConfig(o, policy)
	project := newProjectIdentity(o.Harness, workspace)
	if err := l.setupProject(workspace); err != nil {
		return err
	}
	if err := l.build(ctx, o, project); err != nil {
		return err
	}
	return l.run(ctx, o, project, policy)
}

func (l *Runner) printConfig(o Options, policy AuthPolicy) {
	auth := "Global"
	if policy.IsProject() {
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
