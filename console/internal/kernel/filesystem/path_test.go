package filesystem

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPathParsing(t *testing.T) {
	t.Run("Should clean an absolute path without resolving it", func(t *testing.T) {
		// given
		raw := "/not-on-host/../assets/file"

		// when
		testee, err := ParsePath(raw)

		// then
		require.NoError(t, err)
		require.Equal(t, "/assets/file", testee.String())
	})
	for _, raw := range []string{"", "relative/file", "/file\x00suffix"} {
		t.Run(fmt.Sprintf("Should reject invalid path %q", raw), func(t *testing.T) {
			// given
			// when
			testee, err := ParsePath(raw)

			// then
			require.Error(t, err)
			require.Empty(t, testee.String())
		})
	}
}
