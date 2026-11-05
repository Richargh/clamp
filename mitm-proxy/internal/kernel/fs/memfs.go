package fs

import (
	"bytes"
	stdfs "io/fs"
	"path/filepath"
	"sort"
	"strings"
)

type MemoryOptions struct {
	Files map[Path][]byte
}

// Memory infers directories from file paths. It cannot represent empty directories.
type Memory struct {
	files map[Path][]byte
}

func NewMemory(options MemoryOptions) *Memory {
	files := make(map[Path][]byte, len(options.Files))
	for path, contents := range options.Files {
		files[path] = bytes.Clone(contents)
	}
	return &Memory{files: files}
}

func (m *Memory) ReadFile(path Path) ([]byte, error) {
	if !path.IsValid() {
		return nil, stdfs.ErrInvalid
	}
	if contents, ok := m.files[path]; ok {
		return bytes.Clone(contents), nil
	}
	isDirectory, err := m.IsDirectory(path)
	if err != nil {
		return nil, err
	}
	if isDirectory {
		return nil, ErrIsDirectory
	}
	return nil, stdfs.ErrNotExist
}

func (m *Memory) ListDirectory(path Path) ([]Entry, error) {
	isDirectory, err := m.IsDirectory(path)
	if err != nil {
		return nil, err
	}
	if !isDirectory {
		return nil, ErrNotDirectory
	}
	children := make(map[string]Entry)
	for file := range m.files {
		name, directory, ok := directoryChild(path, file)
		if ok {
			children[name] = Entry{Name: name, IsDirectory: directory}
		}
	}
	result := make([]Entry, 0, len(children))
	for _, entry := range children {
		result = append(result, entry)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func (m *Memory) IsDirectory(path Path) (bool, error) {
	if !path.IsValid() {
		return false, stdfs.ErrInvalid
	}
	if _, ok := m.files[path]; ok {
		return false, nil
	}
	for file := range m.files {
		if _, _, ok := directoryChild(path, file); ok {
			return true, nil
		}
	}
	return false, stdfs.ErrNotExist
}

func directoryChild(directory, file Path) (string, bool, bool) {
	relative, err := filepath.Rel(directory.String(), file.String())
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", false, false
	}
	name, _, nested := strings.Cut(relative, string(filepath.Separator))
	return name, nested, true
}

var _ FileSystem = (*Memory)(nil)
