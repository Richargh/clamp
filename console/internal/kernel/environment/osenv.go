package environment

import (
	"os"

	"github.com/richargh/clamp/console/internal/kernel/clock"
	"github.com/richargh/clamp/console/internal/kernel/filesystem"
	"github.com/richargh/clamp/console/internal/kernel/process"
)

func NewOS() Env {
	return Env{
		Files: filesystem.NewOS(), Processes: process.NewOS(), Clock: clock.NewOS(),
		Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr,
	}
}
