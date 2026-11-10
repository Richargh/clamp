package settings

import (
	"fmt"
	"path/filepath"

	"github.com/richargh/clamp/console/internal/kernel/filesystem"
)

// Project identifies settings shared by a workspace. It contains no user defaults.
type Project struct {
	workspace filesystem.Path
}

func ParseProject(workspace string) (Project, error) {
	path, err := filesystem.ParsePath(workspace)
	if err != nil {
		return Project{}, fmt.Errorf("project workspace: %w", err)
	}
	return Project{workspace: path}, nil
}

func (p Project) Workspace() filesystem.Path { return p.workspace }

func (p Project) Directory() string  { return filepath.Join(p.workspace.String(), ".clamp") }
func (p Project) Dockerfile() string { return filepath.Join(p.Directory(), "clamp.Dockerfile") }
func (p Project) Dockerignore() string {
	return filepath.Join(p.Directory(), "clamp.Dockerfile.dockerignore")
}
func (p Project) AllowedDomains() string { return filepath.Join(p.Directory(), "allowed-domains.d") }
