package config

import (
	"fmt"
	"net/netip"
	"regexp"

	"github.com/richargh/clamp/console/internal/kernel/filesystem"
)

// Raw holds startup values before parsing. It does not read the environment.
type Raw struct {
	AssetsDir         string
	AuthStore         string
	ContainerDNS      string
	ContainerTimezone string
}

// Config holds parsed application settings, not host capabilities.
type Config struct {
	assetsDir         filesystem.Path
	authStore         filesystem.Path
	containerDNS      netip.Addr
	containerTimezone Timezone
}

func Parse(raw Raw) (Config, error) {
	assets, err := filesystem.ParsePath(raw.AssetsDir)
	if err != nil {
		return Config{}, fmt.Errorf("clamp assets: %w", err)
	}
	auth, err := filesystem.ParsePath(raw.AuthStore)
	if err != nil {
		return Config{}, fmt.Errorf("auth store: %w", err)
	}
	if raw.ContainerDNS == "" {
		raw.ContainerDNS = "1.1.1.1"
	}
	dns, err := netip.ParseAddr(raw.ContainerDNS)
	if err != nil {
		return Config{}, fmt.Errorf("container DNS: %w", err)
	}
	zone, err := ParseTimezone(raw.ContainerTimezone)
	if err != nil {
		return Config{}, err
	}
	return Config{
		assetsDir: assets, authStore: auth, containerDNS: dns, containerTimezone: zone,
	}, nil
}

func (c Config) AssetsDir() filesystem.Path  { return c.assetsDir }
func (c Config) AuthStore() filesystem.Path  { return c.authStore }
func (c Config) ContainerDNS() netip.Addr    { return c.containerDNS }
func (c Config) ContainerTimezone() Timezone { return c.containerTimezone }

// Timezone is an optional TZ setting. An empty value leaves the container default.
type Timezone struct{ value string }

var timezonePattern = regexp.MustCompile(`^[A-Za-z0-9_+./-]+$`)

func ParseTimezone(raw string) (Timezone, error) {
	if raw != "" && !timezonePattern.MatchString(raw) {
		return Timezone{}, fmt.Errorf("invalid container timezone: %q", raw)
	}
	return Timezone{value: raw}, nil
}

func (t Timezone) String() string { return t.value }
