package forwardproxy

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/elazarl/goproxy"
	"github.com/richargh/clamp/mitm-proxy/internal/alloweddomains"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/environment"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/network"
)

type Options struct {
	// Proxy must be non-nil. Return nil, nil to connect directly.
	Proxy func(*url.URL) (*url.URL, error)
	// AllowedDomains must be explicit. A missing policy blocks every destination.
	AllowedDomains alloweddomains.AllowedDomains
}

// ForwardHandler forwards HTTP requests and establishes opaque CONNECT tunnels.
type ForwardHandler struct {
	handler http.Handler
	env     *environment.Env
}

// NewHandler constructs a handler without opening a listener.
// Action logs are written to env.Stdout.
func NewHandler(env *environment.Env, options Options) *ForwardHandler {
	transport := NewTransport(env, TransportOptions{Proxy: options.Proxy})
	return &ForwardHandler{handler: newForwardProxy(transport, env.Stdout, options.AllowedDomains), env: env}
}

func (h *ForwardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.handler.ServeHTTP(w, r)
}

// Serve opens listenAddress and writes a JSON readiness record to env.Stdout.
// It serves until ctx is canceled or the listener fails, then closes the listener.
// Cancellation permits active HTTP requests to finish, but does not close
// hijacked CONNECT tunnels.
func (h *ForwardHandler) Serve(ctx context.Context, listenAddress string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	address, err := network.ParseAddress(listenAddress)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	listener, err := h.env.Network.Listen(address)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer func() { _ = listener.Close() }()
	if err := writeReady(h.env, listener.Addr()); err != nil {
		return err
	}
	server := h.server()
	defer func() { _ = server.Close() }()
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return shutdownAndWait(server, done)
	}
}

func (h *ForwardHandler) server() *http.Server {
	return &http.Server{
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
}

func shutdownAndWait(server *http.Server, done <-chan error) error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		_ = server.Close()
		return fmt.Errorf("shutdown: %w", err)
	}
	err := <-done
	if err != nil && !errors.Is(err, net.ErrClosed) && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func newForwardProxy(transport *http.Transport, logWriter io.Writer, allowed alloweddomains.AllowedDomains) *goproxy.ProxyHttpServer {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = false
	proxy.Tr = transport
	logger := newActionLogger(logWriter)
	configureConnectDialer(proxy, transport)
	configureDestinationPolicy(proxy, allowed, logger)
	configureActionLogging(proxy, logger)
	return proxy
}

func configureConnectDialer(proxy *goproxy.ProxyHttpServer, transport *http.Transport) {
	proxy.ConnectDial = nil
	dialDirect := transport.DialContext
	proxy.ConnectDialWithReq = func(req *http.Request, network, address string) (net.Conn, error) {
		hostProxy, err := transport.Proxy(&http.Request{URL: &url.URL{Scheme: "https", Host: address}})
		if err != nil {
			return nil, err
		}
		if hostProxy == nil {
			return dialDirect(req.Context(), network, address)
		}
		dialViaHostProxy := proxy.NewConnectDialToProxyWithHandler(hostProxy.String(), func(connectReq *http.Request) {
			if hostProxy.User != nil {
				password, _ := hostProxy.User.Password()
				credentials := hostProxy.User.Username() + ":" + password
				connectReq.Header.Set("Proxy-Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(credentials)))
			}
		})
		if dialViaHostProxy == nil {
			return nil, fmt.Errorf("unsupported host proxy scheme")
		}
		return dialViaHostProxy(network, address)
	}
}
