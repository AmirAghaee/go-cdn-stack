package originhttp

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/cache"
)

type Client struct {
	httpClient *http.Client
}

func New(httpClient *http.Client) *Client { return &Client{httpClient: httpClient} }

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
	upstreamRequest.Header = http.Header(cloneHeader(request.Header))
	upstreamRequest.Header.Set("X-Forwarded-Host", request.Host)
	upstreamRequest.Header.Set("X-Forwarded-For", request.ClientIP)

	response, err := c.httpClient.Do(upstreamRequest)
	if err != nil {
		return cache.OriginResponse{}, fmt.Errorf("execute origin request: %w", err)
	}
	return cache.OriginResponse{
		StatusCode:    response.StatusCode,
		Header:        cloneHeader(response.Header),
		Body:          response.Body,
		ContentLength: response.ContentLength,
	}, nil
}

func cloneHeader(header map[string][]string) map[string][]string {
	cloned := make(map[string][]string, len(header))
	for key, values := range header {
		cloned[key] = append([]string(nil), values...)
	}
	return cloned
}
