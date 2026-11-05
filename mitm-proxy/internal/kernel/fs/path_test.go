package fs

import (
	stdfs "io/fs"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPath(t *testing.T) {
	t.Run("Should normalize a path", func(t *testing.T) {
		// given
		const raw = "rules/../rules/rules.txt"

		// when
		testee, err := ParsePath(raw)

		// then
		require.NoError(t, err)
		require.Equal(t, filepath.Join("rules", "rules.txt"), testee.String())
	})
	for _, raw := range []string{"", " \t\n", "rules\x00/file"} {
		t.Run("Should reject invalid path "+raw, func(t *testing.T) {
			// given
			value := raw

			// when
			testee, err := ParsePath(value)

			// then
			require.ErrorIs(t, err, stdfs.ErrInvalid)
			require.False(t, testee.IsValid())
		})
	}
	t.Run("Should join a directory and a child path", func(t *testing.T) {
		// given
		testee := mustPath(t, "rules")

		// when
		child, err := testee.Join("rules.txt")

		// then
		require.NoError(t, err)
		require.Equal(t, filepath.Join("rules", "rules.txt"), child.String())
	})
	t.Run("Should reject joining an invalid parent path", func(t *testing.T) {
		// given
		var testee Path

		// when
		_, err := testee.Join("rules.txt")

		// then
		require.ErrorIs(t, err, stdfs.ErrInvalid)
	})
}
