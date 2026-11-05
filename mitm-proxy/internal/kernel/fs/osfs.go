package fs

import (
	stdfs "io/fs"
	"os"
)

type OS struct{}

func NewOS() *OS { return &OS{} }

func (o *OS) ReadFile(path Path) ([]byte, error) {
	isDirectory, err := o.IsDirectory(path)
	if err != nil {
		return nil, err
	}
	if isDirectory {
		return nil, ErrIsDirectory
	}
	return os.ReadFile(path.String())
}

func (o *OS) ListDirectory(path Path) ([]Entry, error) {
	isDirectory, err := o.IsDirectory(path)
	if err != nil {
		return nil, err
	}
	if !isDirectory {
		return nil, ErrNotDirectory
	}
	entries, err := os.ReadDir(path.String())
	if err != nil {
		return nil, err
	}
	result := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, Entry{Name: entry.Name(), IsDirectory: entry.IsDir()})
	}
	return result, nil
}

func (*OS) IsDirectory(path Path) (bool, error) {
	if !path.IsValid() {
		return false, stdfs.ErrInvalid
	}
	info, err := os.Stat(path.String())
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

var _ FileSystem = (*OS)(nil)
