package session

import (
	"bufio"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/richargh/clamp/console/internal/kernel/filesystem"
)

func (l *Runner) authPath() string { return l.config.AuthStore().String() }

func (l *Runner) authPolicy(workspace string, explicit bool) (AuthPolicy, error) {
	if strings.ContainsAny(workspace, "\t\r\n") {
		return AuthPolicy{}, fmt.Errorf("workspace path cannot contain tabs or newlines")
	}
	data, err := l.env.Files.ReadFile(l.authPath())
	if err != nil && !errors.Is(err, filesystem.ErrNotExist) {
		return AuthPolicy{}, err
	}
	raw, kept := readAuthPolicyRecord(string(data), workspace)
	policy, err := l.resolveAuthPolicy(workspace, raw, explicit)
	if err != nil {
		return AuthPolicy{}, err
	}
	if explicit || raw == "" {
		kept = append(kept, workspace+"\t"+policy.String())
		if err := l.saveAuthPolicies(kept); err != nil {
			return AuthPolicy{}, err
		}
	}
	return policy, nil
}

func (l *Runner) resolveAuthPolicy(workspace, raw string, explicit bool) (AuthPolicy, error) {
	if explicit {
		return ProjectAuthPolicy(), nil
	}
	if raw == "" {
		return l.promptAuthPolicy(workspace)
	}
	policy, err := ParseAuthPolicy(raw)
	if err != nil {
		return AuthPolicy{}, fmt.Errorf("invalid Clamp auth policy for %s: %w", workspace, err)
	}
	return policy, nil
}

// readAuthPolicyRecord extracts raw storage data before parsing the selected policy.
func readAuthPolicyRecord(data, workspace string) (string, []string) {
	policy := ""
	var kept []string
	for _, line := range strings.Split(data, "\n") {
		path, value, ok := strings.Cut(line, "\t")
		if ok && path == workspace {
			policy = value
		} else if line != "" {
			kept = append(kept, line)
		}
	}
	return policy, kept
}

func (l *Runner) promptAuthPolicy(workspace string) (AuthPolicy, error) {
	_, _ = fmt.Fprintf(l.env.Stderr, "No Clamp auth preference found for:\n  %s\n\nUse:\n  1) global/shared auth\n  2) per-project auth\n\n", workspace)
	reader := bufio.NewReader(l.env.Stdin)
	for {
		_, _ = fmt.Fprint(l.env.Stderr, "Choice [1/2]: ")
		choice, err := reader.ReadString('\n')
		if err != nil {
			return AuthPolicy{}, fmt.Errorf("unable to read auth preference: %w", err)
		}
		switch strings.TrimSpace(choice) {
		case "", "1":
			return GlobalAuthPolicy(), nil
		case "2":
			return ProjectAuthPolicy(), nil
		default:
			_, _ = fmt.Fprintln(l.env.Stderr, "Please enter 1 or 2.")
		}
	}
}

func (l *Runner) saveAuthPolicies(lines []string) error {
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
