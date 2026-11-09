package environment

import (
	"bytes"
	"strings"

	"github.com/richargh/clamp/console/internal/kernel/clock"
	"github.com/richargh/clamp/console/internal/kernel/filesystem"
	"github.com/richargh/clamp/console/internal/kernel/process"
)

// Memory supplies capabilities without host file access or process execution.
// Configure its adapters before use. It is not concurrent.
type Memory struct {
	Env
	ProcessMemory *process.Memory
	ClockMemory   *clock.Memory
	Output        bytes.Buffer
	Errors        bytes.Buffer
}

func NewMemory() *Memory {
	memory := &Memory{ProcessMemory: process.NewMemory(), ClockMemory: clock.NewMemory()}
	memory.Env = Env{
		Files: filesystem.NewMemory(), Processes: memory.ProcessMemory, Clock: memory.ClockMemory,
		Stdin: strings.NewReader(""), Stdout: &memory.Output, Stderr: &memory.Errors,
	}
	return memory
}
