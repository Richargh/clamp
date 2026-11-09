package process

import (
	"context"
	"os/exec"
)

type OS struct{}

func NewOS() *OS { return &OS{} }

func (*OS) LookPath(name string) (Executable, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return Executable{}, err
	}
	return ParseExecutable(path)
}

func (*OS) Run(ctx context.Context, process Process) error {
	if _, err := ParseExecutable(process.Executable.String()); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, process.Executable.String(), process.Args...)
	cmd.Stdin = process.Stdin
	cmd.Stdout = process.Stdout
	cmd.Stderr = process.Stderr
	return cmd.Run()
}

var _ Runner = (*OS)(nil)
