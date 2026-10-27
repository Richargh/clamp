package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/richargh/clamp/mitm-proxy/internal/cli"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/environment"
)

func main() {
	env := environment.NewOS()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := cli.New(*env).Run(ctx, os.Args[1:]); err != nil {
		_, _ = fmt.Fprintf(env.Stderr, "clamp-mitm-proxy: %v\n", err)
		os.Exit(1)
	}
}
