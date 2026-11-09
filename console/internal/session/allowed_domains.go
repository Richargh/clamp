package session

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/richargh/clamp/console/internal/kernel/filesystem"
)

func (l *Runner) setupAllowedDomains(domains string) error {
	if _, err := l.env.Files.Stat(domains); err == nil {
		return nil
	} else if !errors.Is(err, filesystem.ErrNotExist) {
		return err
	}
	if err := l.env.Files.MkdirAll(domains, 0755); err != nil {
		return err
	}
	source := filepath.Join(l.config.AssetsDir().String(), "clamp-shared", "allowed-domains.d")
	if _, err := l.env.Files.Stat(source); err == nil {
		if err := l.env.Files.CopyDirectory(source, domains); err != nil {
			return err
		}
	} else if !errors.Is(err, filesystem.ErrNotExist) {
		return err
	}
	_, _ = fmt.Fprintf(l.env.Stdout, "[Clamp Project Setup]\n  Created allowed domains directory: %s\n  Review this directory and delete domain files this project should not allow.\n", domains)
	return nil
}
