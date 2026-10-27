package acceptance_test

import (
	"context"
	"net"
	"net/netip"

	"github.com/richargh/clamp/mitm-proxy/internal/kernel/environment"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/network"
)

type localOriginNetwork struct {
	network.Network
}

func (n localOriginNetwork) LookupIP(ctx context.Context, host network.Host) ([]netip.Addr, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if host.String() == "example.com" {
		return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
	}
	return n.Network.LookupIP(ctx, host)
}

func FailDial(env *environment.Env, target network.Address, err error) {
	env.Network = failingDialNetwork{Network: env.Network, target: target, err: err}
}

type failingDialNetwork struct {
	network.Network
	target network.Address
	err    error
}

func (n failingDialNetwork) Dial(ctx context.Context, address network.Address) (net.Conn, error) {
	if address == n.target {
		return nil, n.err
	}
	return n.Network.Dial(ctx, address)
}
