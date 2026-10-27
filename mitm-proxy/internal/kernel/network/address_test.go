package network_test

import (
	"net/netip"
	"strings"
	"testing"

	"github.com/richargh/clamp/mitm-proxy/internal/kernel/network"
)

func TestHostConstruction(t *testing.T) {
	for _, test := range []struct{ input, want string }{
		{"Example.COM", "example.com"},
		{"example.com.", "example.com."},
		{"localhost", "localhost"},
		{"127.0.0.1", "127.0.0.1"},
		{"2001:0db8::1", "2001:db8::1"},
	} {
		t.Run(test.input, func(t *testing.T) {
			host := mustHost(t, test.input)
			if !host.IsValid() || host.String() != test.want {
				t.Fatalf("host = %q, want %q", host, test.want)
			}
		})
	}
	for _, value := range []string{"", " ", "https://example.com", "example.com/path", "user@example.com", "example.com:443", "[::1]", "fe80::1%lo0", ".", "a..b", "-example.com", "example-.com", strings.Repeat("a", 64) + ".com"} {
		t.Run("invalid/"+value, func(t *testing.T) {
			host, err := network.NewHost(value)
			if err == nil || host.IsValid() {
				t.Fatalf("host(%q) = %q, %v; want an error", value, host, err)
			}
		})
	}
}

func TestAddressFromIP(t *testing.T) {
	for _, test := range []struct {
		ip   string
		port uint16
		want string
	}{
		{"8.8.8.8", 443, "8.8.8.8:443"},
		{"2001:0db8::1", 65535, "[2001:db8::1]:65535"},
		{"127.0.0.1", 0, "127.0.0.1:0"},
	} {
		t.Run(test.want, func(t *testing.T) {
			// given
			ip := netip.MustParseAddr(test.ip)

			// when
			testee, err := network.AddressFromIP(ip, test.port)

			// then
			if err != nil {
				t.Fatal(err)
			}
			if !testee.IsValid() || testee.String() != test.want || testee.Host().String() != ip.String() || testee.Port() != test.port {
				t.Fatalf("address = %q, want %q", testee, test.want)
			}
		})
	}
}

func TestAddressFromIPRejectsInvalidIP(t *testing.T) {
	for _, ip := range []netip.Addr{{}, netip.MustParseAddr("fe80::1%lo0")} {
		t.Run(ip.String(), func(t *testing.T) {
			// given
			const port = 443

			// when
			testee, err := network.AddressFromIP(ip, port)

			// then
			if err == nil || testee.IsValid() {
				t.Fatalf("address = %q, %v; want an error", testee, err)
			}
		})
	}
}

func TestAddressConstruction(t *testing.T) {
	for _, value := range []string{"example.com:443", "127.0.0.1:0", "[::1]:65535"} {
		t.Run(value, func(t *testing.T) {
			address := mustAddress(t, value)
			if !address.IsValid() || address.String() != value {
				t.Fatalf("address = %q, want %q", address, value)
			}
			rebuilt, err := network.ParseAddress(address.String())
			if err != nil || rebuilt != address {
				t.Fatalf("rebuilt address = %q, %v; want %q", rebuilt, err, address)
			}
		})
	}
	for _, value := range []string{"", "example.com", "example.com:", "example.com:https", "example.com:-1", "example.com:65536", ":443", "https://example.com:443", "::1:443"} {
		t.Run("invalid/"+value, func(t *testing.T) {
			address, err := network.ParseAddress(value)
			if err == nil || address.IsValid() {
				t.Fatalf("address(%q) = %q, %v; want an error", value, address, err)
			}
		})
	}
	if (network.Host{}).IsValid() || (network.Address{}).IsValid() {
		t.Fatal("zero values must not be valid")
	}
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
