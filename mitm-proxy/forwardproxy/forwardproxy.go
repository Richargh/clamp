// Package forwardproxy provides an HTTP forward proxy with opaque HTTPS CONNECT
// tunnels. It does not decrypt HTTPS traffic or apply destination allowlists.
package forwardproxy

import (
	"net/url"

	implementation "github.com/richargh/clamp/mitm-proxy/internal/forwardproxy"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/environment"
	"golang.org/x/net/http/httpproxy"
)

type Options struct {
	// Proxy selects an upstream proxy. Nil captures host proxy settings at construction.
	// Return nil, nil from the function to connect directly.
	Proxy func(*url.URL) (*url.URL, error)
}

// ForwardHandler forwards HTTP requests and establishes opaque CONNECT tunnels.
type ForwardHandler = implementation.ForwardHandler

// NewHandler constructs a handler with OS networking and host proxy settings.
// Construction does not open a listener or print output. Action logs go to stdout.
// Private destinations are permitted.
// Serve does not close hijacked CONNECT tunnels during shutdown.
func NewHandler(options Options) *ForwardHandler {
	if options.Proxy == nil {
		options.Proxy = httpproxy.FromEnvironment().ProxyFunc()
	}
	return implementation.NewHandler(environment.NewOS(), implementation.Options(options))
}
