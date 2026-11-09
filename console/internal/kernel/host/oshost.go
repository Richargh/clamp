// Package host discovers startup values. Application services do not use it.
package host

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AssetDirectory searches ancestors of the executable and source directories.
// It does not search the working directory: project files are not trusted assets.
// The application supplies its own marker.
func AssetDirectory(marker, source string) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("cannot find executable directory: %w", err)
	}
	for _, start := range []string{filepath.Dir(exe), filepath.Dir(source)} {
		if !filepath.IsAbs(start) {
			continue
		}
		for path := start; path != ""; path = filepath.Dir(path) {
			if info, err := os.Stat(filepath.Join(path, marker)); err == nil && !info.IsDir() {
				return path, nil
			}
			if filepath.Dir(path) == path {
				break
			}
		}
	}
	return "", fmt.Errorf("cannot find asset directory containing %s", marker)
}

func ConfigDirectory() (string, error) {
	directory := os.Getenv("XDG_CONFIG_HOME")
	if directory == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		directory = filepath.Join(home, ".config")
	}
	return filepath.Abs(directory)
}

// Timezone returns the host TZ setting, or a value from its timezone files.
// The application parses this value before use.
func Timezone() string {
	zone := os.Getenv("TZ")
	if zone == "" {
		if data, err := os.ReadFile("/etc/timezone"); err == nil {
			zone = strings.Split(strings.TrimSpace(string(data)), "\n")[0]
		}
	}
	if zone == "" {
		if link, err := os.Readlink("/etc/localtime"); err == nil {
			if _, tail, ok := strings.Cut(link, "zoneinfo/"); ok {
				zone = tail
			}
		}
	}
	return zone
}
