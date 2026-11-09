package filesystem

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

type OS struct{}

func NewOS() *OS { return &OS{} }

func (*OS) ReadFile(path string) ([]byte, error) {
	contents, err := os.ReadFile(path)
	return contents, fileError(err)
}

func (*OS) WriteFile(path string, contents []byte, mode Mode) error {
	return fileError(os.WriteFile(path, contents, fs.FileMode(mode)))
}

func (*OS) MkdirAll(path string, mode Mode) error {
	return fileError(os.MkdirAll(path, fs.FileMode(mode)))
}

func (*OS) Stat(path string) (Info, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Info{}, fileError(err)
	}
	return Info{directory: info.IsDir(), mode: Mode(info.Mode().Perm())}, nil
}

func (*OS) Remove(path string) error { return fileError(os.Remove(path)) }

func (*OS) Rename(source, target string) error {
	return fileError(os.Rename(source, target))
}

func (*OS) CopyDirectory(source, target string) error {
	return fileError(os.CopyFS(target, os.DirFS(source)))
}

func (*OS) Resolve(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fileError(err)
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	return resolved, fileError(err)
}

func (*OS) TempFile(directory, pattern string) (string, error) {
	file, err := os.CreateTemp(directory, pattern)
	if err != nil {
		return "", fileError(err)
	}
	return file.Name(), fileError(file.Close())
}

func fileError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, syscall.ENOTDIR):
		return fmt.Errorf("%w: %v", ErrNotDirectory, err)
	case errors.Is(err, syscall.EISDIR):
		return fmt.Errorf("%w: %v", ErrIsDirectory, err)
	case errors.Is(err, syscall.ENOTEMPTY):
		return fmt.Errorf("%w: %v", ErrNotEmpty, err)
	case os.IsNotExist(err):
		return fmt.Errorf("%w: %v", ErrNotExist, err)
	case os.IsExist(err):
		return fmt.Errorf("%w: %v", ErrExist, err)
	case os.IsPermission(err):
		return fmt.Errorf("%w: %v", ErrPermission, err)
	case errors.Is(err, syscall.EINVAL), errors.Is(err, os.ErrInvalid):
		return fmt.Errorf("%w: %v", ErrInvalid, err)
	default:
		return fmt.Errorf("file system: %v", err)
	}
}

var _ FileSystem = (*OS)(nil)
