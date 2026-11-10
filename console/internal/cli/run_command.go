package cli

import (
	"github.com/richargh/clamp/console/internal/config"
	"github.com/richargh/clamp/console/internal/kernel/environment"
	"github.com/richargh/clamp/console/internal/session"
	"github.com/spf13/cobra"
)

func newRunCommand(env environment.Env, settings config.Config) *cobra.Command {
	input := &runFlags{}
	cmd := &cobra.Command{
		Use:   "run <harness> <folder>",
		Short: "Run a coding agent in a Clamp container",
		Long: "Run a coding agent in a Clamp container. Harness: pi, claude, or opencode.\n\n" +
			"CLAMP_CONTAINER_DNS: Apple container DNS server (default: 1.1.1.1).",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			options, err := input.parse(args)
			if err != nil {
				return err
			}
			selectHarnessVolume := func(workspace session.Workspace) (session.HarnessVolumeChoice, error) {
				return promptHarnessStorage(env, workspace.String())
			}
			return session.Launch(cmd.Context(), env, settings, options, selectHarnessVolume)
		},
	}
	input.bind(cmd)
	return cmd
}
