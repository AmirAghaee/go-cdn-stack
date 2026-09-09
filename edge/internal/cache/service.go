package cache

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/cdn"
)

type CDNStore interface {
	FindByDomain(string) (cdn.CDN, bool)
}

type Store interface {
	Get(string) (Entry, bool)
	Set(string, Entry) error
}

type OriginClient interface {
	Fetch(context.Context, OriginRequest) (OriginResponse, error)
}

type Metrics interface {
	RecordRequest(host, method, status string, duration time.Duration)
	RecordCacheHit(host string)
	RecordCacheMiss(host string)
	RecordOriginRequest(host, status string, duration time.Duration)
	RecordBytesReceived(host string, count int)
	RecordBytesSent(host, cacheStatus string, count int)
	RecordError(host, kind string)
}

type Service struct {
	cdns    CDNStore
	cache   Store
	origin  OriginClient
	metrics Metrics
	now     func() time.Time
}

func NewService(cdns CDNStore, store Store, origin OriginClient, metrics Metrics) *Service {
	return &Service{cdns: cdns, cache: store, origin: origin, metrics: metrics, now: time.Now}
}

func (s *Service) Handle(ctx context.Context, request Request) Response {
	startedAt := s.now()
	host := cdn.NormalizeDomain(request.Host)
	item, ok := s.cdns.FindByDomain(host)
	if !ok {
		s.metrics.RecordError(host, "unknown_host")
		response := Response{
			StatusCode:  http.StatusBadGateway,
			Header:      map[string][]string{"Content-Type": {"text/plain; charset=utf-8"}},
			Body:        []byte(fmt.Sprintf("Unknown host: %s", request.Host)),
			CacheStatus: "error",
		}
		s.recordResponse(request, host, response, startedAt)
		return response
	}

	if request.Method != http.MethodGet {
		response := s.fetch(ctx, request, item, "proxy")
		s.recordResponse(request, host, response, startedAt)
		return response
	}

	cacheKey := host + request.URI
	if cached, found := s.cache.Get(cacheKey); found && s.now().Before(cached.ExpiresAt) {
		s.metrics.RecordCacheHit(host)
		response := Response{StatusCode: cached.StatusCode, Header: cloneHeader(cached.Header), Body: cached.Body, CacheStatus: "hit"}
		s.recordResponse(request, host, response, startedAt)
		return response
	}

	s.metrics.RecordCacheMiss(host)
	response := s.fetch(ctx, request, item, "miss")
	if response.StatusCode < http.StatusBadRequest && isCacheableContentType(firstHeader(response.Header, "Content-Type")) {
		entry := Entry{
			StatusCode: response.StatusCode,
			Header:     cloneHeader(response.Header),
			Body:       append([]byte(nil), response.Body...),
			ExpiresAt:  s.now().Add(time.Duration(item.CacheTTL()) * time.Second),
		}
		if err := s.cache.Set(cacheKey, entry); err != nil {
			s.metrics.RecordError(host, "cache_write")
		}
	}
	s.recordResponse(request, host, response, startedAt)
	return response
}

func (s *Service) fetch(ctx context.Context, request Request, item cdn.CDN, cacheStatus string) Response {
	startedAt := s.now()
	originResponse, err := s.origin.Fetch(ctx, OriginRequest{
		Method: request.Method, Origin: item.Origin(), URI: request.URI, Host: request.Host,
		Header: cloneHeader(request.Header), Body: request.Body, ClientIP: request.ClientIP,
	})
	if err != nil {
		s.metrics.RecordError(item.Domain(), "origin_request")
		s.metrics.RecordOriginRequest(item.Domain(), "error", s.now().Sub(startedAt))
		return Response{
			StatusCode:  http.StatusBadGateway,
			Header:      map[string][]string{"Content-Type": {"text/plain; charset=utf-8"}},
			Body:        []byte("Error forwarding request"),
			CacheStatus: cacheStatus,
		}
	}

	status := strconv.Itoa(originResponse.StatusCode)
	s.metrics.RecordOriginRequest(item.Domain(), status, s.now().Sub(startedAt))
	s.metrics.RecordBytesReceived(item.Domain(), len(originResponse.Body))
	return Response{
		StatusCode:  originResponse.StatusCode,
		Header:      cloneHeader(originResponse.Header),
		Body:        originResponse.Body,
		CacheStatus: cacheStatus,
	}
}

func (s *Service) recordResponse(request Request, host string, response Response, startedAt time.Time) {
	s.metrics.RecordRequest(host, request.Method, strconv.Itoa(response.StatusCode), s.now().Sub(startedAt))
	s.metrics.RecordBytesSent(host, response.CacheStatus, len(response.Body))
}

func cloneHeader(header map[string][]string) map[string][]string {
	cloned := make(map[string][]string, len(header))
	for key, values := range header {
		cloned[key] = append([]string(nil), values...)
	}
	return cloned
}

func firstHeader(header map[string][]string, key string) string {
	for existingKey, values := range header {
		if strings.EqualFold(existingKey, key) && len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

func isCacheableContentType(contentType string) bool {
	cacheable := []string{"image/", "font/", "text/css", "text/javascript", "application/javascript", "application/x-javascript", "video/", "audio/"}
	for _, prefix := range cacheable {
		if strings.HasPrefix(contentType, prefix) {
			return true
		}
	}
	return false
}
