package cache

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/cdn"
)

const (
	statusClientClosedRequest = 499
	unknownHostMetricLabel    = "unknown"
)

type CDNStore interface {
	FindByDomain(string) (cdn.CDN, bool)
}

type Store interface {
	Get(string) (Entry, bool)
	Begin(string, EntryMetadata) (PendingEntry, error)
}

type OriginClient interface {
	Fetch(context.Context, OriginRequest) (OriginResponse, error)
}

type Metrics interface {
	RecordRequest(host, method, status string, duration time.Duration)
	RecordCacheHit(host string)
	RecordCacheMiss(host string)
	RecordOriginRequest(host, status string, duration time.Duration)
	RecordBytesReceived(host string, count int64)
	RecordBytesSent(host, cacheStatus string, count int64)
	RecordError(host, kind string)
}

type Service struct {
	cdns          CDNStore
	cache         Store
	origin        OriginClient
	metrics       Metrics
	maxObjectSize int64
	now           func() time.Time
	flights       flightGroup
}

func NewService(cdns CDNStore, store Store, origin OriginClient, metrics Metrics, maxObjectSize int64) *Service {
	return &Service{
		cdns:          cdns,
		cache:         store,
		origin:        origin,
		metrics:       metrics,
		maxObjectSize: maxObjectSize,
		now:           time.Now,
	}
}

func (s *Service) Handle(ctx context.Context, request Request) Response {
	startedAt := s.now()
	host := cdn.NormalizeDomain(request.Host)
	item, ok := s.cdns.FindByDomain(host)
	if !ok {
		s.metrics.RecordError(unknownHostMetricLabel, "unknown_host")
		response := Response{
			StatusCode:  http.StatusBadGateway,
			Header:      map[string][]string{"Content-Type": {"text/plain; charset=utf-8"}},
			Body:        io.NopCloser(strings.NewReader(fmt.Sprintf("Unknown host: %s", request.Host))),
			CacheStatus: "error",
		}
		return s.trackResponse(request, unknownHostMetricLabel, response, startedAt)
	}
	host = item.Domain()

	if request.Method != http.MethodGet {
		response := s.fetch(ctx, request, item, "proxy")
		return s.trackResponse(request, host, response, startedAt)
	}

	if shouldBypassCache(request) {
		response := s.fetch(ctx, request, item, "bypass")
		return s.trackResponse(request, host, response, startedAt)
	}

	key := cacheKey(item, request.URI, request.Header)
	missRecorded := false
	for {
		if cached, found := s.cache.Get(key); found {
			if s.now().Before(cached.ExpiresAt) {
				s.metrics.RecordCacheHit(host)
				response := Response{StatusCode: cached.StatusCode, Header: cloneHeader(cached.Header), Body: cached.Body, CacheStatus: "hit"}
				return s.trackResponse(request, host, response, startedAt)
			}
			_ = cached.Body.Close()
		}

		if !missRecorded {
			s.metrics.RecordCacheMiss(host)
			missRecorded = true
		}

		flight, leader := s.flights.join(key)
		if !leader {
			select {
			case <-flight.done:
				continue
			case <-ctx.Done():
				response := Response{
					StatusCode:  statusClientClosedRequest,
					Header:      map[string][]string{"Content-Type": {"text/plain; charset=utf-8"}},
					Body:        io.NopCloser(strings.NewReader("Client closed request")),
					CacheStatus: "error",
				}
				return s.trackResponse(request, host, response, startedAt)
			}
		}

		response := s.fetch(ctx, request, item, "miss")
		if expiresAt, cacheable := cacheExpiry(s.now(), item.CacheTTL(), response); cacheable &&
			(s.maxObjectSize <= 0 || responseContentLength(response) <= s.maxObjectSize) {
			entry := EntryMetadata{
				StatusCode: response.StatusCode,
				Header:     cloneHeader(response.Header),
				ExpiresAt:  expiresAt,
			}
			pending, err := s.cache.Begin(key, entry)
			if err != nil {
				s.metrics.RecordError(host, "cache_write")
			} else {
				response.Body = newCacheFillBody(response.Body, pending, s.maxObjectSize, func() {
					s.metrics.RecordError(host, "cache_write")
				})
				response.Body = newFlightBody(response.Body, func() { s.flights.finish(key, flight) })
				return s.trackResponse(request, host, response, startedAt)
			}
		}
		s.flights.finish(key, flight)
		return s.trackResponse(request, host, response, startedAt)
	}
}

func responseContentLength(response Response) int64 {
	value := firstHeader(response.Header, "Content-Length")
	if value == "" {
		return -1
	}
	size, err := strconv.ParseInt(value, 10, 64)
	if err != nil || size < 0 {
		return -1
	}
	return size
}

func (s *Service) trackResponse(request Request, host string, response Response, startedAt time.Time) Response {
	response.Body = newMeteredBody(response.Body, func(count int64, _ error) {
		s.metrics.RecordRequest(host, request.Method, strconv.Itoa(response.StatusCode), s.now().Sub(startedAt))
		s.metrics.RecordBytesSent(host, response.CacheStatus, count)
	})
	return response
}

func (s *Service) fetch(ctx context.Context, request Request, item cdn.CDN, cacheStatus string) Response {
	startedAt := s.now()
	originResponse, err := s.origin.Fetch(ctx, OriginRequest{
		Method: request.Method, Origin: item.Origin(), URI: request.URI, Host: request.Host,
		Header: cloneHeader(request.Header), Body: request.Body, ClientIP: request.ClientIP,
		ForwardedFor: request.ForwardedFor, Scheme: request.Scheme,
	})
	if err != nil {
		s.metrics.RecordError(item.Domain(), "origin_request")
		s.metrics.RecordOriginRequest(item.Domain(), "error", s.now().Sub(startedAt))
		return Response{
			StatusCode:  http.StatusBadGateway,
			Header:      map[string][]string{"Content-Type": {"text/plain; charset=utf-8"}},
			Body:        io.NopCloser(strings.NewReader("Error forwarding request")),
			CacheStatus: cacheStatus,
		}
	}

	status := strconv.Itoa(originResponse.StatusCode)
	originResponse.Body = newMeteredBody(originResponse.Body, func(count int64, err error) {
		s.metrics.RecordOriginRequest(item.Domain(), status, s.now().Sub(startedAt))
		s.metrics.RecordBytesReceived(item.Domain(), count)
		if err != nil && !errors.Is(err, io.EOF) {
			s.metrics.RecordError(item.Domain(), "origin_response")
		}
	})
	header := cloneHeader(originResponse.Header)
	if originResponse.ContentLength >= 0 && firstHeader(header, "Content-Length") == "" {
		header["Content-Length"] = []string{strconv.FormatInt(originResponse.ContentLength, 10)}
	}
	return Response{
		StatusCode:  originResponse.StatusCode,
		Header:      header,
		Body:        originResponse.Body,
		CacheStatus: cacheStatus,
	}
}
