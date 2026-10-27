package network

import (
	"context"
	"net"
	"net/netip"
)

type Dialer func(ctx context.Context, address Address) (net.Conn, error)

type Network interface {
	Listen(address Address) (net.Listener, error)
	Dial(ctx context.Context, address Address) (net.Conn, error)
	LookupIP(ctx context.Context, host Host) ([]netip.Addr, error)
}
