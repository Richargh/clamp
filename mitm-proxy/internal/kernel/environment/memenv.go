package environment

import (
	"bytes"
	"crypto/rand"

	"github.com/richargh/clamp/mitm-proxy/internal/kernel/fs"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/network"
)

func NewMemory() *Env {
	return &Env{
		Random: rand.Reader,
		PID:    1, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{},
		Network:    network.NewMemory(network.MemoryOptions{}),
		FileSystem: fs.NewMemory(fs.MemoryOptions{}),
	}
}
