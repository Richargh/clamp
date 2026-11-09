package environment

import (
	"io"

	"github.com/richargh/clamp/console/internal/kernel/clock"
	"github.com/richargh/clamp/console/internal/kernel/filesystem"
	"github.com/richargh/clamp/console/internal/kernel/process"
)

// Env supplies capabilities. Application configuration is passed separately.
type Env struct {
	Files     filesystem.FileSystem
	Processes process.Runner
	Clock     clock.Clock
	Stdin     io.Reader
	Stdout    io.Writer
	Stderr    io.Writer
}
