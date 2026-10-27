package acceptance_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/richargh/clamp/mitm-proxy/internal/kernel/environment"
	"github.com/stretchr/testify/require"
)

func ExecuteHTTPRequest(serve func(http.ResponseWriter, *http.Request), request *http.Request) {
	serve(httptest.NewRecorder(), request)
}

// ExecuteCONNECTRequest sends CONNECT and closes the connection without starting TLS.
func ExecuteCONNECTRequest(serve func(context.Context, string) error, t *testing.T, env *environment.Env, request *http.Request) {
	t.Helper()
	proxy := serveHandler(t, env, serve)
	conn := Connection(t, env, proxy)
	defer func() { _ = conn.Close() }()
	response, err := ExchangeRequest(conn, request)
	require.NoError(t, err)
	// A successful CONNECT response begins a tunnel, not a finite body.
	_ = conn.Close()
	_ = response.Body.Close()
}

func ExecuteHTTPSRequest(serve func(context.Context, string) error, t *testing.T, env *environment.Env, request *http.Request, destination *destinationSpy) {
	t.Helper()
	response := RecordHTTPSResponse(serve, t, env, request, destination)
	defer func() { _ = response.Body.Close() }()
	_, err := io.Copy(io.Discard, response.Body)
	require.NoError(t, err)
}

// RecordHTTPSResponse sends a request through the proxy with trust in the destination certificate.
// The caller must close the response body.
func RecordHTTPSResponse(serve func(context.Context, string) error, t *testing.T, env *environment.Env, request *http.Request, destination *destinationSpy) *http.Response {
	t.Helper()
	response, err := ExchangeHTTPSRequest(serve, t, env, request, destination)
	require.NoError(t, err)
	return response
}

// ExchangeHTTPSRequest returns request errors so tests can assert TLS failures.
// A nil trustedDestination leaves the destination certificate untrusted.
// The caller must close any returned response body.
func ExchangeHTTPSRequest(serve func(context.Context, string) error, t *testing.T, env *environment.Env, request *http.Request, trustedDestination *destinationSpy) (*http.Response, error) {
	t.Helper()
	proxy := serveHandler(t, env, serve)
	var trustedServer *httptest.Server
	if trustedDestination != nil {
		trustedServer = trustedDestination.server
	}
	client := ProxyClient(t, env, proxy, trustedServer)
	return client.Do(request)
}

func RecordHTTPResponse(serve func(http.ResponseWriter, *http.Request), request *http.Request) *http.Response {
	recorder := httptest.NewRecorder()
	serve(recorder, request)
	return recorder.Result()
}

func ReadAndCloseResponseBody(t *testing.T, response *http.Response) string {
	t.Helper()
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	return string(body)
}
