package session

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/richargh/clamp/console/internal/kernel/filesystem"
)

func (l *Runner) setupProject(workspace Workspace) error {
	dir := filepath.Join(workspace.String(), ".clamp")
	if err := l.env.Files.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err := l.setupAllowedDomains(filepath.Join(dir, "allowed-domains.d")); err != nil {
		return err
	}
	_, err := l.env.Files.Stat(filepath.Join(workspace.String(), "mise.toml"))
	mise := err == nil
	if err != nil && !errors.Is(err, filesystem.ErrNotExist) {
		return err
	}
	for _, file := range []struct {
		name    string
		content []byte
	}{
		{"clamp.Dockerfile.dockerignore", defaultProjectDockerignore(mise)},
		{"clamp.Dockerfile", defaultProjectDockerfile(mise)},
	} {
		path := filepath.Join(dir, file.name)
		if _, err := l.env.Files.Stat(path); errors.Is(err, filesystem.ErrNotExist) {
			if err := l.env.Files.WriteFile(path, file.content, 0644); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(l.env.Stdout, "  Created project file: %s\n", path)
		} else if err != nil {
			return err
		}
	}
	if mise {
		data, err := l.env.Files.ReadFile(filepath.Join(dir, "clamp.Dockerfile"))
		if err != nil {
			return err
		}
		if !installsMiseTools(data) {
			_, _ = fmt.Fprintln(l.env.Stdout, "  Note: mise.toml exists, but this Dockerfile does not install mise tools.\n  Add the mise install block to the Dockerfile, or delete it to regenerate.")
		}
	}
	return nil
}

func installsMiseTools(dockerfile []byte) bool {
	for _, line := range strings.Split(string(dockerfile), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "#") && strings.Contains(line, "mise install -C /tmp") {
			return true
		}
	}
	return false
}
