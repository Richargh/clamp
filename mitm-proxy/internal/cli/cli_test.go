package cli

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/richargh/clamp/mitm-proxy/internal/kernel"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/environment"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/network"
	"github.com/spf13/cobra"
)

func TestNewCommandDoesNotReuseParsedFlags(t *testing.T) {
	env := listenFailureEnvironment(t)
	first := New(*env)
	if err := first.Run(t.Context(), []string{"serve", "--listen=127.0.0.1:0"}); err == nil || !strings.Contains(err.Error(), "listen:") {
		t.Fatalf("expected injected listen failure, got %v", err)
	}
	second := New(*env)
	if err := second.Run(t.Context(), []string{"serve"}); err == nil || !strings.Contains(err.Error(), "required flag") {
		t.Fatalf("second invocation reused flags: %v", err)
	}
}

func TestVersion(t *testing.T) {
	for _, version := range []string{"", "1.2.3-dev+abc"} {
		for _, flag := range []string{"-v", "--version"} {
			t.Run(version+flag, func(t *testing.T) {
				env := environment.NewMemory()
				env.Version = kernel.ParseClampVersion(version)
				var stdout, stderr bytes.Buffer
				env.Stdout, env.Stderr = &stdout, &stderr
				cmd := New(*env)
				if err := cmd.Run(t.Context(), []string{flag}); err != nil {
					t.Fatal(err)
				}
				want := "clamp-mitm-proxy version " + env.Version.String() + "\n"
				if stdout.String() != want || stderr.Len() != 0 {
					t.Fatalf("stdout = %q, stderr = %q, want %q", stdout.String(), stderr.String(), want)
				}
			})
		}
	}
}

func TestSubcommandOutputUsesInjectedWriters(t *testing.T) {
	env := environment.NewMemory()
	var stdout, stderr bytes.Buffer
	env.Stdout, env.Stderr = &stdout, &stderr
	cmd := New(*env)
	cmd.command.AddCommand(&cobra.Command{
		Use: "test-output",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), "normal output"); err != nil {
				return err
			}
			_, err := fmt.Fprintln(cmd.ErrOrStderr(), "diagnostic output")
			return err
		},
	})
	if err := cmd.Run(t.Context(), []string{"test-output"}); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "normal output\n" || stderr.String() != "diagnostic output\n" {
		t.Fatalf("stdout = %q, stderr = %q", stdout.String(), stderr.String())
	}
}

func TestHelp(t *testing.T) {
	for _, args := range [][]string{nil, {"--help"}, {"help", "serve"}, {"serve", "--help"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			output, err := execute(t, args...)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(output, "Usage:") {
				t.Fatalf("missing help: %s", output)
			}
		})
	}
}

func TestInvalidArguments(t *testing.T) {
	serveFlags := []string{"--listen=127.0.0.1:0"}
	type testCase struct {
		args []string
		want string
	}
	tests := []testCase{
		{[]string{"unknown"}, "unknown command"},
		{[]string{"init-ca"}, "unknown command"},
		{[]string{"serve", "--cert=cert"}, "unknown flag"},
		{[]string{"serve", "--key=key"}, "unknown flag"},
		{[]string{"serve", "--unknown"}, "unknown flag"},
		{[]string{"serve", "--ready-file=ready"}, "unknown flag"},
		{[]string{"serve", "--session-id=test"}, "unknown flag"},
		{[]string{"serve"}, "required flag"},
		{append(append([]string{"serve"}, serveFlags...), "extra"), "unknown command"},
	}
	for i, flag := range serveFlags {
		missing := append([]string{"serve"}, serveFlags[:i]...)
		missing = append(missing, serveFlags[i+1:]...)
		tests = append(tests, testCase{missing, "required flag"})
		for _, value := range []string{"", " \t\n"} {
			blank := append([]string{"serve"}, serveFlags...)
			blank[i+1] = strings.SplitN(flag, "=", 2)[0] + "=" + value
			tests = append(tests, testCase{blank, "must not be blank"})
		}
	}
	for _, test := range tests {
		t.Run(strings.Join(test.args, " "), func(t *testing.T) {
			output, err := execute(t, test.args...)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
			if output != "" {
				t.Fatalf("error should be returned without duplicate output: %q", output)
			}
		})
	}
}

func TestServeReturnsRuntimeErrors(t *testing.T) {
	for _, flag := range []string{"--ignore-proxy-env=false", "--ignore-proxy-env"} {
		t.Run(flag, func(t *testing.T) {
			_, err := execute(t, "serve", "--listen", "127.0.0.1:0", flag)
			if err == nil || !strings.Contains(err.Error(), "listen:") {
				t.Fatalf("expected listen error, got %v", err)
			}
		})
	}
}

func listenFailureEnvironment(t *testing.T) *environment.Env {
	t.Helper()
	address, err := network.ParseAddress("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	env := environment.NewMemory()
	env.Network = network.NewMemory(network.MemoryOptions{
		ListenErrors: map[network.Address]error{address: fmt.Errorf("test listen failure")},
	})
	return env
}

func execute(t *testing.T, args ...string) (string, error) {
	t.Helper()
	env := listenFailureEnvironment(t)
	var output bytes.Buffer
	env.Stdout = &output
	env.Stderr = &output
	cmd := New(*env)
	err := cmd.Run(context.Background(), args)
	return output.String(), err
}
