package forwardproxy

import (
	"log"
	"net/http"
	"net/url"

	"github.com/elazarl/goproxy"
	"github.com/richargh/clamp/mitm-proxy/internal/alloweddomains"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/network"
)

type destinationPolicyDecision struct {
	address network.Address
	allowed bool
}

type destinationPolicy struct {
	allowed alloweddomains.AllowedDomains
}

func configureDestinationPolicy(proxy *goproxy.ProxyHttpServer, allowed alloweddomains.AllowedDomains, logger *log.Logger) {
	policy := destinationPolicy{allowed: allowed}
	proxy.OnRequest().Do(policy)
	proxy.OnRequest().HandleConnect(withConnectLogging(policy, logger))
}

func decideDestinationPolicy(target *url.URL, allowed alloweddomains.AllowedDomains) destinationPolicyDecision {
	address, err := network.AddressFromURL(target)
	return destinationPolicyDecision{
		address: address,
		allowed: err == nil && allowed != nil && allowed.IsAllowed(address),
	}
}

func (p destinationPolicy) Handle(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	decision := decideDestinationPolicy(req.URL, p.allowed)
	ctx.UserData = decision
	if !decision.allowed {
		return req, goproxy.NewResponse(req, goproxy.ContentTypeText, http.StatusForbidden, "Destination blocked\n")
	}
	return req, nil
}

func (p destinationPolicy) HandleConnect(authority string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
	decision := decideDestinationPolicy(&url.URL{Scheme: "https", Host: authority}, p.allowed)
	ctx.UserData = decision
	if !decision.address.IsValid() {
		return goproxy.RejectConnect, authority
	}
	destination := decision.address.String()
	if !decision.allowed {
		ctx.Resp = goproxy.NewResponse(ctx.Req, goproxy.ContentTypeText, http.StatusForbidden, "Destination blocked\n")
		return goproxy.RejectConnect, destination
	}
	return goproxy.OkConnect, destination
}
