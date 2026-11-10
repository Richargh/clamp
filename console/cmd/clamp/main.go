package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/richargh/clamp/console/internal/cli"
	"github.com/richargh/clamp/console/internal/config"
	"github.com/richargh/clamp/console/internal/kernel/environment"
	"github.com/richargh/clamp/console/internal/kernel/host"
	"github.com/richargh/clamp/console/internal/settings"
)

func main() {
	env := environment.NewOS()
	settings, err := startupConfig()
	if err != nil {
		_, _ = fmt.Fprintln(env.Stderr, err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := cli.New(env, settings).Run(ctx, os.Args[1:]); err != nil {
		_, _ = fmt.Fprintln(env.Stderr, err)
		var exit interface{ ExitCode() int }
		if errors.As(err, &exit) && exit.ExitCode() > 0 {
			os.Exit(exit.ExitCode())
		}
		os.Exit(1)
	}
}

// Resolve host settings once at the composition root, not inside services.
func startupConfig() (config.Config, error) {
	_, source, _, _ := runtime.Caller(0)
	assets, err := host.AssetDirectory("clamp-base.Dockerfile", source)
	if err != nil {
		return config.Config{}, fmt.Errorf("cannot find Clamp assets; keep the executable in the Clamp asset directory or a subdirectory: %w", err)
	}
	directory, err := host.ConfigDirectory()
	if err != nil {
		return config.Config{}, err
	}
	user, err := settings.BootstrapUser(directory, os.Getenv("CLAMP_CONTAINER_DNS"), host.Timezone())
	if err != nil {
		return config.Config{}, err
	}
	return config.New(assets, user)
}
