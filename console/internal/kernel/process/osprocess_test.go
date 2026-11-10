package process

import (
	"bufio"
	"context"
	"io"
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
	t.Run("Should stop a running process when its context is canceled", func(t *testing.T) {
		// given
		testee := NewOS()
		executable, err := testee.LookPath("sh")
		require.NoError(t, err)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		output, writer := io.Pipe()
		defer func() { _ = output.Close() }()
		finished := make(chan error, 1)
		go func() {
			err := testee.Run(ctx, Process{Executable: executable, Args: []string{"-c", "printf 'ready\\n'; exec sleep 30"}, Stdout: writer})
			_ = writer.CloseWithError(err)
			finished <- err
		}()
		_, err = bufio.NewReader(output).ReadString('\n')
		require.NoError(t, err)
		// when
		cancel()
		err = <-finished
		// then
		require.Error(t, err)
	})
}
