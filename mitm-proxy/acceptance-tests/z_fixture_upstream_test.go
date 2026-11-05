package acceptance_test

import (
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/richargh/clamp/mitm-proxy/internal/alloweddomains"
	"github.com/richargh/clamp/mitm-proxy/internal/forwardproxy"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/environment"
	"github.com/stretchr/testify/require"
)

func DirectProxy(*url.URL) (*url.URL, error) {
	return nil, nil
}

type upstreamProxySpy struct {
	URL      string
	requests atomic.Int32
}

func NewUpstreamProxySpy(t *testing.T, env *environment.Env) *upstreamProxySpy {
	t.Helper()
	spy := &upstreamProxySpy{}
	handler := forwardproxy.NewHandler(env, forwardproxy.Options{Proxy: DirectProxy, AllowedDomains: alloweddomains.AllowAll()})
	server := Server(t, env.Network, "127.0.0.1:0", false, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		spy.requests.Add(1)
		handler.ServeHTTP(w, r)
	}))
	spy.URL = server.URL
	return spy
}

// URLWithDetails keeps the spy's address and adds the URL details supplied by the test.
func (s *upstreamProxySpy) URLWithDetails(t *testing.T, details url.URL) *url.URL {
	t.Helper()
	base, err := url.Parse(s.URL)
	require.NoError(t, err)
	details.Scheme = base.Scheme
	details.Host = base.Host
	return &details
}

func (s *upstreamProxySpy) RequestCount() int32 {
	return s.requests.Load()
}
