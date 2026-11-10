// Package hostproxy owns the host proxy process for a container session.
package hostproxy

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"path/filepath"
	"strings"
	"time"

	"github.com/richargh/clamp/console/internal/kernel/environment"
	"github.com/richargh/clamp/console/internal/kernel/filesystem"
	"github.com/richargh/clamp/console/internal/kernel/process"
)

// Options supplies the session paths and domain policy. The proxy executable is
// fixed by this package, not selected by callers.
type Options struct {
	AssetsDir    filesystem.Path
	Workspace    filesystem.Path
	LogDirectory filesystem.Path
	LogName      string
	NoFirewall   bool
}

type Instance interface {
	Port() uint16
	Done() <-chan struct{}
	Close()
}

// HostProxy uses the same service code with OS or memory capabilities.
type HostProxy struct{ env environment.Env }

func New(env environment.Env) *HostProxy { return &HostProxy{env: env} }

type instance struct {
	port   uint16
	done   chan struct{}
	cancel context.CancelFunc
}

func (p *instance) Port() uint16          { return p.port }
func (p *instance) Done() <-chan struct{} { return p.done }
func (p *instance) Close()                { p.cancel(); <-p.done }

func (h *HostProxy) Start(ctx context.Context, options Options) (Instance, error) {
	executable, err := process.ParseExecutable(filepath.Join(options.AssetsDir.String(), "bin", "clamp-mitm-proxy"))
	if err != nil {
		return nil, err
	}
	log, err := h.openLog(options)
	if err != nil {
		return nil, err
	}
	childCtx, cancel := context.WithCancel(ctx)
	p := &instance{done: make(chan struct{}), cancel: cancel}
	output, writer := io.Pipe()
	ready := make(chan error, 1)
	go p.readOutput(output, log, ready)
	go func() {
		err := h.env.Processes.Run(childCtx, process.Process{
			Executable: executable, Args: proxyArgs(options), Stdout: writer, Stderr: log,
		})
		_ = writer.CloseWithError(err)
	}()
	return awaitReady(ctx, p, ready)
}

func (h *HostProxy) openLog(options Options) (io.WriteCloser, error) {
	if options.LogName == "" || options.LogName == "." || options.LogName == ".." ||
		filepath.Base(options.LogName) != options.LogName || strings.ContainsRune(options.LogName, '\x00') {
		return nil, fmt.Errorf("invalid proxy log filename: %q", options.LogName)
	}
	return h.env.Files.OpenAppend(filepath.Join(options.LogDirectory.String(), options.LogName), 0600)
}

func proxyArgs(options Options) []string {
	args := []string{"serve", "--listen", "0.0.0.0:0"}
	if options.NoFirewall {
		return append(args, "--no-firewall")
	}
	return append(args, "--allowed-domains-dir", filepath.Join(options.Workspace.String(), ".clamp", "allowed-domains.d"))
}

func (p *instance) readOutput(output *io.PipeReader, log io.WriteCloser, ready chan<- error) {
	defer close(p.done)
	defer func() { _ = output.Close(); _ = log.Close() }()
	reader := bufio.NewReader(output)
	port, err := readReady(reader)
	p.port = port
	ready <- err
	// Drain output after a log write fails, so process shutdown cannot block.
	if _, err := io.Copy(log, reader); err != nil {
		_, _ = io.Copy(io.Discard, reader)
	}
}

func readReady(reader *bufio.Reader) (uint16, error) {
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return 0, err
	}
	var record struct {
		Schema  int    `json:"schema"`
		Address string `json:"address"`
	}
	if err := json.Unmarshal(line, &record); err != nil {
		return 0, err
	}
	if record.Schema != 1 {
		return 0, errors.New("unsupported proxy readiness schema")
	}
	return parseAddress(record.Address)
}

func parseAddress(raw string) (uint16, error) {
	address, err := netip.ParseAddrPort(raw)
	if err != nil {
		return 0, err
	}
	if (!address.Addr().Is4() && !address.Addr().IsUnspecified()) || address.Port() == 0 {
		return 0, errors.New("invalid proxy listen address")
	}
	return address.Port(), nil
}

func awaitReady(ctx context.Context, p *instance, ready <-chan error) (Instance, error) {
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	select {
	case err := <-ready:
		if err == nil {
			return p, nil
		}
		p.Close()
		return nil, fmt.Errorf("host proxy readiness: %w", err)
	case <-ctx.Done():
		p.Close()
		return nil, ctx.Err()
	case <-timer.C:
		p.Close()
		return nil, errors.New("host proxy readiness timed out")
	}
}
