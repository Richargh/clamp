package clock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMemoryClock(t *testing.T) {
	t.Run("Should start at a fixed UTC time", func(t *testing.T) {
		// given
		testee := NewMemory()

		// when
		actual := testee.Now()

		// then
		require.Equal(t, time.Unix(0, 0).UTC(), actual)
	})
	t.Run("Should return the configured time", func(t *testing.T) {
		// given
		testee := NewMemory()
		now := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
		testee.Time = now

		// when
		actual := testee.Now()

		// then
		require.Equal(t, now, actual)
	})
}
