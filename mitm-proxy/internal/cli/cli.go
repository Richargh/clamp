package cli

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/richargh/clamp/mitm-proxy/internal/alloweddomains"
	"github.com/richargh/clamp/mitm-proxy/internal/forwardproxy"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/environment"
	filesystem "github.com/richargh/clamp/mitm-proxy/internal/kernel/fs"
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
	var allowedDomainsDir string
	enableFirewall := true
	serve := &cobra.Command{
		Use:   "serve",
		Short: "Forward HTTP and tunnel HTTPS without TLS interception",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			allowed := alloweddomains.AllowAll()
			if enableFirewall {
				if strings.TrimSpace(allowedDomainsDir) == "" {
					return fmt.Errorf("--allowed-domains-dir is required unless --no-firewall is set")
				}
				directory, err := filesystem.ParsePath(allowedDomainsDir)
				if err != nil {
					return fmt.Errorf("allowed domains: %w", err)
				}
				allowed, err = alloweddomains.Load(env.FileSystem, directory)
				if err != nil {
					return err
				}
			}
			handler := forwardproxy.NewHandler(&env, forwardproxy.Options{Proxy: proxyFromEnvironment(ignoreProxyEnv), AllowedDomains: allowed})
			return handler.Serve(cmd.Context(), listenAddress)
		},
	}
	serve.Flags().StringVar(&listenAddress, "listen", "", "Host address and port")
	serve.Flags().BoolVar(&ignoreProxyEnv, "ignore-proxy-env", false, "Ignore host proxy environment variables and connect directly")
	serve.Flags().StringVar(&allowedDomainsDir, "allowed-domains-dir", "", "Folder with domain:port rules in top-level .txt files")
	serve.Flags().BoolFunc("no-firewall", "Allow all destinations; keep request logging active", func(value string) error {
		disabled, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		enableFirewall = !disabled
		return nil
	})
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
