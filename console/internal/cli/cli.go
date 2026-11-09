package cli

import (
	"context"

	"github.com/richargh/clamp/console/internal/config"
	"github.com/richargh/clamp/console/internal/kernel/environment"
	"github.com/spf13/cobra"
)

type CLI struct{ command *cobra.Command }

func New(env environment.Env, settings config.Config) *CLI {
	return &CLI{command: newCommand(env, settings)}
}

func (c *CLI) Run(ctx context.Context, args []string) error {
	c.command.SetArgs(append([]string{}, args...))
	return c.command.ExecuteContext(ctx)
}

func newCommand(env environment.Env, settings config.Config) *cobra.Command {
	root := &cobra.Command{
		Use:               "clamp",
		Short:             "Run a coding agent in a Clamp container",
		SilenceErrors:     true,
		SilenceUsage:      true,
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	}
	root.SetOut(env.Stdout)
	root.SetErr(env.Stderr)
	var version bool
	root.Flags().BoolVarP(&version, "version", "v", false, "Show the Clamp version")
	root.RunE = func(cmd *cobra.Command, _ []string) error {
		if version {
			return printVersion(env, settings.AssetsDir())
		}
		return cmd.Help()
	}
	root.AddCommand(newRunCommand(env, settings))
	return root
}
