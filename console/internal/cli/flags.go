package cli

import (
	"github.com/richargh/clamp/console/internal/session"
	"github.com/spf13/cobra"
)

// runFlags holds raw command input until the launch boundary parses it.
type runFlags struct {
	options session.Options
	runtime string
}

func (f *runFlags) parse(args []string) (session.Options, error) {
	harness, err := session.ParseHarness(args[0])
	if err != nil {
		return session.Options{}, err
	}
	runtime, err := session.ParseRuntime(f.runtime)
	if err != nil {
		return session.Options{}, err
	}
	options := f.options
	options.Harness = harness
	options.Runtime = runtime
	options.Folder = args[1]
	return options, nil
}

func (f *runFlags) bind(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.StringVar(&f.runtime, "runtime", "docker", "Container runtime: docker or container")
	flags.BoolVar(&f.options.Rebuild, "rebuild", false, "Force rebuild of the container images")
	flags.BoolVar(&f.options.NoCache, "no-cache", false, "Rebuild without the image layer cache")
	flags.BoolVar(&f.options.PerProjectAuth, "per-project-auth", false, "Use separate credentials for this project")
	flags.BoolVar(&f.options.NoFirewall, "no-firewall", false, "Allow all outbound traffic")
	flags.BoolVar(&f.options.Danger, "danger", false, "Run the agent without permission confirmations")
	flags.BoolVar(&f.options.Shell, "shell", false, "Run startup, then open a shell")
}
