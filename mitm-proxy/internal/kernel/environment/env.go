package environment

import (
	"io"

	"github.com/richargh/clamp/mitm-proxy/internal/kernel"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/network"
)

type Env struct {
	Version kernel.ClampVersion
	Random  io.Reader
	PID     int
	Stdout  io.Writer
	Stderr  io.Writer
	Network network.Network
}
