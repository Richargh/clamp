package process

import (
	"context"
	"fmt"
	"slices"
)

// Memory records processes without starting programs. Configure it before use.
// It is not concurrent.
type Memory struct {
	Executables map[string]Executable
	Calls       []Process
	OnRun       func(context.Context, Process) error
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
	m.Calls = append(m.Calls, process)
	if m.OnRun != nil {
		return m.OnRun(ctx, process)
	}
	return nil
}

var _ Runner = (*Memory)(nil)
