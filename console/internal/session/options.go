package session

import (
	"fmt"

	"github.com/richargh/clamp/console/internal/kernel/process"
)

// Harness is a supported coding agent and its container configuration.
type Harness struct {
	name      string
	display   string
	prefix    string
	configDir string
	configEnv string
	danger    string
}

func ParseHarness(raw string) (Harness, error) {
	switch raw {
	case "pi":
		return Harness{
			name:      "pi",
			display:   "Pi",
			prefix:    "pi",
			configDir: "/home/dev/.pi",
			configEnv: "CLAUDE_CONFIG_DIR",
		}, nil
	case "claude":
		return Harness{
			name:      "claude",
			display:   "Claude Code",
			prefix:    "claude-clamp",
			configDir: "/home/dev/.claude",
			configEnv: "CLAUDE_CONFIG_DIR",
			danger:    "--dangerously-skip-permissions",
		}, nil
	case "opencode":
		return Harness{
			name:      "opencode",
			display:   "OpenCode",
			prefix:    "opencode-clamp",
			configDir: "/home/dev/.local/share/opencode",
			configEnv: "OPENCODE_CONFIG_DIR",
		}, nil
	default:
		return Harness{}, fmt.Errorf("unsupported harness: %s", raw)
	}
}

func (h Harness) Name() string    { return h.name }
func (h Harness) Display() string { return h.display }

type Runtime struct {
	name       string
	executable process.Executable
}

func ParseRuntime(raw string) (Runtime, error) {
	if raw != "docker" && raw != "container" {
		return Runtime{}, fmt.Errorf("unsupported runtime: %s (expected docker or container)", raw)
	}
	return Runtime{name: raw}, nil
}

type Options struct {
	Harness        Harness
	Runtime        Runtime
	Folder         string
	Rebuild        bool
	NoCache        bool
	PerProjectAuth bool
	NoFirewall     bool
	Danger         bool
	Shell          bool
}
