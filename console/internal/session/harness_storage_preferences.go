package session

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/richargh/clamp/console/internal/kernel/filesystem"
)

func (l *Runner) authPath() string { return l.config.AuthStore().String() }

func (l *Runner) loadHarnessStorage(workspace string, explicit bool) (HarnessVolumeChoice, error) {
	raw, _, err := l.harnessStorageRecord(workspace)
	if err != nil {
		return NoChoiceMade, err
	}
	if explicit {
		return ProjectVolume, nil
	}
	choice, err := ParseHarnessVolumeChoice(raw)
	if err != nil {
		return NoChoiceMade, fmt.Errorf("invalid Clamp harness storage for %s: %w", workspace, err)
	}
	return choice, nil
}

func (l *Runner) rememberHarnessStorage(workspace string, choice HarnessVolumeChoice) error {
	var value string
	switch choice {
	case SharedVolume:
		value = "global"
	case ProjectVolume:
		value = "project"
	default:
		return fmt.Errorf("cannot save harness storage choice %d", choice)
	}
	raw, kept, err := l.harnessStorageRecord(workspace)
	if err != nil {
		return err
	}
	if raw == value {
		return nil
	}
	kept = append(kept, workspace+"\t"+value)
	return l.saveHarnessStorage(kept)
}

func (l *Runner) harnessStorageRecord(workspace string) (string, []string, error) {
	if strings.ContainsAny(workspace, "\t\r\n") {
		return "", nil, fmt.Errorf("workspace path cannot contain tabs or newlines")
	}
	data, err := l.env.Files.ReadFile(l.authPath())
	if err != nil && !errors.Is(err, filesystem.ErrNotExist) {
		return "", nil, err
	}
	raw, kept := readHarnessStorageRecord(string(data), workspace)
	return raw, kept, nil
}

// readHarnessStorageRecord extracts raw data before parsing the storage choice.
func readHarnessStorageRecord(data, workspace string) (string, []string) {
	storage := ""
	var kept []string
	for _, line := range strings.Split(data, "\n") {
		path, value, ok := strings.Cut(line, "\t")
		if ok && path == workspace {
			storage = value
		} else if line != "" {
			kept = append(kept, line)
		}
	}
	return storage, kept
}

func (l *Runner) saveHarnessStorage(lines []string) error {
	directory := filepath.Dir(l.authPath())
	if err := l.env.Files.MkdirAll(directory, 0700); err != nil {
		return err
	}
	// Replace the file atomically, so an interrupted write cannot lose preferences.
	temp, err := l.env.Files.TempFile(directory, "auth-projects.*")
	if err != nil {
		return err
	}
	defer func() { _ = l.env.Files.Remove(temp) }()
	if err := l.env.Files.WriteFile(temp, []byte(strings.Join(lines, "\n")+"\n"), 0600); err != nil {
		return err
	}
	return l.env.Files.Rename(temp, l.authPath())
}
