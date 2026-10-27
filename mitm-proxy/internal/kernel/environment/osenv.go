package environment

import (
	"crypto/rand"
	"os"

	"github.com/richargh/clamp/mitm-proxy/internal/kernel"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/network"
)

func NewOS() *Env {
	return &Env{
		Version: kernel.ParseClampVersion(kernel.BuildVersion),
		Random:  rand.Reader,
		PID:     os.Getpid(), Stdout: os.Stdout, Stderr: os.Stderr,
		Network: network.NewOS(),
	}
}
