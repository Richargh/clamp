package cli

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/richargh/clamp/mitm-proxy/internal/forwardproxy"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/environment"
	"github.com/spf13/cobra"
	"golang.org/x/net/http/httpproxy"
)

type CLI struct{ command *cobra.Command }

func New(env environment.Env) *CLI { return &CLI{command: newCommand(env)} }

func (c *CLI) Run(ctx context.Context, args []string) error {
	c.command.SetArgs(append([]string{}, args...))
	return c.command.ExecuteContext(ctx)
}

func newCommand(env environment.Env) *cobra.Command {
	root := &cobra.Command{
		Use:           "clamp-mitm-proxy",
		Version:       env.Version.String(),
		Short:         "Host-side HTTP forward proxy with HTTPS tunneling",
		SilenceErrors: true, // The entry point reports errors once.
		SilenceUsage:  true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
	}

	root.SetOut(env.Stdout)
	root.SetErr(env.Stderr)

	configureCommands(root, env)
	return root
}

func configureCommands(root *cobra.Command, env environment.Env) {
	var listenAddress string
	var ignoreProxyEnv bool
	serve := &cobra.Command{
		Use:   "serve",
		Short: "Forward HTTP and tunnel HTTPS without TLS interception",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			handler := forwardproxy.NewHandler(&env, forwardproxy.Options{Proxy: proxyFromEnvironment(ignoreProxyEnv)})
			return handler.Serve(cmd.Context(), listenAddress)
		},
	}
	serve.Flags().StringVar(&listenAddress, "listen", "", "Host address and port")
	serve.Flags().BoolVar(&ignoreProxyEnv, "ignore-proxy-env", false, "Ignore host proxy environment variables and connect directly")
	markNonBlankFlags(serve, "listen")

	root.AddCommand(serve)
}

func proxyFromEnvironment(ignore bool) func(*url.URL) (*url.URL, error) {
	if ignore {
		return func(*url.URL) (*url.URL, error) { return nil, nil }
	}

	return httpproxy.FromEnvironment().ProxyFunc()
}

func markNonBlankFlags(cmd *cobra.Command, names ...string) {
	for _, name := range names {
		if err := cmd.MarkFlagRequired(name); err != nil {
			panic(err)
		}
	}
	cmd.PreRunE = func(cmd *cobra.Command, _ []string) error {
		return validateNotBlank(cmd, names...)
	}
}

func validateNotBlank(cmd *cobra.Command, names ...string) error {
	for _, name := range names {
		flag := cmd.Flags().Lookup(name)
		if flag.Changed && strings.TrimSpace(flag.Value.String()) == "" {
			return fmt.Errorf("--%s must not be blank", name)
		}
	}
	return nil
}
