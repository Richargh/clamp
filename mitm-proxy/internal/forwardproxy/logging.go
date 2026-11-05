package forwardproxy

import (
	"io"
	"log"
	"net"
	"net/http"

	"github.com/elazarl/goproxy"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/network"
)

func newActionLogger(writer io.Writer) *log.Logger {
	if writer == nil {
		writer = io.Discard
	}
	return log.New(writer, "", log.LstdFlags)
}

func configureActionLogging(proxy *goproxy.ProxyHttpServer, logger *log.Logger) {
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

func withConnectLogging(policy goproxy.HttpsHandler, logger *log.Logger) goproxy.HttpsHandler {
	return goproxy.FuncHttpsHandler(func(authority string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
		action, destination := policy.HandleConnect(authority, ctx)
		decision, ok := ctx.UserData.(destinationPolicyDecision)
		if ok && !decision.allowed {
			if !decision.address.IsValid() {
				logger.Printf("HTTPS CONNECT outcome=rejected")
			} else {
				logger.Printf("HTTPS CONNECT destination=%q outcome=rejected", decision.address.String())
			}
		}
		return action, destination
	})
}

func httpLogDestination(req *http.Request) string {
	address, err := network.AddressFromURL(req.URL)
	if err != nil {
		return "invalid"
	}
	return address.String()
}
