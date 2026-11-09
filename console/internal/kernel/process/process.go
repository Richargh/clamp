package process

import (
	"context"
	"fmt"
	"io"
	"strings"
)

// Executable identifies a program. LookPath resolves names through the runner.
type Executable struct{ value string }

func ParseExecutable(raw string) (Executable, error) {
	if raw == "" || strings.ContainsRune(raw, '\x00') {
		return Executable{}, fmt.Errorf("invalid executable: %q", raw)
	}
	return Executable{value: raw}, nil
}

func (e Executable) String() string { return e.value }

type Process struct {
	Executable Executable
	Args       []string
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
}

// Runner owns executable lookup and process execution.
type Runner interface {
	LookPath(name string) (Executable, error)
	Run(context.Context, Process) error
}
