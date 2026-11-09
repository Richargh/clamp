package session

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/richargh/clamp/console/internal/kernel/filesystem"
)

// Workspace is an absolute host workspace path that can be used in a bind mount
// and in the tab-separated auth store. Parsing does not access the file system.
type Workspace struct{ path filesystem.Path }

func ParseWorkspace(raw string) (Workspace, error) {
	path, err := parseMountPath(raw)
	if err != nil {
		return Workspace{}, fmt.Errorf("workspace: %w", err)
	}
	return Workspace{path: path}, nil
}

func (w Workspace) String() string { return w.path.String() }

func (w Workspace) baseName() string { return filepath.Base(w.String()) }

func (w Workspace) SessionLogs() (SessionLogDirectory, error) {
	return ParseSessionLogDirectory(filepath.Join(w.String(), ".clamp", "sessions"))
}

// SessionLogDirectory identifies the host directory mounted for session logs.
type SessionLogDirectory struct{ path filesystem.Path }

func ParseSessionLogDirectory(raw string) (SessionLogDirectory, error) {
	path, err := parseMountPath(raw)
	if err != nil {
		return SessionLogDirectory{}, fmt.Errorf("session log directory: %w", err)
	}
	return SessionLogDirectory{path: path}, nil
}

func (d SessionLogDirectory) String() string { return d.path.String() }

func parseMountPath(raw string) (filesystem.Path, error) {
	if strings.ContainsAny(raw, ":\t\r\n") {
		return filesystem.Path{}, fmt.Errorf("mount path cannot contain colons, tabs, or newlines: %q", raw)
	}
	return filesystem.ParsePath(raw)
}

func (l *Runner) resolveWorkspace(folder string) (Workspace, error) {
	resolved, err := l.env.Files.Resolve(folder)
	if err != nil {
		return Workspace{}, err
	}
	workspace, err := ParseWorkspace(resolved)
	if err != nil {
		return Workspace{}, err
	}
	info, err := l.env.Files.Stat(workspace.String())
	if err != nil {
		return Workspace{}, err
	}
	if !info.IsDir() {
		return Workspace{}, fmt.Errorf("not a folder: %s", workspace)
	}
	return workspace, nil
}
