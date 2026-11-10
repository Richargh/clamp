package process

import (
	"context"
	"fmt"
	"slices"
	"sync"
)

// Memory records processes without starting programs. Configure it before use
// and inspect Calls after execution ends. Run supports concurrent calls; OnRun
// must also support them when it is configured.
type Memory struct {
	Executables map[string]Executable
	Calls       []Process
	OnRun       func(context.Context, Process) error
	mu          sync.Mutex
}

func NewMemory() *Memory {
	return &Memory{Executables: make(map[string]Executable)}
}

func (m *Memory) LookPath(name string) (Executable, error) {
	if executable, ok := m.Executables[name]; ok {
		if _, err := ParseExecutable(executable.String()); err != nil {
			return Executable{}, err
		}
		return executable, nil
	}
	return Executable{}, fmt.Errorf("executable not found in memory: %s", name)
}

func (m *Memory) Run(ctx context.Context, process Process) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := ParseExecutable(process.Executable.String()); err != nil {
		return err
	}
	process.Args = slices.Clone(process.Args)
	m.mu.Lock()
	m.Calls = append(m.Calls, process)
	m.mu.Unlock()
	if m.OnRun != nil {
		return m.OnRun(ctx, process)
	}
	return nil
}

var _ Runner = (*Memory)(nil)
