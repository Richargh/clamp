package network_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/richargh/clamp/mitm-proxy/internal/kernel/network"
	"golang.org/x/net/nettest"
)

func TestOSNetworkContract(t *testing.T) {
	testNetworkContract(t, func() network.Network { return network.NewOS() })
}

func TestMemoryNetworkContract(t *testing.T) {
	testNetworkContract(t, func() network.Network { return network.NewMemory(network.MemoryOptions{}) })
}

func testNetworkContract(t *testing.T, newNetwork func() network.Network) {
	t.Helper()
	for _, encrypted := range []bool{false, true} {
		name := "HTTP"
		if encrypted {
			name = "TLS"
		}
		t.Run(name, func(t *testing.T) { testNetworkHTTP(t, newNetwork(), encrypted) })
	}
	t.Run("lookup", func(t *testing.T) {
		ports := newNetwork()
		for _, value := range []string{"127.0.0.1", "::1", "localhost"} {
			t.Run(value, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
				defer cancel()
				addresses, err := ports.LookupIP(ctx, mustHost(t, value))
				if err != nil {
					t.Fatal(err)
				}
				if len(addresses) == 0 {
					t.Fatal("lookup returned no addresses")
				}
				for _, address := range addresses {
					if !address.IsLoopback() {
						t.Fatalf("lookup(%q) returned non-loopback address %v", value, address)
					}
					if value != "localhost" && address.Unmap() != netip.MustParseAddr(value) {
						t.Fatalf("lookup(%q) returned %v", value, address)
					}
				}
			})
		}
	})
	t.Run("canceled dial", func(t *testing.T) {
		ports := newNetwork()
		listener := listenLoopback(t, ports)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		conn, err := ports.Dial(ctx, mustAddress(t, listener.Addr().String()))
		if conn != nil {
			_ = conn.Close()
			t.Fatal("canceled dial opened a connection")
		}
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("dial = %v, want context.Canceled", err)
		}
	})
	t.Run("invalid values", func(t *testing.T) {
		ports := newNetwork()
		if listener, err := ports.Listen(network.Address{}); err == nil || listener != nil {
			t.Fatalf("zero-value listen = %v, %v", listener, err)
		}
		for _, address := range []network.Address{{}, mustAddress(t, "127.0.0.1:0")} {
			if conn, err := ports.Dial(t.Context(), address); err == nil || conn != nil {
				t.Fatalf("invalid dial = %v, %v", conn, err)
			}
		}
		if addresses, err := ports.LookupIP(t.Context(), network.Host{}); err == nil || addresses != nil {
			t.Fatalf("zero-value lookup = %v, %v", addresses, err)
		}
	})
	t.Run("listener ownership and closure", func(t *testing.T) {
		ports := newNetwork()
		listener := listenLoopback(t, ports)
		address := mustAddress(t, listener.Addr().String())
		if duplicate, err := ports.Listen(address); err == nil {
			_ = duplicate.Close()
			t.Fatal("duplicate listener succeeded")
		}
		done := make(chan error, 1)
		go func() {
			conn, err := listener.Accept()
			if conn != nil {
				_ = conn.Close()
			}
			done <- err
		}()
		if err := listener.Close(); err != nil {
			t.Fatal(err)
		}
		select {
		case err := <-done:
			if !errors.Is(err, net.ErrClosed) {
				t.Fatalf("accept = %v, want net.ErrClosed", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("close did not unblock accept")
		}
	})
	t.Run("connection semantics", func(t *testing.T) {
		nettest.TestConn(t, func() (net.Conn, net.Conn, func(), error) {
			ports := newNetwork()
			listener, err := ports.Listen(mustAddress(t, "127.0.0.1:0"))
			if err != nil {
				return nil, nil, nil, err
			}
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			client, err := ports.Dial(ctx, mustAddress(t, listener.Addr().String()))
			if err != nil {
				_ = listener.Close()
				return nil, nil, nil, err
			}
			server, err := listener.Accept()
			if err != nil {
				_ = client.Close()
				_ = listener.Close()
				return nil, nil, nil, err
			}
			return client, server, func() {
				_ = client.Close()
				_ = server.Close()
				_ = listener.Close()
			}, nil
		})
	})
}

func listenLoopback(t *testing.T, ports network.Network) net.Listener {
	t.Helper()
	listener, err := ports.Listen(mustAddress(t, "127.0.0.1:0"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	if mustAddress(t, listener.Addr().String()).Port() == 0 {
		t.Fatal("listen did not allocate a port")
	}
	return listener
}

func testNetworkHTTP(t *testing.T, ports network.Network, encrypted bool) {
	t.Helper()
	server := &httptest.Server{
		Listener: listenLoopback(t, ports),
		Config: &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, r.Method+" "+r.URL.RequestURI()+" ")
			_, _ = io.Copy(w, r.Body)
		})},
	}
	if encrypted {
		server.StartTLS()
	} else {
		server.Start()
	}
	defer server.Close()
	roots := x509.NewCertPool()
	if encrypted {
		roots.AddCert(server.Certificate())
	}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12},
		DialContext: func(ctx context.Context, _, address string) (net.Conn, error) {
			target, err := network.ParseAddress(address)
			if err != nil {
				return nil, err
			}
			return ports.Dial(ctx, target)
		},
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
	response, err := client.Post(server.URL+"/test?query=kept", "text/plain", strings.NewReader("payload"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusCreated || string(body) != "POST /test?query=kept payload" {
		t.Fatalf("response = %d %q", response.StatusCode, body)
	}
	if encrypted && !response.TLS.PeerCertificates[0].Equal(server.Certificate()) {
		t.Fatal("unexpected peer certificate")
	}
}
