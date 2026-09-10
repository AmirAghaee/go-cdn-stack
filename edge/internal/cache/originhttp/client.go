package originhttp

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/cache"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/platform/proxyheaders"
)

type Client struct {
	httpClient *http.Client
}

// New creates an origin client that pins each connection to an allowed DNS
// result and returns redirects to the downstream client instead of following
// them. allowedCIDRs are explicit exceptions for deployments with private
// origins; an empty list permits only public destinations.
func New(timeout time.Duration, allowedCIDRs []string) (*Client, error) {
	allowedNetworks, err := parseAllowedCIDRs(allowedCIDRs)
	if err != nil {
		return nil, err
	}

	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = (&safeDialer{
		resolver:        net.DefaultResolver,
		dialContext:     dialer.DialContext,
		allowedNetworks: allowedNetworks,
	}).DialContext

	return newClient(&http.Client{Timeout: timeout, Transport: transport}), nil
}

func newClient(httpClient *http.Client) *Client {
	clone := *httpClient
	clone.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &Client{httpClient: &clone}
}

func (c *Client) Fetch(ctx context.Context, request cache.OriginRequest) (cache.OriginResponse, error) {
	origin, err := url.Parse(request.Origin)
	if err != nil {
		return cache.OriginResponse{}, fmt.Errorf("parse origin URL: %w", err)
	}
	reference, err := url.ParseRequestURI(request.URI)
	if err != nil {
		return cache.OriginResponse{}, fmt.Errorf("parse request URI: %w", err)
	}
	rawPath := strings.TrimRight(origin.EscapedPath(), "/") + reference.EscapedPath()
	origin.Path = strings.TrimRight(origin.Path, "/") + reference.Path
	origin.RawPath = rawPath
	origin.RawQuery = reference.RawQuery
	target := origin

	upstreamRequest, err := http.NewRequestWithContext(ctx, request.Method, target.String(), request.Body)
	if err != nil {
		return cache.OriginResponse{}, fmt.Errorf("create origin request: %w", err)
	}
	upstreamRequest.Header = proxyheaders.EndToEnd(request.Header)
	proxyheaders.RemoveForwarding(upstreamRequest.Header)
	upstreamRequest.Header.Set("X-Forwarded-Host", request.Host)
	forwardedFor := request.ForwardedFor
	if forwardedFor == "" {
		forwardedFor = request.ClientIP
	}
	if forwardedFor != "" {
		upstreamRequest.Header.Set("X-Forwarded-For", forwardedFor)
	}
	if request.Scheme == "http" || request.Scheme == "https" {
		upstreamRequest.Header.Set("X-Forwarded-Proto", request.Scheme)
	}

	response, err := c.httpClient.Do(upstreamRequest)
	if err != nil {
		return cache.OriginResponse{}, fmt.Errorf("execute origin request: %w", err)
	}
	return cache.OriginResponse{
		StatusCode:    response.StatusCode,
		Header:        proxyheaders.EndToEnd(response.Header),
		Body:          response.Body,
		ContentLength: response.ContentLength,
	}, nil
}
