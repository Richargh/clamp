package alloweddomains

import (
	"fmt"
	"net"
	"strings"

	filesystem "github.com/richargh/clamp/mitm-proxy/internal/kernel/fs"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/network"
)

// AllowedDomains is either a restricted domain set or an unrestricted policy.
// The unexported method keeps implementations in this package.
type AllowedDomains interface {
	IsAllowed(network.Address) bool
	IsNotAllowed(network.Address) bool
	isAllowedDomains()
}

type restrictedDomains struct {
	endpoints map[network.Address]struct{}
}

type unrestrictedDomains struct{}

// AllowAll creates an explicit policy that permits every destination.
func AllowAll() AllowedDomains {
	return unrestrictedDomains{}
}

// FromAddresses creates a restricted policy from explicit domain and port pairs.
// No addresses means that every destination is blocked.
func FromAddresses(addresses ...network.Address) (AllowedDomains, error) {
	result := restrictedDomains{endpoints: make(map[network.Address]struct{}, len(addresses))}
	for _, address := range addresses {
		if !isDomainEndpoint(address) {
			return nil, fmt.Errorf("expected domain:port")
		}
		result.endpoints[address] = struct{}{}
	}
	return result, nil
}

// Load reads top-level .txt files once. Blank lines and # comments are ignored.
// An empty directory creates a restricted policy that blocks every destination.
func Load(files filesystem.FileSystem, directory filesystem.Path) (AllowedDomains, error) {
	entries, err := files.ListDirectory(directory)
	if err != nil {
		return nil, fmt.Errorf("allowed domains: %w", err)
	}
	result := restrictedDomains{endpoints: make(map[network.Address]struct{})}
	for _, entry := range entries {
		if entry.IsDirectory || !strings.HasSuffix(entry.Name, ".txt") {
			continue
		}
		path, err := directory.Join(entry.Name)
		if err != nil {
			return nil, fmt.Errorf("allowed domains: %w", err)
		}
		if err := result.loadFile(files, path); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (a restrictedDomains) loadFile(files filesystem.FileSystem, path filesystem.Path) error {
	data, err := files.ReadFile(path)
	if err != nil {
		return fmt.Errorf("allowed domains: %w", err)
	}
	for index, line := range strings.Split(string(data), "\n") {
		line, _, _ = strings.Cut(line, "#")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		endpoint, err := parseAllowedEndpoint(line)
		if err != nil {
			return fmt.Errorf("invalid allowed endpoint in %s:%d (%w)", path.String(), index+1, err)
		}
		a.endpoints[endpoint] = struct{}{}
	}
	return nil
}

func parseAllowedEndpoint(line string) (network.Address, error) {
	address, err := network.ParseAddress(line)
	if err != nil || !isDomainEndpoint(address) {
		return network.Address{}, fmt.Errorf("expected domain:port")
	}
	return address, nil
}

func isDomainEndpoint(address network.Address) bool {
	return address.IsValid() && address.Port() != 0 && net.ParseIP(address.Host().String()) == nil
}

func (a restrictedDomains) IsAllowed(address network.Address) bool {
	_, ok := a.endpoints[address]
	return ok
}

func (a restrictedDomains) IsNotAllowed(address network.Address) bool {
	return !a.IsAllowed(address)
}

func (restrictedDomains) isAllowedDomains() {}

func (unrestrictedDomains) IsAllowed(network.Address) bool { return true }

func (a unrestrictedDomains) IsNotAllowed(address network.Address) bool {
	return !a.IsAllowed(address)
}

func (unrestrictedDomains) isAllowedDomains() {}
