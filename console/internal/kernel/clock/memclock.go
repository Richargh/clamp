package clock

import "time"

// Memory returns a configured time without reading the host clock.
type Memory struct{ Time time.Time }

func NewMemory() *Memory { return &Memory{Time: time.Unix(0, 0).UTC()} }

func (m *Memory) Now() time.Time { return m.Time }

var _ Clock = (*Memory)(nil)
