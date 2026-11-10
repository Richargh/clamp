package settings

import (
	"fmt"
	"net/netip"
	"regexp"

	"github.com/richargh/clamp/console/internal/kernel/filesystem"
)

// RawUser holds personal settings before parsing.
type RawUser struct {
	AuthStore         string
	ContainerDNS      string
	ContainerTimezone string
}

// User holds personal defaults and the location of per-workspace preferences.
// A preference indexed by workspace is still owned by the user.
type User struct {
	authStore         filesystem.Path
	containerDNS      netip.Addr
	containerTimezone Timezone
}

func ParseUser(raw RawUser) (User, error) {
	auth, err := filesystem.ParsePath(raw.AuthStore)
	if err != nil {
		return User{}, fmt.Errorf("auth store: %w", err)
	}
	if raw.ContainerDNS == "" {
		raw.ContainerDNS = "1.1.1.1"
	}
	dns, err := netip.ParseAddr(raw.ContainerDNS)
	if err != nil {
		return User{}, fmt.Errorf("container DNS: %w", err)
	}
	zone, err := ParseTimezone(raw.ContainerTimezone)
	if err != nil {
		return User{}, err
	}
	return User{authStore: auth, containerDNS: dns, containerTimezone: zone}, nil
}

func (u User) AuthStore() filesystem.Path  { return u.authStore }
func (u User) ContainerDNS() netip.Addr    { return u.containerDNS }
func (u User) ContainerTimezone() Timezone { return u.containerTimezone }

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
