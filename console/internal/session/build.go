package session

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
)

var completeDockerfile = regexp.MustCompile(`(?m)^[\t ]*FROM[\t ]+`)

func (l *Runner) build(ctx context.Context, o Options, project projectIdentity) error {
	for _, name := range []string{"clamp-base", project.toolImage.String()} {
		dockerfile := filepath.Join(l.config.AssetsDir().String(), name+".Dockerfile")
		if _, err := l.env.Files.Stat(dockerfile); err != nil {
			return fmt.Errorf("missing Dockerfile: %s: %w", dockerfile, err)
		}
	}
	for _, name := range []string{"clamp-base", project.toolImage.String()} {
		if !o.Rebuild && !o.NoCache && l.execute(ctx, o.Runtime.executable, []string{"image", "inspect", name}, true) == nil {
			continue
		}
		args := l.buildArgs(o)
		args = append(args, "-f", filepath.Join(l.config.AssetsDir().String(), name+".Dockerfile"), "-t", name, l.config.AssetsDir().String())
		_, _ = fmt.Fprintf(l.env.Stdout, "[Clamp Image Build]\n  Image: %s\n", name)
		if err := l.execute(ctx, o.Runtime.executable, args, false); err != nil {
			return err
		}
	}
	return l.buildProjectImage(ctx, o, project)
}

func (l *Runner) buildProjectImage(ctx context.Context, o Options, project projectIdentity) error {
	dir := filepath.Join(project.workspace.String(), ".clamp")
	path := filepath.Join(dir, "clamp.Dockerfile")
	data, err := l.env.Files.ReadFile(path)
	if err != nil {
		return err
	}
	if completeDockerfile.Match(data) {
		_, _ = fmt.Fprintln(l.env.Stderr, "[Warning] This is a legacy complete Dockerfile; remove its standard Clamp header to use the dynamic header.")
	} else {
		path, err = l.env.Files.TempFile(dir, "clamp.generated.*")
		if err != nil {
			return err
		}
		defer func() { _ = l.env.Files.Remove(path); _ = l.env.Files.Remove(path + ".dockerignore") }()
		if err := l.env.Files.WriteFile(path, renderProjectDockerfile(project.toolImage, data), 0600); err != nil {
			return err
		}
		ignore, err := l.env.Files.ReadFile(filepath.Join(dir, "clamp.Dockerfile.dockerignore"))
		if err != nil {
			return err
		}
		if err := l.env.Files.WriteFile(path+".dockerignore", renderProjectDockerignore(ignore), 0600); err != nil {
			return err
		}
	}
	args := l.buildArgs(o)
	args = append(args, "--build-arg", "CLAMP_TOOL_IMAGE="+project.toolImage.String(), "-f", path, "-t", project.image.String(), project.workspace.String())
	_, _ = fmt.Fprintf(l.env.Stdout, "[Clamp Image Build]\n  Image: %s\n  Dockerfile: .clamp/clamp.Dockerfile\n  Why: Clamp builds a per-project image so project Dockerfile changes, allowed-domain files, and tool setup are applied before the agent starts.\n", project.image)
	return l.execute(ctx, o.Runtime.executable, args, false)
}

func (l *Runner) buildArgs(o Options) []string {
	args := []string{"build"}
	if o.Runtime.name == "container" {
		args = append(args, "--dns", l.config.ContainerDNS().String())
	}
	if o.NoCache {
		args = append(args, "--no-cache")
	}
	return args
}
