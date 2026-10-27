package forwardproxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/richargh/clamp/mitm-proxy/internal/kernel/environment"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/network"
)

type TransportOptions struct {
	// Proxy must be non-nil. Return nil, nil to connect directly.
	Proxy func(*url.URL) (*url.URL, error)
}

func NewTransport(env *environment.Env, options TransportOptions) *http.Transport {
	return &http.Transport{
		Proxy:                 hostProxySelector(env, options),
		DialContext:           transportDialContext(env),
		TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		IdleConnTimeout:       60 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
	}
}

func hostProxySelector(env *environment.Env, options TransportOptions) func(*http.Request) (*url.URL, error) {
	logger := newActionLogger(env.Stdout)
	return func(req *http.Request) (*url.URL, error) {
		proxy, err := options.Proxy(req.URL)
		if err == nil && proxy != nil {
			if proxy.Scheme == "http" && proxy.User != nil {
				logger.Printf("UPSTREAM proxy=%q warning=%q", proxy.Host, "proxy credentials are sent over unencrypted HTTP")
			} else {
				logger.Printf("UPSTREAM proxy=%q", proxy.Host)
			}
		}
		return proxy, err
	}
}

func transportDialContext(env *environment.Env) func(context.Context, string, string) (net.Conn, error) {
	dial := destinationDialer(env)
	return func(ctx context.Context, networkName, address string) (net.Conn, error) {
		target, err := network.ParseAddress(address)
		if err != nil {
			return nil, err
		}
		if target.Port() == 0 {
			return nil, fmt.Errorf("invalid destination")
		}
		if networkName != "tcp" {
			return nil, fmt.Errorf("unsupported transport network")
		}
		return dial(ctx, target)
	}
}

func destinationDialer(env *environment.Env) network.Dialer {
	return func(ctx context.Context, address network.Address) (net.Conn, error) {
		target, err := resolveIP(ctx, env, address)
		if err != nil {
			return nil, err
		}
		return env.Network.Dial(ctx, target)
	}
}
