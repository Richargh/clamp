package network

import (
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
)

type Address struct {
	host Host
	port uint16
}

func newAddress(host Host, port uint16) (Address, error) {
	if !host.IsValid() {
		return Address{}, fmt.Errorf("invalid address host")
	}
	return Address{host: host, port: port}, nil
}

func AddressFromIP(ip netip.Addr, port uint16) (Address, error) {
	if !ip.IsValid() || ip.Zone() != "" {
		return Address{}, fmt.Errorf("invalid address IP")
	}
	return newAddress(Host{value: ip.String()}, port)
}

func AddressFromURL(value *url.URL) (Address, error) {
	if value == nil {
		return Address{}, fmt.Errorf("missing address URL")
	}
	host, err := NewHost(value.Hostname())
	if err != nil {
		return Address{}, err
	}
	if strings.HasSuffix(value.Host, ":") {
		return Address{}, fmt.Errorf("invalid address port")
	}
	portText := value.Port()
	if portText == "" {
		switch strings.ToLower(value.Scheme) {
		case "http":
			portText = "80"
		case "https":
			portText = "443"
		default:
			return Address{}, fmt.Errorf("no default address port for URL scheme")
		}
	}
	port, err := strconv.ParseUint(portText, 10, 16)
	if err != nil || port == 0 {
		return Address{}, fmt.Errorf("invalid address port")
	}
	return newAddress(host, uint16(port))
}

func ParseAddress(value string) (Address, error) {
	hostText, portText, err := net.SplitHostPort(value)
	if err != nil {
		return Address{}, fmt.Errorf("invalid address")
	}
	host, err := NewHost(hostText)
	if err != nil {
		return Address{}, err
	}
	port, err := strconv.ParseUint(portText, 10, 16)
	if err != nil {
		return Address{}, fmt.Errorf("invalid address port")
	}
	return newAddress(host, uint16(port))
}

func (a Address) Host() Host { return a.host }

func (a Address) Port() uint16 { return a.port }

func (a Address) String() string {
	return net.JoinHostPort(a.host.String(), strconv.FormatUint(uint64(a.port), 10))
}

func (a Address) IsValid() bool { return a.host.IsValid() }
