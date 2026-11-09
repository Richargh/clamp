package process

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOSProcess(t *testing.T) {
	t.Run("Should retain the process exit status without a custom error wrapper", func(t *testing.T) {
		// given
		testee := NewOS()
		executable, err := testee.LookPath("sh")
		require.NoError(t, err)

		// when
		err = testee.Run(t.Context(), Process{Executable: executable, Args: []string{"-c", "exit 42"}})

		// then
		var exit *exec.ExitError
		require.ErrorAs(t, err, &exit)
		require.Equal(t, 42, exit.ExitCode())
	})
}
