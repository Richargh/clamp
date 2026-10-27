package acceptance_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func RequireSingle[T any](t *testing.T, items []T) T {
	t.Helper()
	require.Len(t, items, 1)
	return items[0]
}
