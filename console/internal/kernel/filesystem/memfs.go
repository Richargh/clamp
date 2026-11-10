package filesystem

import (
	"bytes"
	"errors"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type memoryEntry struct {
	info     Info
	contents []byte
}

// Memory stores files and directories without host access. Relative paths use
// the virtual root as their working directory. Symbolic links are not modeled.
type Memory struct {
	entries  map[string]memoryEntry
	nextTemp uint64
}

func NewMemory() *Memory {
	root := string(filepath.Separator)
	return &Memory{entries: map[string]memoryEntry{
		root:                       {info: Info{directory: true, mode: 0755}},
		filepath.Join(root, "tmp"): {info: Info{directory: true, mode: 0700}},
	}}
}

func (m *Memory) ReadFile(path string) ([]byte, error) {
	entry, err := m.lookup(path)
	if err != nil {
		return nil, err
	}
	if entry.info.IsDir() {
		return nil, ErrIsDirectory
	}
	return bytes.Clone(entry.contents), nil
}

func (m *Memory) WriteFile(path string, contents []byte, mode Mode) error {
	if path == "" {
		return ErrNotExist
	}
	entry, err := m.lookup(path)
	if err != nil && !errors.Is(err, ErrNotExist) {
		return err
	}
	if err == nil && entry.info.IsDir() {
		return ErrIsDirectory
	}
	if err == nil {
		mode = entry.info.Mode()
	}
	path = memoryPath(path)
	if err := m.requireDirectory(filepath.Dir(path)); err != nil {
		return err
	}
	m.entries[path] = memoryEntry{info: Info{mode: mode & 0777}, contents: bytes.Clone(contents)}
	return nil
}

func (m *Memory) OpenAppend(path string, mode Mode) (io.WriteCloser, error) {
	if _, err := m.lookup(path); errors.Is(err, ErrNotExist) {
		if err := m.WriteFile(path, nil, mode); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	entry, err := m.lookup(path)
	if err != nil {
		return nil, err
	}
	if entry.info.IsDir() {
		return nil, ErrIsDirectory
	}
	return &memoryAppend{files: m, path: memoryPath(path)}, nil
}

type memoryAppend struct {
	files  *Memory
	path   string
	closed bool
	mu     sync.Mutex
}

func (w *memoryAppend) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return 0, ErrInvalid
	}
	entry := w.files.entries[w.path]
	entry.contents = append(entry.contents, data...)
	w.files.entries[w.path] = entry
	return len(data), nil
}

func (w *memoryAppend) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closed = true
	return nil
}

func (m *Memory) MkdirAll(path string, mode Mode) error {
	entry, err := m.lookup(path)
	if err == nil {
		if !entry.info.IsDir() {
			return ErrNotDirectory
		}
		return nil
	}
	if !errors.Is(err, ErrNotExist) || path == "" {
		return err
	}
	path = memoryPath(path)
	if err := m.MkdirAll(filepath.Dir(path), mode); err != nil {
		return err
	}
	m.entries[path] = memoryEntry{info: Info{directory: true, mode: mode & 0777}}
	return nil
}

func (m *Memory) Stat(path string) (Info, error) {
	entry, err := m.lookup(path)
	return entry.info, err
}

func (m *Memory) Remove(path string) error {
	entry, err := m.lookup(path)
	if err != nil {
		return err
	}
	path = memoryPath(path)
	if entry.info.IsDir() && len(m.descendants(path)) > 0 {
		return ErrNotEmpty
	}
	delete(m.entries, path)
	return nil
}

func (m *Memory) Rename(source, target string) error {
	if target == "" {
		return ErrNotExist
	}
	entry, err := m.lookup(source)
	if err != nil {
		return err
	}
	source, target = memoryPath(source), memoryPath(target)
	if source == target {
		return nil
	}
	if isDescendant(source, target) {
		return ErrInvalid
	}
	if err := m.requireDirectory(filepath.Dir(target)); err != nil {
		return err
	}
	if err := m.checkReplacement(target, entry.info); err != nil {
		return err
	}
	paths := append(m.descendants(source), source)
	moved := make(map[string]memoryEntry, len(paths))
	for _, path := range paths {
		moved[target+strings.TrimPrefix(path, source)] = m.entries[path]
		delete(m.entries, path)
	}
	for path, entry := range moved {
		m.entries[path] = entry
	}
	return nil
}

func (m *Memory) CopyDirectory(source, target string) error {
	if target == "" {
		return ErrNotExist
	}
	if err := m.requireDirectory(source); err != nil {
		return err
	}
	source, target = memoryPath(source), memoryPath(target)
	if source == target || isDescendant(source, target) {
		return ErrInvalid
	}
	paths := m.descendants(source)
	if err := m.MkdirAll(target, 0777); err != nil {
		return err
	}
	for _, path := range paths {
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if err := m.copyEntry(path, filepath.Join(target, relative)); err != nil {
			return err
		}
	}
	return nil
}

func (m *Memory) Resolve(path string) (string, error) {
	if path == "" {
		path = string(filepath.Separator)
	}
	if _, err := m.lookup(path); err != nil {
		return "", err
	}
	return memoryPath(path), nil
}

func (m *Memory) TempFile(directory, pattern string) (string, error) {
	if directory == "" {
		directory = filepath.Join(string(filepath.Separator), "tmp")
	}
	if strings.ContainsRune(pattern, filepath.Separator) || strings.ContainsRune(pattern, '\x00') {
		return "", ErrInvalid
	}
	if err := m.requireDirectory(directory); err != nil {
		return "", err
	}
	prefix, suffix := pattern, ""
	if index := strings.LastIndex(pattern, "*"); index >= 0 {
		prefix, suffix = pattern[:index], pattern[index+1:]
	}
	for {
		m.nextTemp++
		name := prefix + strconv.FormatUint(m.nextTemp, 10) + suffix
		path := filepath.Join(memoryPath(directory), name)
		if _, exists := m.entries[path]; exists {
			continue
		}
		return path, m.WriteFile(path, nil, 0600)
	}
}

func (m *Memory) lookup(path string) (memoryEntry, error) {
	if strings.ContainsRune(path, '\x00') {
		return memoryEntry{}, ErrInvalid
	}
	if path == "" {
		return memoryEntry{}, ErrNotExist
	}
	path = memoryPath(path)
	if entry, ok := m.entries[path]; ok {
		return entry, nil
	}
	for parent := filepath.Dir(path); parent != path; parent = filepath.Dir(parent) {
		if entry, ok := m.entries[parent]; ok && !entry.info.IsDir() {
			return memoryEntry{}, ErrNotDirectory
		}
		path = parent
	}
	return memoryEntry{}, ErrNotExist
}

func (m *Memory) requireDirectory(path string) error {
	entry, err := m.lookup(path)
	if err != nil {
		return err
	}
	if !entry.info.IsDir() {
		return ErrNotDirectory
	}
	return nil
}

func (m *Memory) checkReplacement(target string, source Info) error {
	entry, err := m.lookup(target)
	if errors.Is(err, ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if entry.info.IsDir() && !source.IsDir() {
		return ErrIsDirectory
	}
	if !entry.info.IsDir() && source.IsDir() {
		return ErrNotDirectory
	}
	if entry.info.IsDir() && len(m.descendants(target)) > 0 {
		return ErrNotEmpty
	}
	return nil
}

func (m *Memory) copyEntry(source, target string) error {
	entry := m.entries[source]
	if entry.info.IsDir() {
		return m.MkdirAll(target, 0777)
	}
	if _, exists := m.entries[target]; exists {
		return ErrExist
	}
	return m.WriteFile(target, entry.contents, 0666|entry.info.Mode()&0111)
}

func (m *Memory) descendants(directory string) []string {
	var paths []string
	for path := range m.entries {
		if isDescendant(directory, path) {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	return paths
}

func memoryPath(path string) string {
	if !filepath.IsAbs(path) {
		path = filepath.Join(string(filepath.Separator), path)
	}
	return filepath.Clean(path)
}

func isDescendant(directory, path string) bool {
	relative, err := filepath.Rel(directory, path)
	return err == nil && relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

var _ FileSystem = (*Memory)(nil)
