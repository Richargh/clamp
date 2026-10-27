package network

import (
	"context"
	"fmt"
	"maps"
	"net"
	"net/netip"
	"slices"
	"strings"
	"sync"
)

type MemoryOptions struct {
	Hosts        map[Host][]netip.Addr
	LookupErrors map[Host]error
	DialErrors   map[Address]error
	ListenErrors map[Address]error
}

type MemoryNetwork struct {
	mu        sync.Mutex
	options   MemoryOptions
	listeners map[netip.AddrPort]*memoryListener
	nextPort  uint16
}

var _ Network = (*MemoryNetwork)(nil)

func NewMemory(options MemoryOptions) *MemoryNetwork {
	hosts := make(map[Host][]netip.Addr, len(options.Hosts))
	for host, addresses := range options.Hosts {
		hosts[host] = slices.Clone(addresses)
	}
	return &MemoryNetwork{
		options: MemoryOptions{
			Hosts: hosts, LookupErrors: maps.Clone(options.LookupErrors),
			DialErrors: maps.Clone(options.DialErrors), ListenErrors: maps.Clone(options.ListenErrors),
		},
		listeners: make(map[netip.AddrPort]*memoryListener),
		nextPort:  49152,
	}
}

func (n *MemoryNetwork) LookupIP(ctx context.Context, host Host) ([]netip.Addr, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !host.IsValid() {
		return nil, fmt.Errorf("invalid lookup host")
	}
	if err := n.options.LookupErrors[host]; err != nil {
		return nil, err
	}
	if addresses, ok := n.options.Hosts[host]; ok {
		for _, address := range addresses {
			if !address.IsValid() || address.Zone() != "" {
				return nil, fmt.Errorf("invalid configured IP address")
			}
		}
		return slices.Clone(addresses), nil
	}
	if ip, err := netip.ParseAddr(host.String()); err == nil {
		return []netip.Addr{ip}, nil
	}
	if strings.TrimSuffix(host.String(), ".") == "localhost" {
		return []netip.Addr{netip.MustParseAddr("127.0.0.1"), netip.IPv6Loopback()}, nil
	}
	return nil, &net.DNSError{Err: "no such host", Name: host.String(), IsNotFound: true}
}

func (n *MemoryNetwork) endpoint(ctx context.Context, address Address) (netip.AddrPort, error) {
	if !address.IsValid() {
		return netip.AddrPort{}, fmt.Errorf("invalid address")
	}
	addresses, err := n.LookupIP(ctx, address.Host())
	if err != nil {
		return netip.AddrPort{}, err
	}
	if len(addresses) == 0 {
		return netip.AddrPort{}, fmt.Errorf("no destination addresses")
	}
	return netip.AddrPortFrom(addresses[0].Unmap(), address.Port()), nil
}

func (n *MemoryNetwork) Listen(address Address) (net.Listener, error) {
	if !address.IsValid() {
		return nil, fmt.Errorf("invalid listen address")
	}
	if err := n.options.ListenErrors[address]; err != nil {
		return nil, err
	}
	endpoint, err := n.endpoint(context.Background(), address)
	if err != nil {
		return nil, err
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if endpoint.Port() == 0 {
		endpoint, err = n.allocateEndpoint(endpoint.Addr())
		if err != nil {
			return nil, err
		}
	}
	if n.listeners[endpoint] != nil {
		return nil, fmt.Errorf("address already in use")
	}
	listener := &memoryListener{
		network: n, address: endpoint,
		pending: make(chan net.Conn, 64), closed: make(chan struct{}),
	}
	n.listeners[endpoint] = listener
	return listener, nil
}

func (n *MemoryNetwork) Dial(ctx context.Context, address Address) (net.Conn, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !address.IsValid() || address.Port() == 0 {
		return nil, fmt.Errorf("invalid dial address")
	}
	if err := n.options.DialErrors[address]; err != nil {
		return nil, err
	}
	endpoint, err := n.endpoint(ctx, address)
	if err != nil {
		return nil, err
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	listener := n.listeners[endpoint]
	if listener == nil {
		return nil, fmt.Errorf("connection refused")
	}
	local, err := n.allocateEndpoint(endpoint.Addr())
	if err != nil {
		return nil, err
	}
	client, server := net.Pipe()
	clientConn := &memoryConn{Conn: client, local: local, remote: endpoint}
	serverConn := &memoryConn{Conn: server, local: endpoint, remote: local}
	select {
	case listener.pending <- serverConn:
		return clientConn, nil
	default:
		_ = client.Close()
		_ = server.Close()
		return nil, fmt.Errorf("listener backlog full")
	}
}

func (n *MemoryNetwork) allocateEndpoint(ip netip.Addr) (netip.AddrPort, error) {
	for range 16384 {
		endpoint := netip.AddrPortFrom(ip, n.nextPort)
		n.nextPort++
		if n.nextPort == 0 {
			n.nextPort = 49152
		}
		if n.listeners[endpoint] == nil {
			return endpoint, nil
		}
	}
	return netip.AddrPort{}, fmt.Errorf("no available ports")
}

type memoryAddr struct{ value netip.AddrPort }

func (a memoryAddr) Network() string { return "tcp" }
func (a memoryAddr) String() string  { return a.value.String() }

type memoryConn struct {
	net.Conn
	local  netip.AddrPort
	remote netip.AddrPort
}

func (c *memoryConn) LocalAddr() net.Addr  { return memoryAddr{c.local} }
func (c *memoryConn) RemoteAddr() net.Addr { return memoryAddr{c.remote} }

type memoryListener struct {
	network *MemoryNetwork
	address netip.AddrPort
	pending chan net.Conn
	closed  chan struct{}
}

func (l *memoryListener) Addr() net.Addr { return memoryAddr{l.address} }

func (l *memoryListener) Accept() (net.Conn, error) {
	select {
	case <-l.closed:
		return nil, net.ErrClosed
	default:
	}
	select {
	case conn := <-l.pending:
		return conn, nil
	case <-l.closed:
		return nil, net.ErrClosed
	}
}

func (l *memoryListener) Close() error {
	l.network.mu.Lock()
	defer l.network.mu.Unlock()
	select {
	case <-l.closed:
		return net.ErrClosed
	default:
	}
	delete(l.network.listeners, l.address)
	close(l.closed)
	for {
		select {
		case conn := <-l.pending:
			_ = conn.Close()
		default:
			return nil
		}
	}
}
