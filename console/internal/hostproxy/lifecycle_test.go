package hostproxy

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/richargh/clamp/console/internal/kernel/environment"
	"github.com/richargh/clamp/console/internal/kernel/process"
	"github.com/stretchr/testify/require"
)

func TestHostProxyLifecycle(t *testing.T) {
	t.Run("Should cancel the process and wait for completion when closed", func(t *testing.T) {
		// given
		env, options := proxyFixture(t, "memory", `{"schema":1,"address":"0.0.0.0:23456"}`)
		processes := env.Processes.(*process.Memory)
		onRun := processes.OnRun
		var proxyContext context.Context
		exited := make(chan struct{})
		processes.OnRun = func(ctx context.Context, p process.Process) error {
			proxyContext = ctx
			defer close(exited)
			return onRun(ctx, p)
		}
		testee := New(env)
		proxy, err := testee.Start(t.Context(), options)
		require.NoError(t, err)
		// when
		proxy.Close()
		// then
		require.ErrorIs(t, proxyContext.Err(), context.Canceled)
		select {
		case <-exited:
		default:
			t.Fatal("Close returned before the process exited")
		}
	})
	t.Run("Should append stdout and stderr to the session log", func(t *testing.T) {
		// given
		env, options := proxyFixture(t, "memory", "")
		processes := env.Processes.(*process.Memory)
		processes.OnRun = func(ctx context.Context, p process.Process) error {
			if _, err := io.WriteString(p.Stdout, "{\"schema\":1,\"address\":\"0.0.0.0:23456\"}\nrequest log\n"); err != nil {
				return err
			}
			if _, err := io.WriteString(p.Stderr, "diagnostic\n"); err != nil {
				return err
			}
			<-ctx.Done()
			return ctx.Err()
		}
		logPath := filepath.Join(options.LogDirectory.String(), options.LogName)
		require.NoError(t, env.Files.WriteFile(logPath, []byte("existing\n"), 0600))
		testee := New(env)
		// when
		proxy, err := testee.Start(t.Context(), options)
		require.NoError(t, err)
		proxy.Close()
		// then
		log, err := env.Files.ReadFile(logPath)
		require.NoError(t, err)
		require.Contains(t, string(log), "existing\n")
		require.Contains(t, string(log), "request log\n")
		require.Contains(t, string(log), "diagnostic\n")
		require.NotContains(t, string(log), "schema")
	})
	t.Run("Should close the instance when its process exits", func(t *testing.T) {
		// given
		env, options := proxyFixture(t, "memory", "")
		processes := env.Processes.(*process.Memory)
		finish := make(chan struct{})
		processes.OnRun = func(ctx context.Context, p process.Process) error {
			if _, err := io.WriteString(p.Stdout, "{\"schema\":1,\"address\":\"0.0.0.0:23456\"}\n"); err != nil {
				return err
			}
			select {
			case <-finish:
				return errors.New("proxy stopped")
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		testee := New(env)
		proxy, err := testee.Start(t.Context(), options)
		require.NoError(t, err)
		defer proxy.Close()
		// when
		close(finish)
		// then
		select {
		case <-proxy.Done():
		case <-time.After(2 * time.Second):
			t.Fatal("proxy exit was not reported")
		}
	})
	t.Run("Should report a process startup failure", func(t *testing.T) {
		// given
		env, options := proxyFixture(t, "memory", "")
		processes := env.Processes.(*process.Memory)
		failure := errors.New("cannot start proxy")
		processes.OnRun = func(context.Context, process.Process) error { return failure }
		testee := New(env)
		// when
		_, err := testee.Start(t.Context(), options)
		// then
		require.ErrorIs(t, err, failure)
		require.Len(t, processes.Calls, 1)
	})
	t.Run("Should stop waiting for readiness when the caller cancels", func(t *testing.T) {
		// given
		env, options := proxyFixture(t, "memory", "")
		processes := env.Processes.(*process.Memory)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		processes.OnRun = func(context.Context, process.Process) error { cancel(); return ctx.Err() }
		testee := New(env)
		// when
		_, err := testee.Start(ctx, options)
		// then
		require.ErrorIs(t, err, context.Canceled)
	})
	t.Run("Should reject an unusable log path before starting a process", func(t *testing.T) {
		// given
		memory := environment.NewMemory()
		_, options := proxyFixture(t, "memory", "")
		testee := New(memory.Env)
		// when
		_, err := testee.Start(t.Context(), options)
		// then
		require.Error(t, err)
		require.Empty(t, memory.ProcessMemory.Calls)
	})
}
