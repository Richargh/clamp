package clock

import "time"

// Clock supplies the current time independently of host access.
type Clock interface {
	Now() time.Time
}
