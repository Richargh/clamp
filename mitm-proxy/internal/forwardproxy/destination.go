package forwardproxy

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/richargh/clamp/mitm-proxy/internal/kernel/environment"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/network"
)

func splitAuthority(authority string, defaultPort int) (string, int, error) {
	if authority == "" || strings.ContainsAny(authority, "\x00\r\n\t /@") {
		return "", 0, fmt.Errorf("invalid authority")
	}

	host, portText, err := net.SplitHostPort(authority)
	if err != nil {
		if strings.Contains(authority, ":") {
			if ip := net.ParseIP(authority); ip != nil {
				return authority, defaultPort, nil
			}
			return "", 0, fmt.Errorf("invalid authority")
		}
		host, portText = authority, strconv.Itoa(defaultPort)
	}
	if host == "" {
		return "", 0, fmt.Errorf("invalid authority")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return "", 0, fmt.Errorf("invalid port")
	}
	return strings.ToLower(strings.TrimSuffix(host, ".")), port, nil
}

func resolveIP(ctx context.Context, env *environment.Env, address network.Address) (network.Address, error) {
	addresses, err := env.Network.LookupIP(ctx, address.Host())
	if err != nil || len(addresses) == 0 {
		return network.Address{}, fmt.Errorf("dns failure")
	}
	return network.AddressFromIP(addresses[0], address.Port())
}
