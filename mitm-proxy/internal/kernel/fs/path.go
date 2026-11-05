package fs

import (
	stdfs "io/fs"
	"path/filepath"
	"strings"
)

// Path is a normalized host path. Its zero value is invalid.
type Path struct {
	value string
}

func ParsePath(value string) (Path, error) {
	if strings.TrimSpace(value) == "" || strings.ContainsRune(value, '\x00') {
		return Path{}, stdfs.ErrInvalid
	}
	return Path{value: filepath.Clean(value)}, nil
}

func (p Path) String() string { return p.value }

func (p Path) IsValid() bool { return p.value != "" }

func (p Path) Join(name string) (Path, error) {
	if !p.IsValid() || strings.ContainsRune(name, '\x00') {
		return Path{}, stdfs.ErrInvalid
	}
	return ParsePath(filepath.Join(p.value, name))
}
