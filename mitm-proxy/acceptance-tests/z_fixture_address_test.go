package acceptance_test

import (
	"testing"

	"github.com/richargh/clamp/mitm-proxy/internal/kernel/network"
)

func MustAddress(t *testing.T, value string) network.Address {
	t.Helper()
	address, err := network.ParseAddress(value)
	if err != nil {
		t.Fatal(err)
	}
	return address
}

func MustHost(t *testing.T, value string) network.Host {
	t.Helper()
	host, err := network.NewHost(value)
	if err != nil {
		t.Fatal(err)
	}
	return host
}
