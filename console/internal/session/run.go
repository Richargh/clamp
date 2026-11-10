package session

import (
	"context"
	"fmt"

	"github.com/richargh/clamp/console/internal/hostproxy"
	"path/filepath"

	"github.com/richargh/clamp/console/internal/kernel/process"
)

func (l *Runner) run(ctx context.Context, o Options, session containerSession) error {
	if err := l.env.Files.MkdirAll(session.logDirectory.String(), 0755); err != nil {
		return err
	}
	proxy, err := hostproxy.New(l.env).Start(ctx, hostproxy.Options{
		AssetsDir: l.config.AssetsDir(), Workspace: session.workspace.path,
		LogDirectory: session.logDirectory.path, LogName: session.logName.String(),
		NoFirewall: o.NoFirewall,
	})
	if err != nil {
		return fmt.Errorf("start host proxy: %w", err)
	}
	defer proxy.Close()
	session.proxyPort = proxy.Port()
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		select {
		case <-proxy.Done():
			cancel()
		case <-runCtx.Done():
		}
	}()
	err = l.execute(runCtx, o.Runtime.executable, l.containerArgs(o, session), false)
	_ = l.shutdown(context.WithoutCancel(ctx), session.workspace, session.logName)
	return err
}

func (l *Runner) shutdown(ctx context.Context, workspace Workspace, logName SessionLogName) error {
	executable, err := process.ParseExecutable(filepath.Join(l.config.AssetsDir().String(), "shutdown-scripts", "clamp-shutdown.sh"))
	if err != nil {
		return err
	}
	return l.execute(ctx, executable, []string{workspace.String(), logName.String()}, false)
}
