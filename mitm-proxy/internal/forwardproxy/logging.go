package forwardproxy

import (
	"io"
	"log"
	"net"
	"net/http"
	"strconv"

	"github.com/elazarl/goproxy"
)

func newActionLogger(writer io.Writer) *log.Logger {
	if writer == nil {
		writer = io.Discard
	}
	return log.New(writer, "", log.LstdFlags)
}

func configureActionLogging(proxy *goproxy.ProxyHttpServer, writer io.Writer) {
	logger := newActionLogger(writer)
	// GoProxy warnings can contain raw upstream errors and request URLs.
	proxy.Logger = log.New(io.Discard, "", 0)
	proxy.OnResponse().DoFunc(func(response *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		destination := httpLogDestination(ctx.Req)
		if response == nil || ctx.Error != nil {
			logger.Printf("HTTP method=%q destination=%q outcome=failed", ctx.Req.Method, destination)
		} else {
			logger.Printf("HTTP method=%q destination=%q status=%d", ctx.Req.Method, destination, response.StatusCode)
		}
		return response
	})
	proxy.OnRequest().HandleConnectFunc(func(authority string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
		action, destination := validateTunnelDestination(authority, ctx)
		if action.Action == goproxy.ConnectReject {
			logger.Printf("HTTPS CONNECT outcome=rejected")
		}
		return action, destination
	})
	dial := proxy.ConnectDialWithReq
	proxy.ConnectDialWithReq = func(req *http.Request, networkName, address string) (net.Conn, error) {
		conn, err := dial(req, networkName, address)
		outcome := "connected"
		if err != nil {
			outcome = "failed"
		}
		logger.Printf("HTTPS CONNECT destination=%q outcome=%s", address, outcome)
		return conn, err
	}
}

func httpLogDestination(req *http.Request) string {
	defaultPort := 80
	if req.URL.Scheme == "https" {
		defaultPort = 443
	}
	host, port, err := splitAuthority(req.URL.Host, defaultPort)
	if err != nil {
		return "invalid"
	}
	return net.JoinHostPort(host, strconv.Itoa(port))
}
