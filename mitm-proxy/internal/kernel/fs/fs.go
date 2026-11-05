package fs

import "errors"

// FileSystem exposes read operations with domain paths and entries.
// Missing paths and invalid paths use io/fs.ErrNotExist and io/fs.ErrInvalid.
// OS errors retain their standard error identity, including io/fs.ErrPermission.
type FileSystem interface {
	ReadFile(path Path) ([]byte, error)
	ListDirectory(path Path) ([]Entry, error)
	IsDirectory(path Path) (bool, error)
}

type Entry struct {
	Name        string
	IsDirectory bool
}

var (
	ErrNotDirectory = errors.New("path is not a directory")
	ErrIsDirectory  = errors.New("path is a directory")
)
