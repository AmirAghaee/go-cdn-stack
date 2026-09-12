package originhttp

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/cache"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/platform/proxyheaders"
)

type Client struct {
	httpClient *http.Client
	active     chan struct{}
}

type Options struct {
	DialTimeout           time.Duration
	ResponseHeaderTimeout time.Duration
	IdleConnTimeout       time.Duration
	MaxIdleConns          int
	MaxIdleConnsPerHost   int
	MaxConcurrent         int
}

// New creates an origin client that pins each connection to an allowed DNS
// result and returns redirects to the downstream client instead of following
// them. allowedCIDRs are explicit exceptions for deployments with private
// origins; an empty list permits only public destinations.
func New(options Options, allowedCIDRs []string) (*Client, error) {
	if options.DialTimeout <= 0 || options.ResponseHeaderTimeout <= 0 || options.IdleConnTimeout <= 0 ||
		options.MaxIdleConns <= 0 || options.MaxIdleConnsPerHost <= 0 || options.MaxConcurrent <= 0 {
		return nil, fmt.Errorf("origin HTTP limits must be greater than zero")
	}
	allowedNetworks, err := parseAllowedCIDRs(allowedCIDRs)
	if err != nil {
		return nil, err
	}

	dialer := &net.Dialer{Timeout: options.DialTimeout, KeepAlive: 30 * time.Second}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.ResponseHeaderTimeout = options.ResponseHeaderTimeout
	transport.IdleConnTimeout = options.IdleConnTimeout
	transport.MaxIdleConns = options.MaxIdleConns
	transport.MaxIdleConnsPerHost = options.MaxIdleConnsPerHost
	transport.MaxConnsPerHost = options.MaxConcurrent
	transport.DialContext = (&safeDialer{
		resolver:        net.DefaultResolver,
		dialContext:     dialer.DialContext,
		allowedNetworks: allowedNetworks,
	}).DialContext

	return newClient(&http.Client{Transport: transport}, options.MaxConcurrent), nil
}

func newClient(httpClient *http.Client, maxConcurrent int) *Client {
	clone := *httpClient
	clone.Timeout = 0
	clone.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &Client{httpClient: &clone, active: make(chan struct{}, maxConcurrent)}
}

func (c *Client) Fetch(ctx context.Context, request cache.OriginRequest) (cache.OriginResponse, error) {
	select {
	case c.active <- struct{}{}:
	case <-ctx.Done():
		return cache.OriginResponse{}, fmt.Errorf("wait for origin request capacity: %w", ctx.Err())
	}
	release := sync.OnceFunc(func() { <-c.active })

	origin, err := url.Parse(request.Origin)
	if err != nil {
		release()
		return cache.OriginResponse{}, fmt.Errorf("parse origin URL: %w", err)
	}
	reference, err := url.ParseRequestURI(request.URI)
	if err != nil {
		release()
		return cache.OriginResponse{}, fmt.Errorf("parse request URI: %w", err)
	}
	rawPath := strings.TrimRight(origin.EscapedPath(), "/") + reference.EscapedPath()
	origin.Path = strings.TrimRight(origin.Path, "/") + reference.Path
	origin.RawPath = rawPath
	origin.RawQuery = reference.RawQuery
	target := origin

	upstreamRequest, err := http.NewRequestWithContext(ctx, request.Method, target.String(), request.Body)
	if err != nil {
		release()
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
		release()
		return cache.OriginResponse{}, fmt.Errorf("execute origin request: %w", err)
	}
	return cache.OriginResponse{
		StatusCode:    response.StatusCode,
		Header:        proxyheaders.EndToEnd(response.Header),
		Body:          &releaseBody{ReadCloser: response.Body, release: release},
		ContentLength: response.ContentLength,
	}, nil
}

func (c *Client) CloseIdleConnections() {
	c.httpClient.CloseIdleConnections()
}

type releaseBody struct {
	io.ReadCloser
	release func()
}

func (b *releaseBody) Read(buffer []byte) (int, error) {
	count, err := b.ReadCloser.Read(buffer)
	if err != nil {
		b.release()
	}
	return count, err
}

func (b *releaseBody) Close() error {
	err := b.ReadCloser.Close()
	b.release()
	return err
}
