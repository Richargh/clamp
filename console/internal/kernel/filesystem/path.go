package filesystem

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Path is an absolute, clean path. Parsing does not access the file system.
type Path struct{ value string }

func ParsePath(raw string) (Path, error) {
	if !filepath.IsAbs(raw) || strings.ContainsRune(raw, '\x00') {
		return Path{}, fmt.Errorf("expected an absolute file path: %q", raw)
	}
	return Path{value: filepath.Clean(raw)}, nil
}

func (p Path) String() string { return p.value }
