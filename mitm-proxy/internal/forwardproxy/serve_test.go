package forwardproxy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/richargh/clamp/mitm-proxy/internal/kernel/environment"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/network"
)

func TestServePrintsReadinessAndClosesListener(t *testing.T) {
	// given
	env, listener := memoryProxy(t)
	env.PID = 42
	written := make(chan []byte, 2)
	env.Stdout = readinessWriter{written: written}
	var stderr bytes.Buffer
	env.Stderr = &stderr
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	testee := NewHandler(env, Options{Proxy: directProxy})
	done := make(chan error, 1)

	// when
	go func() { done <- testee.Serve(ctx, "127.0.0.1:0") }()
	var data []byte
	select {
	case data = <-written:
	case err := <-done:
		t.Fatalf("exited before readiness: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("readiness timeout")
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("shutdown timeout")
	}

	// then
	if len(data) == 0 || data[len(data)-1] != '\n' || bytes.Count(data, []byte{'\n'}) != 1 {
		t.Fatalf("readiness must be one JSON line: %q", data)
	}
	if bytes.Contains(data, []byte(`"session_id"`)) {
		t.Fatalf("readiness must not contain a session ID: %q", data)
	}
	var record readyRecord
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	if record.PID != 42 || record.Address != listener.Addr().String() || record.Schema != 1 {
		t.Fatalf("readiness = %+v", record)
	}
	if len(written) != 0 || stderr.Len() != 0 {
		t.Fatalf("unexpected additional output: stdout lines=%d, stderr=%q", len(written), stderr.String())
	}
	select {
	case <-listener.closed:
	default:
		t.Fatal("listener not closed")
	}
}

func TestServeReturnsUnexpectedListenerClosure(t *testing.T) {
	// given
	env, listener := memoryProxy(t)
	_ = listener.Close()
	testee := NewHandler(env, Options{Proxy: directProxy})

	// when
	err := testee.Serve(t.Context(), "127.0.0.1:0")

	// then
	if !errors.Is(err, net.ErrClosed) {
		t.Fatalf("serve = %v, want net.ErrClosed", err)
	}
}

func TestServeClosesListenerWhenStdoutFails(t *testing.T) {
	// given
	env, listener := memoryProxy(t)
	failure := errors.New("stdout failed")
	env.Stdout = readinessWriter{err: failure}
	testee := NewHandler(env, Options{Proxy: directProxy})

	// when
	err := testee.Serve(t.Context(), "127.0.0.1:0")

	// then
	if !errors.Is(err, failure) {
		t.Fatalf("serve = %v, want stdout failure", err)
	}
	select {
	case <-listener.closed:
	default:
		t.Fatal("listener leaked after readiness failure")
	}
}

func TestServeDoesNotPrintReadinessWhenListenFails(t *testing.T) {
	// given
	env := environment.NewMemory()
	var stdout bytes.Buffer
	env.Stdout = &stdout
	address, err := network.ParseAddress("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	failure := errors.New("listen failed")
	env.Network = network.NewMemory(network.MemoryOptions{
		ListenErrors: map[network.Address]error{address: failure},
	})
	testee := NewHandler(env, Options{Proxy: directProxy})

	// when
	err = testee.Serve(t.Context(), address.String())

	// then
	if !errors.Is(err, failure) || stdout.Len() != 0 {
		t.Fatalf("serve = %v, stdout = %q; want listen failure without readiness", err, stdout.String())
	}
}

func memoryProxy(t *testing.T) (*environment.Env, *idleListener) {
	t.Helper()
	env := environment.NewMemory()
	listener := &idleListener{closed: make(chan struct{})}
	env.Network = listenerNetwork{Network: env.Network, listener: listener, t: t}
	t.Cleanup(func() { _ = listener.Close() })
	return env, listener
}

type idleListener struct {
	closed chan struct{}
	once   sync.Once
}

func (l *idleListener) Accept() (net.Conn, error) {
	<-l.closed
	return nil, net.ErrClosed
}
func (l *idleListener) Close() error {
	l.once.Do(func() { close(l.closed) })
	return nil
}
func (l *idleListener) Addr() net.Addr { return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 12345} }

type listenerNetwork struct {
	network.Network
	listener *idleListener
	t        *testing.T
}

func (n listenerNetwork) Listen(address network.Address) (net.Listener, error) {
	if address.String() != "127.0.0.1:0" {
		n.t.Errorf("listen(%q)", address)
	}
	return n.listener, nil
}

type readinessWriter struct {
	written chan []byte
	err     error
}

func (w readinessWriter) Write(data []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	w.written <- bytes.Clone(data)
	return len(data), nil
}
