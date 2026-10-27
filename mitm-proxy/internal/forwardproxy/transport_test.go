package forwardproxy

import (
	"context"
	"errors"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"testing"

	"github.com/richargh/clamp/mitm-proxy/internal/kernel/environment"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/network"
)

func TestTransportRejectsInvalidDestination(t *testing.T) {
	for _, address := range []string{"example.com:0", "example.com", ":443", "example.com:65536"} {
		t.Run(address, func(t *testing.T) {
			// given
			testee := NewTransport(environment.NewMemory(), TransportOptions{Proxy: directProxy})
			defer testee.CloseIdleConnections()

			// when
			conn, err := testee.DialContext(context.Background(), "tcp", address)

			// then
			if conn != nil {
				_ = conn.Close()
				t.Fatal("invalid destination must not connect")
			}
			if err == nil || !strings.Contains(err.Error(), "invalid") {
				t.Fatalf("dial = %v, want invalid destination rejection", err)
			}
		})
	}
}

func TestResolveIP(t *testing.T) {
	for _, test := range []struct {
		name string
		ips  []netip.Addr
		want string
	}{
		{"public", []netip.Addr{netip.MustParseAddr("8.8.8.8")}, "8.8.8.8:443"},
		{"private", []netip.Addr{netip.MustParseAddr("10.0.0.1")}, "10.0.0.1:443"},
		{"loopback", []netip.Addr{netip.MustParseAddr("127.0.0.1")}, "127.0.0.1:443"},
		{"IPv6", []netip.Addr{netip.MustParseAddr("::1")}, "[::1]:443"},
		{"mixed", []netip.Addr{netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("127.0.0.1")}, "8.8.8.8:443"},
	} {
		t.Run(test.name, func(t *testing.T) {
			// given
			env := environment.NewMemory()
			address := mustAddress(t, "example.com:443")
			env.Network = network.NewMemory(network.MemoryOptions{
				Hosts: map[network.Host][]netip.Addr{address.Host(): test.ips},
			})

			// when
			testee, err := resolveIP(context.Background(), env, address)

			// then
			if err != nil {
				t.Fatal(err)
			}
			if testee != mustAddress(t, test.want) {
				t.Fatalf("destination = %s, want %s", testee, test.want)
			}
		})
	}
}

func TestResolveIPHandlesDNSFailure(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
	}{
		{"empty answer", nil},
		{"lookup failure", errors.New("lookup failed")},
	} {
		t.Run(test.name, func(t *testing.T) {
			// given
			env := environment.NewMemory()
			address := mustAddress(t, "example.com:443")
			env.Network = network.NewMemory(network.MemoryOptions{
				Hosts:        map[network.Host][]netip.Addr{address.Host(): nil},
				LookupErrors: map[network.Host]error{address.Host(): test.err},
			})

			// when
			testee, err := resolveIP(context.Background(), env, address)

			// then
			if err == nil || err.Error() != "dns failure" || testee.IsValid() {
				t.Fatalf("resolve = %s, %v; want DNS failure", testee, err)
			}
		})
	}
}

func TestTransportUsesInjectedNetwork(t *testing.T) {
	for _, ip := range []string{"8.8.8.8", "10.0.0.1", "127.0.0.1", "::1"} {
		t.Run(ip, func(t *testing.T) {
			// given
			stop := errors.New("test dial")
			env := environment.NewMemory()
			resolved := netip.MustParseAddr(ip)
			target := mustAddress(t, netip.AddrPortFrom(resolved, 443).String())
			env.Network = network.NewMemory(network.MemoryOptions{
				Hosts: map[network.Host][]netip.Addr{
					mustHost(t, "example.com"): {resolved},
				},
				DialErrors: map[network.Address]error{target: stop},
			})
			testee := NewTransport(env, TransportOptions{Proxy: directProxy})
			defer testee.CloseIdleConnections()

			// when
			_, err := testee.DialContext(context.Background(), "tcp", "example.com:443")

			// then
			proxy, proxyErr := testee.Proxy(&http.Request{URL: &url.URL{Scheme: "https", Host: "example.com"}})
			if proxyErr != nil || proxy != nil {
				t.Fatalf("proxy = %v, %v; want a direct connection", proxy, proxyErr)
			}
			if !errors.Is(err, stop) {
				t.Fatalf("dial = %v, want injected dial error", err)
			}
		})
	}
}

func directProxy(*url.URL) (*url.URL, error) {
	return nil, nil
}

func mustHost(t *testing.T, value string) network.Host {
	t.Helper()
	host, err := network.NewHost(value)
	if err != nil {
		t.Fatal(err)
	}
	return host
}

func mustAddress(t *testing.T, value string) network.Address {
	t.Helper()
	address, err := network.ParseAddress(value)
	if err != nil {
		t.Fatal(err)
	}
	return address
}
