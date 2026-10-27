package network

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"time"
)

type OSNetwork struct {
	dialer net.Dialer
}

var _ Network = (*OSNetwork)(nil)

func NewOS() *OSNetwork {
	return &OSNetwork{dialer: net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}}
}

func (n *OSNetwork) Listen(address Address) (net.Listener, error) {
	if !address.IsValid() {
		return nil, fmt.Errorf("invalid listen address")
	}
	return net.Listen("tcp", address.String())
}

func (n *OSNetwork) Dial(ctx context.Context, address Address) (net.Conn, error) {
	if !address.IsValid() || address.Port() == 0 {
		return nil, fmt.Errorf("invalid dial address")
	}
	return n.dialer.DialContext(ctx, "tcp", address.String())
}

func (n *OSNetwork) LookupIP(ctx context.Context, host Host) ([]netip.Addr, error) {
	if !host.IsValid() {
		return nil, fmt.Errorf("invalid lookup host")
	}
	return net.DefaultResolver.LookupNetIP(ctx, "ip", host.String())
}
