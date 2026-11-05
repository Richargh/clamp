package network

import (
	"fmt"
	"net/netip"
	"strings"
)

type Host struct {
	value string
}

func NewHost(value string) (Host, error) {
	if ip, err := netip.ParseAddr(value); err == nil {
		if ip.Zone() != "" {
			return Host{}, fmt.Errorf("scoped IP addresses are not supported")
		}
		return Host{value: ip.String()}, nil
	}
	name := strings.TrimSuffix(value, ".")
	if name == "" || len(name) > 253 {
		return Host{}, fmt.Errorf("invalid host")
	}
	for _, label := range strings.Split(name, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return Host{}, fmt.Errorf("invalid host")
		}
		for _, char := range label {
			letter := char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z'
			digit := char >= '0' && char <= '9'
			if !letter && !digit && char != '-' {
				return Host{}, fmt.Errorf("invalid host")
			}
		}
	}
	return Host{value: strings.ToLower(name)}, nil
}

func (h Host) String() string { return h.value }

func (h Host) IsValid() bool { return h.value != "" }
