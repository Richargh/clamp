package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/richargh/clamp/console/internal/kernel/environment"
	"github.com/richargh/clamp/console/internal/kernel/filesystem"
)

func printVersion(env environment.Env, assets filesystem.Path) error {
	data, err := env.Files.ReadFile(filepath.Join(assets.String(), "VERSION"))
	if err != nil {
		return fmt.Errorf("missing version file: %w", err)
	}
	version := strings.Split(strings.TrimSpace(string(data)), "\n")[0]
	_, err = fmt.Fprintf(env.Stdout, "Clamp %s\n", version)
	return err
}
