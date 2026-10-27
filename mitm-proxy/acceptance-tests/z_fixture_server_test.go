package acceptance_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/richargh/clamp/mitm-proxy/internal/forwardproxy"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/environment"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/network"
)

func ServeHandler(t *testing.T, env *environment.Env, testee *forwardproxy.ForwardHandler) *url.URL {
	t.Helper()
	return serveHandler(t, env, testee.Serve)
}

func serveHandler(t *testing.T, env *environment.Env, serve func(context.Context, string) error) *url.URL {
	t.Helper()
	written := make(chan []byte, 1)
	stdout := env.Stdout
	env.Stdout = readyWriter{written: written}
	t.Cleanup(func() { env.Stdout = stdout })
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- serve(ctx, "127.0.0.1:0") }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil && !errors.Is(err, context.Canceled) {
				t.Errorf("serve = %v", err)
			}
		case <-time.After(15 * time.Second):
			t.Error("proxy shutdown timeout")
		}
	})
	select {
	case data := <-written:
		var record struct {
			Address string `json:"address"`
		}
		if err := json.Unmarshal(data, &record); err != nil {
			t.Fatal(err)
		}
		return &url.URL{Scheme: "http", Host: record.Address}
	case err := <-done:
		// Leave the result available for cleanup.
		done <- err
		t.Fatalf("serve exited before readiness: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("readiness timeout")
	}
	return nil
}

type readyWriter struct{ written chan []byte }

func (w readyWriter) Write(data []byte) (int, error) {
	w.written <- append([]byte(nil), data...)
	return len(data), nil
}

func Server(t *testing.T, ports network.Network, address string, encrypted bool, handler http.Handler) *httptest.Server {
	t.Helper()
	listener, err := ports.Listen(MustAddress(t, address))
	if err != nil {
		t.Fatal(err)
	}
	server := &httptest.Server{Listener: listener, Config: &http.Server{Handler: handler}}
	if encrypted {
		server.StartTLS()
	} else {
		server.Start()
	}
	t.Cleanup(server.Close)
	return server
}
