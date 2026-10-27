package acceptance_test

import (
	"net/netip"
	"testing"

	"github.com/richargh/clamp/mitm-proxy/internal/kernel/environment"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/network"
)

func ForEachEnvironment(t *testing.T, behavior string, scenario func(*testing.T, *environment.Env)) {
	t.Helper()
	for _, adapter := range []struct {
		name string
		new  func(*testing.T) *environment.Env
	}{
		{"in-memory environment", NewMemoryEnvironment},
		{"OS environment", NewOSEnvironment},
	} {
		t.Run(behavior+" with an "+adapter.name, func(t *testing.T) {
			scenario(t, adapter.new(t))
		})
	}
}

func NewMemoryEnvironment(t *testing.T) *environment.Env {
	t.Helper()
	return MemoryEnvironmentAtIP(t, "127.0.0.1")
}

func MemoryEnvironmentAtIP(t *testing.T, ip string) *environment.Env {
	t.Helper()
	env := environment.NewMemory()
	env.Stdout = &LogBuffer{}
	env.Network = network.NewMemory(network.MemoryOptions{
		Hosts: map[network.Host][]netip.Addr{
			MustHost(t, "example.com"): {netip.MustParseAddr(ip)},
		},
	})
	return env
}

func NewOSEnvironment(t *testing.T) *environment.Env {
	t.Helper()
	env := environment.NewOS()
	env.Stdout = &LogBuffer{}
	// Keep the origin alias local without bypassing host proxy selection.
	env.Network = localOriginNetwork{Network: env.Network}
	return env
}
