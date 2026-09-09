package cache

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/cdn"
)

const statusClientClosedRequest = 499

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
		s.metrics.RecordError(host, "unknown_host")
		response := Response{
			StatusCode:  http.StatusBadGateway,
			Header:      map[string][]string{"Content-Type": {"text/plain; charset=utf-8"}},
			Body:        io.NopCloser(strings.NewReader(fmt.Sprintf("Unknown host: %s", request.Host))),
			CacheStatus: "error",
		}
		return s.trackResponse(request, host, response, startedAt)
	}

	if request.Method != http.MethodGet {
		response := s.fetch(ctx, request, item, "proxy")
		return s.trackResponse(request, host, response, startedAt)
	}

	if shouldBypassCache(request) {
		response := s.fetch(ctx, request, item, "bypass")
		return s.trackResponse(request, host, response, startedAt)
	}

	key := cacheKey(host, request.URI, request.Header)
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

type flight struct {
	done chan struct{}
}

type flightGroup struct {
	mu      sync.Mutex
	flights map[string]*flight
}

func (g *flightGroup) join(key string) (*flight, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if existing := g.flights[key]; existing != nil {
		return existing, false
	}
	if g.flights == nil {
		g.flights = make(map[string]*flight)
	}
	created := &flight{done: make(chan struct{})}
	g.flights[key] = created
	return created, true
}

func (g *flightGroup) finish(key string, completed *flight) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.flights[key] != completed {
		return
	}
	delete(g.flights, key)
	close(completed.done)
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

type meteredBody struct {
	body     io.ReadCloser
	count    int64
	onFinish func(int64, error)
	once     sync.Once
}

func newMeteredBody(body io.ReadCloser, onFinish func(int64, error)) io.ReadCloser {
	return &meteredBody{body: body, onFinish: onFinish}
}

func (b *meteredBody) Read(buffer []byte) (int, error) {
	count, err := b.body.Read(buffer)
	b.count += int64(count)
	if err != nil {
		b.finish(err)
	}
	return count, err
}

func (b *meteredBody) Close() error {
	err := b.body.Close()
	b.finish(err)
	return err
}

func (b *meteredBody) finish(err error) {
	b.once.Do(func() { b.onFinish(b.count, err) })
}

type cacheFillBody struct {
	body          io.ReadCloser
	pending       PendingEntry
	maxObjectSize int64
	written       int64
	active        bool
	finished      bool
	onError       func()
}

type flightBody struct {
	body     io.ReadCloser
	onFinish func()
	once     sync.Once
}

func newFlightBody(body io.ReadCloser, onFinish func()) io.ReadCloser {
	return &flightBody{body: body, onFinish: onFinish}
}

func (b *flightBody) Read(buffer []byte) (int, error) {
	count, err := b.body.Read(buffer)
	if err != nil {
		b.once.Do(b.onFinish)
	}
	return count, err
}

func (b *flightBody) Close() error {
	err := b.body.Close()
	b.once.Do(b.onFinish)
	return err
}

func newCacheFillBody(body io.ReadCloser, pending PendingEntry, maxObjectSize int64, onError func()) io.ReadCloser {
	return &cacheFillBody{
		body: body, pending: pending, maxObjectSize: maxObjectSize,
		active: true, onError: onError,
	}
}

func (b *cacheFillBody) Read(buffer []byte) (int, error) {
	count, readErr := b.body.Read(buffer)
	if count > 0 && b.active {
		if b.maxObjectSize > 0 && b.written+int64(count) > b.maxObjectSize {
			b.abort(false)
		} else {
			written, writeErr := b.pending.Write(buffer[:count])
			b.written += int64(written)
			if writeErr != nil || written != count {
				b.abort(true)
			}
		}
	}

	if readErr == io.EOF {
		b.finished = true
		if b.active {
			b.active = false
			if err := b.pending.Commit(); err != nil {
				b.onError()
			}
		}
	} else if readErr != nil {
		b.abort(false)
	}
	return count, readErr
}

func (b *cacheFillBody) Close() error {
	if !b.finished {
		b.abort(false)
	}
	return b.body.Close()
}

func (b *cacheFillBody) abort(recordError bool) {
	if !b.active {
		return
	}
	b.active = false
	if err := b.pending.Abort(); err != nil || recordError {
		b.onError()
	}
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

func cloneHeader(header map[string][]string) map[string][]string {
	cloned := make(map[string][]string, len(header))
	for key, values := range header {
		cloned[key] = append([]string(nil), values...)
	}
	return cloned
}
