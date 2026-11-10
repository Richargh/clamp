package config

import (
	"fmt"

	"github.com/richargh/clamp/console/internal/kernel/filesystem"
	"github.com/richargh/clamp/console/internal/settings"
)

// Raw holds installation and user inputs before parsing.
// Project settings are resolved separately for each workspace.
type Raw struct {
	AssetsDir         string
	AuthStore         string
	ContainerDNS      string
	ContainerTimezone string
}

// Config holds installation data and user settings, not project settings or host capabilities.
type Config struct {
	assetsDir filesystem.Path
	user      settings.User
}

func Parse(raw Raw) (Config, error) {
	user, err := settings.ParseUser(settings.RawUser{
		AuthStore:         raw.AuthStore,
		ContainerDNS:      raw.ContainerDNS,
		ContainerTimezone: raw.ContainerTimezone,
	})
	if err != nil {
		return Config{}, err
	}
	return New(raw.AssetsDir, user)
}

// New combines installation data with parsed user settings.
func New(assetsDir string, user settings.User) (Config, error) {
	assets, err := filesystem.ParsePath(assetsDir)
	if err != nil {
		return Config{}, fmt.Errorf("clamp assets: %w", err)
	}
	return Config{assetsDir: assets, user: user}, nil
}

func (c Config) AssetsDir() filesystem.Path { return c.assetsDir }
func (c Config) User() settings.User        { return c.user }
