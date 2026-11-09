package clock

import "time"

type OS struct{}

func NewOS() *OS { return &OS{} }

func (*OS) Now() time.Time { return time.Now() }

var _ Clock = (*OS)(nil)
