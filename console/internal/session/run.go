package session

import (
	"context"
	"path/filepath"

	"github.com/richargh/clamp/console/internal/kernel/process"
)

func (l *Runner) run(ctx context.Context, o Options, project projectIdentity, policy AuthPolicy) error {
	session, err := newContainerSession(project, policy, l.env.Clock.Now())
	if err != nil {
		return err
	}
	if err := l.env.Files.MkdirAll(session.logDirectory.String(), 0755); err != nil {
		return err
	}
	err = l.execute(ctx, o.Runtime.executable, l.containerArgs(o, session), false)
	_ = l.shutdown(context.WithoutCancel(ctx), project.workspace, session.logName)
	return err
}

func (l *Runner) shutdown(ctx context.Context, workspace Workspace, logName SessionLogName) error {
	executable, err := process.ParseExecutable(filepath.Join(l.config.AssetsDir().String(), "shutdown-scripts", "clamp-shutdown.sh"))
	if err != nil {
		return err
	}
	return l.execute(ctx, executable, []string{workspace.String(), logName.String()}, false)
}
