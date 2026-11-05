package forwardproxy

import (
	"context"
	"fmt"

	"github.com/richargh/clamp/mitm-proxy/internal/kernel/environment"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/network"
)

func resolveIP(ctx context.Context, env *environment.Env, address network.Address) (network.Address, error) {
	addresses, err := env.Network.LookupIP(ctx, address.Host())
	if err != nil || len(addresses) == 0 {
		return network.Address{}, fmt.Errorf("dns failure")
	}
	return network.AddressFromIP(addresses[0], address.Port())
}
