package filesystem

import "errors"

// FileSystem isolates host file access from session and CLI code.
type FileSystem interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, contents []byte, mode Mode) error
	MkdirAll(path string, mode Mode) error
	Stat(path string) (Info, error)
	Remove(path string) error
	Rename(source, target string) error
	CopyDirectory(source, target string) error
	Resolve(path string) (string, error)
	TempFile(directory, pattern string) (string, error)
}

// Mode contains file permission bits.
type Mode uint32

type Info struct {
	directory bool
	mode      Mode
}

func (i Info) IsDir() bool { return i.directory }

func (i Info) Mode() Mode { return i.mode }

var (
	ErrNotExist     = errors.New("path does not exist")
	ErrExist        = errors.New("path already exists")
	ErrNotDirectory = errors.New("path is not a directory")
	ErrIsDirectory  = errors.New("path is a directory")
	ErrNotEmpty     = errors.New("directory is not empty")
	ErrPermission   = errors.New("permission denied")
	ErrInvalid      = errors.New("invalid file operation")
)
