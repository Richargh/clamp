package session

import (
	"context"
	"fmt"
)

// Session is prepared launch state. Volume selection is separate from user
// interaction, so a CLI can ask for a missing choice before launch.
type Session struct {
	runner    *Runner
	options   Options
	project   projectIdentity
	container containerSession
}

func (s *Session) Workspace() Workspace { return s.container.workspace }

// HasNoHarnessVolume reports whether a volume choice is still required.
// It does not inspect whether a named volume exists in the runtime.
func (s *Session) HasNoHarnessVolume() bool {
	return s.container.harnessStorage.Choice() == NoChoiceMade
}

func (s *Session) SetHarnessVolume(choice HarnessVolumeChoice) error {
	storage, err := resolveHarnessStorage(choice, s.project)
	if err != nil {
		return err
	}
	s.container.harnessStorage = storage
	return nil
}

func (s *Session) Launch(ctx context.Context) error {
	if s.HasNoHarnessVolume() {
		return fmt.Errorf("harness storage selection is required for %s", s.Workspace().String())
	}
	l := s.runner
	if err := l.rememberHarnessStorage(s.Workspace().String(), s.container.harnessStorage.Choice()); err != nil {
		return err
	}
	l.printConfig(s.options, s.container.harnessStorage.Choice())
	if err := l.setupProject(s.Workspace()); err != nil {
		return err
	}
	if err := l.build(ctx, s.options, s.project); err != nil {
		return err
	}
	return l.run(ctx, s.options, s.container)
}
