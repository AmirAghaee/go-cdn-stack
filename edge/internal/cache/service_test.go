package cache

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/cdn"
)

type fakeCDNStore struct{ item cdn.CDN }

func (f fakeCDNStore) FindByDomain(domain string) (cdn.CDN, bool) {
	return f.item, domain == f.item.Domain() && f.item.IsActive()
}

type fakeCacheStore struct {
	items     map[string]Entry
	gets      int
	begins    int
	aborts    int
	writeErr  error
	commitErr error
}

func (f *fakeCacheStore) Get(key string) (Entry, bool) {
	f.gets++
	item, ok := f.items[key]
	return item, ok
}

func (f *fakeCacheStore) Begin(key string, item EntryMetadata) (PendingEntry, error) {
	f.begins++
	return &fakePendingEntry{store: f, key: key, metadata: item}, nil
}

type fakePendingEntry struct {
	store    *fakeCacheStore
	key      string
	metadata EntryMetadata
	body     bytes.Buffer
}

func (f *fakePendingEntry) Write(data []byte) (int, error) {
	if f.store.writeErr != nil {
		return 0, f.store.writeErr
	}
	return f.body.Write(data)
}
func (f *fakePendingEntry) Commit() error {
	if f.store.commitErr != nil {
		return f.store.commitErr
	}
	f.store.items[f.key] = Entry{
		StatusCode: f.metadata.StatusCode,
		Header:     f.metadata.Header,
		Body:       io.NopCloser(bytes.NewReader(f.body.Bytes())),
		ExpiresAt:  f.metadata.ExpiresAt,
		StoredAt:   f.metadata.StoredAt,
		InitialAge: f.metadata.InitialAge,
	}
	return nil
}
func (f *fakePendingEntry) Abort() error {
	f.store.aborts++
	return nil
}

type fakeOrigin struct {
	request   OriginRequest
	responses []OriginResponse
	calls     int
}

func (f *fakeOrigin) Fetch(_ context.Context, request OriginRequest) (OriginResponse, error) {
	f.request = request
	if f.calls < len(f.responses) {
		response := f.responses[f.calls]
		f.calls++
		return response, nil
	}
	f.calls++
	return originResponse("image"), nil
}

type fakeMetrics struct{}

func (fakeMetrics) RecordRequest(string, string, string, time.Duration) {}
func (fakeMetrics) RecordCacheHit(string)                               {}
func (fakeMetrics) RecordCacheMiss(string)                              {}
func (fakeMetrics) RecordOriginRequest(string, string, time.Duration)   {}
func (fakeMetrics) RecordBytesReceived(string, int64)                   {}
func (fakeMetrics) RecordBytesSent(string, string, int64)               {}
func (fakeMetrics) RecordError(string, string)                          {}

type hostRecordingMetrics struct {
	hosts []string
}

func (m *hostRecordingMetrics) record(host string) {
	m.hosts = append(m.hosts, host)
}

func (m *hostRecordingMetrics) RecordRequest(host, _, _ string, _ time.Duration) {
	m.record(host)
}
func (m *hostRecordingMetrics) RecordCacheHit(host string)  { m.record(host) }
func (m *hostRecordingMetrics) RecordCacheMiss(host string) { m.record(host) }
func (m *hostRecordingMetrics) RecordOriginRequest(host, _ string, _ time.Duration) {
	m.record(host)
}
func (m *hostRecordingMetrics) RecordBytesReceived(host string, _ int64) { m.record(host) }
func (m *hostRecordingMetrics) RecordBytesSent(host, _ string, _ int64)  { m.record(host) }
func (m *hostRecordingMetrics) RecordError(host, _ string)               { m.record(host) }

type concurrentCacheStore struct {
	mu    sync.Mutex
	items map[string]concurrentCacheEntry
}

func TestUnknownHostsUseOneMetricLabel(t *testing.T) {
	item := mustCDN(t, 60)
	metrics := &hostRecordingMetrics{}
	service := NewService(
		fakeCDNStore{item: item},
		&fakeCacheStore{items: make(map[string]Entry)},
		&fakeOrigin{},
		metrics,
		0,
	)

	for _, host := range []string{"attacker-one.example", "attacker-two.example:8080", "127.0.0.1"} {
		response := service.Handle(context.Background(), Request{Method: http.MethodGet, Host: host, URI: "/"})
		if response.StatusCode != http.StatusBadGateway {
			t.Fatalf("response status for host %q = %d", host, response.StatusCode)
		}
		_ = readResponseBody(t, response)
	}

	if len(metrics.hosts) == 0 {
		t.Fatal("no metrics were recorded")
	}
	for _, host := range metrics.hosts {
		if host != unknownHostMetricLabel {
			t.Fatalf("metric host = %q, want %q", host, unknownHostMetricLabel)
		}
	}
}

func TestKnownHostMetricsUseConfiguredDomain(t *testing.T) {
	item := mustCDN(t, 60)
	metrics := &hostRecordingMetrics{}
	service := NewService(
		fakeCDNStore{item: item},
		&fakeCacheStore{items: make(map[string]Entry)},
		&fakeOrigin{},
		metrics,
		0,
	)

	response := service.Handle(context.Background(), Request{
		Method: http.MethodPost,
		Host:   "CDN.EXAMPLE.:443",
		URI:    "/",
	})
	_ = readResponseBody(t, response)

	if len(metrics.hosts) == 0 {
		t.Fatal("no metrics were recorded")
	}
	for _, host := range metrics.hosts {
		if host != item.Domain() {
			t.Fatalf("metric host = %q, want configured domain %q", host, item.Domain())
		}
	}
}

type concurrentCacheEntry struct {
	metadata EntryMetadata
	body     []byte
}

func (s *concurrentCacheStore) Get(key string) (Entry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, found := s.items[key]
	if !found {
		return Entry{}, false
	}
	return Entry{
		StatusCode: item.metadata.StatusCode,
		Header:     cloneHeader(item.metadata.Header),
		Body:       io.NopCloser(bytes.NewReader(item.body)),
		ExpiresAt:  item.metadata.ExpiresAt,
		StoredAt:   item.metadata.StoredAt,
		InitialAge: item.metadata.InitialAge,
	}, true
}

func (s *concurrentCacheStore) Begin(key string, metadata EntryMetadata) (PendingEntry, error) {
	return &concurrentPendingEntry{store: s, key: key, metadata: metadata}, nil
}

type concurrentPendingEntry struct {
	store    *concurrentCacheStore
	key      string
	metadata EntryMetadata
	body     bytes.Buffer
}

func (e *concurrentPendingEntry) Write(data []byte) (int, error) { return e.body.Write(data) }
func (e *concurrentPendingEntry) Abort() error                   { return nil }
func (e *concurrentPendingEntry) Commit() error {
	e.store.mu.Lock()
	defer e.store.mu.Unlock()
	e.store.items[e.key] = concurrentCacheEntry{
		metadata: e.metadata,
		body:     append([]byte(nil), e.body.Bytes()...),
	}
	return nil
}

type blockingOrigin struct {
	mu      sync.Mutex
	calls   int
	release chan struct{}
}

func (o *blockingOrigin) Fetch(ctx context.Context, _ OriginRequest) (OriginResponse, error) {
	o.mu.Lock()
	o.calls++
	o.mu.Unlock()
	return OriginResponse{
		StatusCode:    http.StatusOK,
		Header:        map[string][]string{"Content-Type": {"image/png"}},
		Body:          &blockingBody{ctx: ctx, release: o.release, reader: strings.NewReader("shared-origin-body")},
		ContentLength: int64(len("shared-origin-body")),
	}, nil
}

func (o *blockingOrigin) callCount() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.calls
}

type blockingBody struct {
	ctx     context.Context
	release <-chan struct{}
	reader  *strings.Reader
}

func (b *blockingBody) Read(buffer []byte) (int, error) {
	select {
	case <-b.release:
		return b.reader.Read(buffer)
	case <-b.ctx.Done():
		return 0, b.ctx.Err()
	}
}

func (*blockingBody) Close() error { return nil }

func TestCacheMissPreservesRequestURIAndHeaders(t *testing.T) {
	item, err := cdn.New("id", "cdn.example", "http://origin.example/base", true, 60)
	if err != nil {
		t.Fatal(err)
	}
	origin := &fakeOrigin{}
	store := &fakeCacheStore{items: make(map[string]Entry)}
	service := NewService(fakeCDNStore{item: item}, store, origin, fakeMetrics{}, 0)

	response := service.Handle(context.Background(), Request{
		Method: http.MethodGet, Host: "cdn.example", URI: "/asset?id=7",
		Header: map[string][]string{"X-Test-Header": {"preserved"}}, Body: io.NopCloser(strings.NewReader("")),
	})

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", response.StatusCode)
	}
	if got := readResponseBody(t, response); got != "image" {
		t.Fatalf("response body = %q", got)
	}
	if origin.request.URI != "/asset?id=7" {
		t.Fatalf("origin URI = %q", origin.request.URI)
	}
	if got := origin.request.Header["X-Test-Header"][0]; got != "preserved" {
		t.Fatalf("origin header = %q", got)
	}
	if _, ok := store.items[cacheKey(item, "/asset?id=7", nil)]; !ok {
		t.Fatal("cache entry was not stored with the complete request URI")
	}
}

func TestConfigurationChangesDoNotReuseCachedContent(t *testing.T) {
	oldItem, err := cdn.New("old-id", "cdn.example", "http://old-origin.example", true, 60)
	if err != nil {
		t.Fatal(err)
	}
	tests := map[string]struct {
		id     string
		origin string
		ttl    uint
	}{
		"origin changed":       {id: "old-id", origin: "http://new-origin.example", ttl: 60},
		"TTL disabled":         {id: "old-id", origin: "http://old-origin.example", ttl: 0},
		"domain was recreated": {id: "new-id", origin: "http://old-origin.example", ttl: 60},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			currentItem, createErr := cdn.New(test.id, oldItem.Domain(), test.origin, true, test.ttl)
			if createErr != nil {
				t.Fatal(createErr)
			}
			store := &fakeCacheStore{items: map[string]Entry{
				cacheKey(oldItem, "/asset", nil): {
					StatusCode: http.StatusOK,
					Header:     map[string][]string{"Content-Type": {"image/png"}},
					Body:       stream("old-content"),
					ExpiresAt:  time.Now().Add(time.Hour),
				},
			}}
			origin := &fakeOrigin{responses: []OriginResponse{originResponse("current-content")}}
			service := NewService(fakeCDNStore{item: currentItem}, store, origin, fakeMetrics{}, 0)

			response := service.Handle(context.Background(), Request{
				Method: http.MethodGet,
				Host:   currentItem.Domain(),
				URI:    "/asset",
			})

			if got := readResponseBody(t, response); got != "current-content" {
				t.Fatalf("response body = %q, want current origin content", got)
			}
			if response.CacheStatus != "miss" {
				t.Fatalf("cache status = %q, want miss", response.CacheStatus)
			}
			if origin.calls != 1 {
				t.Fatalf("origin calls = %d, want 1", origin.calls)
			}
			if test.ttl == 0 && store.begins != 0 {
				t.Fatalf("cache begins = %d, want 0 when caching is disabled", store.begins)
			}
		})
	}
}

func TestConcurrentMissesForSameKeyUseOneOriginRequest(t *testing.T) {
	item := mustCDN(t, 60)
	store := &concurrentCacheStore{items: make(map[string]concurrentCacheEntry)}
	origin := &blockingOrigin{release: make(chan struct{})}
	service := NewService(fakeCDNStore{item: item}, store, origin, fakeMetrics{}, 0)
	request := Request{Method: http.MethodGet, Host: item.Domain(), URI: "/asset"}

	leader := service.Handle(context.Background(), request)
	followerDone := make(chan Response, 1)
	go func() { followerDone <- service.Handle(context.Background(), request) }()

	select {
	case <-followerDone:
		t.Fatal("follower returned before the cache fill completed")
	case <-time.After(50 * time.Millisecond):
	}
	if calls := origin.callCount(); calls != 1 {
		t.Fatalf("origin calls before release = %d", calls)
	}

	close(origin.release)
	if body := readResponseBody(t, leader); body != "shared-origin-body" {
		t.Fatalf("leader body = %q", body)
	}
	select {
	case follower := <-followerDone:
		if body := readResponseBody(t, follower); follower.CacheStatus != "hit" || body != "shared-origin-body" {
			t.Fatalf("follower cache status=%q body=%q", follower.CacheStatus, body)
		}
	case <-time.After(time.Second):
		t.Fatal("follower did not resume after the cache fill completed")
	}
	if calls := origin.callCount(); calls != 1 {
		t.Fatalf("origin calls = %d", calls)
	}
}

func TestCanceledFollowerDoesNotCancelLeader(t *testing.T) {
	item := mustCDN(t, 60)
	store := &concurrentCacheStore{items: make(map[string]concurrentCacheEntry)}
	origin := &blockingOrigin{release: make(chan struct{})}
	service := NewService(fakeCDNStore{item: item}, store, origin, fakeMetrics{}, 0)
	request := Request{Method: http.MethodGet, Host: item.Domain(), URI: "/asset"}

	leader := service.Handle(context.Background(), request)
	followerCtx, cancelFollower := context.WithCancel(context.Background())
	cancelFollower()
	follower := service.Handle(followerCtx, request)
	if follower.StatusCode != statusClientClosedRequest {
		t.Fatalf("canceled follower status = %d", follower.StatusCode)
	}
	_ = readResponseBody(t, follower)
	if calls := origin.callCount(); calls != 1 {
		t.Fatalf("origin calls after follower cancellation = %d", calls)
	}

	close(origin.release)
	if body := readResponseBody(t, leader); body != "shared-origin-body" {
		t.Fatalf("leader body = %q", body)
	}
	response := service.Handle(context.Background(), request)
	if body := readResponseBody(t, response); response.CacheStatus != "hit" || body != "shared-origin-body" {
		t.Fatalf("cached response status=%q body=%q", response.CacheStatus, body)
	}
	if calls := origin.callCount(); calls != 1 {
		t.Fatalf("origin calls = %d", calls)
	}
}

func TestSensitiveRequestsBypassCache(t *testing.T) {
	tests := map[string]map[string][]string{
		"authorization":           {"Authorization": {"Bearer secret"}},
		"authorization lowercase": {"authorization": {"Bearer secret"}},
		"cookie":                  {"Cookie": {"session=secret"}},
		"range":                   {"Range": {"bytes=0-99"}},
		"cache control no cache":  {"Cache-Control": {"no-cache"}},
		"cache control no store":  {"Cache-Control": {"max-age=60, NO-STORE"}},
		"cache control multiple":  {"cache-control": {"max-age=60", "no-cache"}},
		"request max age":         {"Cache-Control": {"max-age=10"}},
		"request min fresh":       {"Cache-Control": {"min-fresh=10"}},
		"pragma no cache":         {"Pragma": {"no-cache"}},
		"if match":                {"If-Match": {`"asset-v2"`}},
		"if none match":           {"If-None-Match": {`"asset-v1"`}},
		"if modified since":       {"If-Modified-Since": {time.Now().UTC().Format(http.TimeFormat)}},
		"if unmodified since":     {"If-Unmodified-Since": {time.Now().UTC().Format(http.TimeFormat)}},
		"if range":                {"If-Range": {`"asset-v1"`}},
	}

	for name, header := range tests {
		t.Run(name, func(t *testing.T) {
			item := mustCDN(t, 60)
			key := cacheKey(item, "/private", header)
			store := &fakeCacheStore{items: map[string]Entry{
				key: {
					StatusCode: http.StatusOK,
					Header:     map[string][]string{"Content-Type": {"image/png"}},
					Body:       stream("cached-private-response"),
					ExpiresAt:  time.Now().Add(time.Minute),
				},
			}}
			origin := &fakeOrigin{}
			service := NewService(fakeCDNStore{item: item}, store, origin, fakeMetrics{}, 0)

			response := service.Handle(context.Background(), Request{
				Method: http.MethodGet,
				Host:   item.Domain(),
				URI:    "/private",
				Header: header,
			})

			if body := readResponseBody(t, response); response.CacheStatus != "bypass" || body != "image" {
				t.Fatalf("response cache status=%q body=%q", response.CacheStatus, body)
			}
			if store.gets != 0 || store.begins != 0 {
				t.Fatalf("cache operations: gets=%d begins=%d", store.gets, store.begins)
			}
			if origin.calls != 1 {
				t.Fatalf("origin calls = %d", origin.calls)
			}
		})
	}
}

func TestConditionalRequestUsesOriginPreconditionResultInsteadOfCachedResponse(t *testing.T) {
	item := mustCDN(t, 60)
	request := Request{
		Method: http.MethodGet, Host: item.Domain(), URI: "/asset",
		Header: map[string][]string{"If-Match": {`"asset-v2"`}},
	}
	store := &fakeCacheStore{items: map[string]Entry{
		cacheKey(item, request.URI, request.Header): {
			StatusCode: http.StatusOK,
			Header:     map[string][]string{"Content-Type": {"image/png"}, "ETag": {`"asset-v1"`}},
			Body:       stream("cached"),
			ExpiresAt:  time.Now().Add(time.Minute),
		},
	}}
	origin := &fakeOrigin{responses: []OriginResponse{{
		StatusCode:    http.StatusPreconditionFailed,
		Header:        map[string][]string{"Content-Type": {"text/plain"}},
		Body:          stream("precondition failed"),
		ContentLength: int64(len("precondition failed")),
	}}}
	service := NewService(fakeCDNStore{item: item}, store, origin, fakeMetrics{}, 0)

	response := service.Handle(context.Background(), request)
	body := readResponseBody(t, response)
	if response.StatusCode != http.StatusPreconditionFailed || response.CacheStatus != "bypass" || body != "precondition failed" {
		t.Fatalf("response status=%d cache status=%q body=%q", response.StatusCode, response.CacheStatus, body)
	}
	if values := origin.request.Header["If-Match"]; len(values) != 1 || values[0] != `"asset-v2"` {
		t.Fatalf("origin If-Match = %v", values)
	}
}

func TestOnlyIfCachedServesFreshCacheHitWithoutOrigin(t *testing.T) {
	item := mustCDN(t, 60)
	request := Request{
		Method: http.MethodGet, Host: item.Domain(), URI: "/asset",
		Header: map[string][]string{"Cache-Control": {"only-if-cached"}},
	}
	store := &fakeCacheStore{items: map[string]Entry{
		cacheKey(item, request.URI, request.Header): {
			StatusCode: http.StatusOK,
			Header:     map[string][]string{"Content-Type": {"image/png"}},
			Body:       stream("cached"),
			ExpiresAt:  time.Now().Add(time.Minute),
		},
	}}
	origin := &fakeOrigin{}
	service := NewService(fakeCDNStore{item: item}, store, origin, fakeMetrics{}, 0)

	response := service.Handle(context.Background(), request)
	if body := readResponseBody(t, response); response.CacheStatus != "hit" || body != "cached" {
		t.Fatalf("response cache status=%q body=%q", response.CacheStatus, body)
	}
	if origin.calls != 0 {
		t.Fatalf("origin calls = %d", origin.calls)
	}
}

func TestCacheHitReplacesAgeWithCurrentStoredAge(t *testing.T) {
	now := time.Date(2026, time.September, 9, 12, 0, 30, 0, time.UTC)
	item := mustCDN(t, 60)
	request := Request{Method: http.MethodGet, Host: item.Domain(), URI: "/asset"}
	store := &fakeCacheStore{items: map[string]Entry{
		cacheKey(item, request.URI, request.Header): {
			StatusCode: http.StatusOK,
			Header:     map[string][]string{"Content-Type": {"image/png"}, "aGe": {"15"}},
			Body:       stream("cached"),
			ExpiresAt:  now.Add(time.Minute),
			StoredAt:   now.Add(-20 * time.Second),
			InitialAge: 15 * time.Second,
		},
	}}
	service := NewService(fakeCDNStore{item: item}, store, &fakeOrigin{}, fakeMetrics{}, 0)
	service.now = func() time.Time { return now }

	response := service.Handle(context.Background(), request)
	if body := readResponseBody(t, response); body != "cached" {
		t.Fatalf("response body = %q", body)
	}
	if values := headerValues(response.Header, "Age"); len(values) != 1 || values[0] != "35" {
		t.Fatalf("response Age = %v, headers = %v", values, response.Header)
	}
	if _, found := response.Header["aGe"]; found {
		t.Fatalf("stored Age casing was replayed: %v", response.Header)
	}
}

func TestOnlyIfCachedNeverContactsOriginWhenCacheCannotSatisfyRequest(t *testing.T) {
	tests := map[string]struct {
		header  map[string][]string
		cached  bool
		expires time.Time
	}{
		"miss": {
			header: map[string][]string{"Cache-Control": {"only-if-cached"}},
		},
		"stale": {
			header: map[string][]string{"Cache-Control": {"only-if-cached"}},
			cached: true, expires: time.Now().Add(-time.Minute),
		},
		"requires revalidation": {
			header: map[string][]string{"Cache-Control": {"only-if-cached, no-cache"}},
			cached: true, expires: time.Now().Add(time.Minute),
		},
		"has validator": {
			header: map[string][]string{"Cache-Control": {"only-if-cached"}, "If-Match": {`"asset-v2"`}},
			cached: true, expires: time.Now().Add(time.Minute),
		},
		"has freshness constraint": {
			header: map[string][]string{"Cache-Control": {"only-if-cached, min-fresh=10"}},
			cached: true, expires: time.Now().Add(time.Minute),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			item := mustCDN(t, 60)
			request := Request{Method: http.MethodGet, Host: item.Domain(), URI: "/asset", Header: test.header}
			store := &fakeCacheStore{items: make(map[string]Entry)}
			if test.cached {
				store.items[cacheKey(item, request.URI, request.Header)] = Entry{
					StatusCode: http.StatusOK,
					Header:     map[string][]string{"Content-Type": {"image/png"}},
					Body:       stream("cached"),
					ExpiresAt:  test.expires,
				}
			}
			origin := &fakeOrigin{}
			service := NewService(fakeCDNStore{item: item}, store, origin, fakeMetrics{}, 0)

			response := service.Handle(context.Background(), request)
			body := readResponseBody(t, response)
			if response.StatusCode != http.StatusGatewayTimeout || response.CacheStatus != "miss" {
				t.Fatalf("response status=%d cache status=%q body=%q", response.StatusCode, response.CacheStatus, body)
			}
			if origin.calls != 0 || store.begins != 0 {
				t.Fatalf("origin calls=%d cache begins=%d", origin.calls, store.begins)
			}
		})
	}
}

func TestResponseCacheAdmissionRejectsUnsafeResponses(t *testing.T) {
	tests := map[string]OriginResponse{
		"set cookie": {
			StatusCode: http.StatusOK,
			Header:     map[string][]string{"Content-Type": {"image/png"}, "Set-Cookie": {"session=secret"}},
		},
		"private": {
			StatusCode: http.StatusOK,
			Header:     map[string][]string{"Content-Type": {"image/png"}, "Cache-Control": {"PRIVATE"}},
		},
		"no store": {
			StatusCode: http.StatusOK,
			Header:     map[string][]string{"Content-Type": {"image/png"}, "Cache-Control": {"public, no-store"}},
		},
		"no store lowercase multi value": {
			StatusCode: http.StatusOK,
			Header:     map[string][]string{"Content-Type": {"image/png"}, "cache-control": {"public", "NO-STORE"}},
		},
		"no cache": {
			StatusCode: http.StatusOK,
			Header:     map[string][]string{"Content-Type": {"image/png"}, "Cache-Control": {"no-cache"}},
		},
		"partial response": {
			StatusCode: http.StatusPartialContent,
			Header:     map[string][]string{"Content-Type": {"image/png"}},
		},
		"unsupported vary": {
			StatusCode: http.StatusOK,
			Header:     map[string][]string{"Content-Type": {"image/png"}, "Vary": {"Accept-Language"}},
		},
		"wildcard vary": {
			StatusCode: http.StatusOK,
			Header:     map[string][]string{"Content-Type": {"image/png"}, "Vary": {"*"}},
		},
	}

	for name, originResponse := range tests {
		t.Run(name, func(t *testing.T) {
			item := mustCDN(t, 60)
			store := &fakeCacheStore{items: make(map[string]Entry)}
			originResponse.Body = stream("origin")
			originResponse.ContentLength = int64(len("origin"))
			origin := &fakeOrigin{responses: []OriginResponse{originResponse}}
			service := NewService(fakeCDNStore{item: item}, store, origin, fakeMetrics{}, 0)

			response := service.Handle(context.Background(), Request{
				Method: http.MethodGet,
				Host:   item.Domain(),
				URI:    "/asset",
			})

			_ = readResponseBody(t, response)
			if response.CacheStatus != "miss" || store.begins != 0 {
				t.Fatalf("cache status=%q begins=%d", response.CacheStatus, store.begins)
			}
		})
	}
}

func TestAcceptEncodingVariantsDoNotShareEntries(t *testing.T) {
	item := mustCDN(t, 60)
	store := &fakeCacheStore{items: make(map[string]Entry)}
	origin := &fakeOrigin{responses: []OriginResponse{
		{
			StatusCode: http.StatusOK,
			Header: map[string][]string{
				"Content-Type":     {"text/css"},
				"Content-Encoding": {"gzip"},
				"Vary":             {"Accept-Encoding"},
			},
			Body:          stream("gzip-response"),
			ContentLength: int64(len("gzip-response")),
		},
		{
			StatusCode:    http.StatusOK,
			Header:        map[string][]string{"Content-Type": {"text/css"}, "Vary": {"accept-encoding"}},
			Body:          stream("identity-response"),
			ContentLength: int64(len("identity-response")),
		},
	}}
	service := NewService(fakeCDNStore{item: item}, store, origin, fakeMetrics{}, 0)

	gzipRequest := Request{
		Method: http.MethodGet,
		Host:   item.Domain(),
		URI:    "/asset.css",
		Header: map[string][]string{"Accept-Encoding": {"GZIP"}},
	}
	identityRequest := Request{Method: http.MethodGet, Host: item.Domain(), URI: "/asset.css"}

	if response := service.Handle(context.Background(), gzipRequest); readResponseBody(t, response) != "gzip-response" {
		t.Fatal("unexpected gzip response body")
	}
	if response := service.Handle(context.Background(), identityRequest); readResponseBody(t, response) != "identity-response" {
		t.Fatal("unexpected identity response body")
	}
	response := service.Handle(context.Background(), gzipRequest)
	body := readResponseBody(t, response)
	if response.CacheStatus != "hit" || body != "gzip-response" {
		t.Fatalf("cached gzip response status=%q body=%q", response.CacheStatus, body)
	}
	if origin.calls != 2 {
		t.Fatalf("origin calls = %d", origin.calls)
	}
}

func TestLegacyCacheKeyIsNotRead(t *testing.T) {
	item := mustCDN(t, 60)
	store := &fakeCacheStore{items: map[string]Entry{
		item.Domain() + "/asset": {
			StatusCode: http.StatusOK,
			Header:     map[string][]string{"Content-Type": {"image/png"}},
			Body:       stream("legacy-private-response"),
			ExpiresAt:  time.Now().Add(time.Minute),
		},
	}}
	origin := &fakeOrigin{}
	service := NewService(fakeCDNStore{item: item}, store, origin, fakeMetrics{}, 0)

	response := service.Handle(context.Background(), Request{Method: http.MethodGet, Host: item.Domain(), URI: "/asset"})

	body := readResponseBody(t, response)
	if response.CacheStatus != "miss" || body != "image" {
		t.Fatalf("response cache status=%q body=%q", response.CacheStatus, body)
	}
	if origin.calls != 1 {
		t.Fatalf("origin calls = %d", origin.calls)
	}
	if _, found := store.items[item.Domain()+"/asset"]; !found {
		t.Fatal("legacy entry should be left for normal cleanup")
	}
}

func TestCacheEntryIsCommittedOnlyAfterOriginEOF(t *testing.T) {
	item := mustCDN(t, 60)
	store := &fakeCacheStore{items: make(map[string]Entry)}
	origin := &fakeOrigin{responses: []OriginResponse{originResponse("streamed")}}
	service := NewService(fakeCDNStore{item: item}, store, origin, fakeMetrics{}, 0)
	request := Request{Method: http.MethodGet, Host: item.Domain(), URI: "/asset"}

	response := service.Handle(context.Background(), request)
	key := cacheKey(item, request.URI, request.Header)
	if _, found := store.items[key]; found {
		t.Fatal("cache entry became visible before the origin body reached EOF")
	}
	if body := readResponseBody(t, response); body != "streamed" {
		t.Fatalf("response body = %q", body)
	}
	if _, found := store.items[key]; !found {
		t.Fatal("cache entry was not committed after EOF")
	}
}

func TestUnknownLengthResponseCrossingLimitContinuesWithoutCaching(t *testing.T) {
	item := mustCDN(t, 60)
	store := &fakeCacheStore{items: make(map[string]Entry)}
	origin := &fakeOrigin{responses: []OriginResponse{{
		StatusCode:    http.StatusOK,
		Header:        map[string][]string{"Content-Type": {"image/png"}},
		Body:          stream("larger-than-limit"),
		ContentLength: -1,
	}}}
	service := NewService(fakeCDNStore{item: item}, store, origin, fakeMetrics{}, 5)

	response := service.Handle(context.Background(), Request{Method: http.MethodGet, Host: item.Domain(), URI: "/asset"})
	if body := readResponseBody(t, response); body != "larger-than-limit" {
		t.Fatalf("response body = %q", body)
	}
	if store.begins != 1 || store.aborts != 1 || len(store.items) != 0 {
		t.Fatalf("cache begins=%d aborts=%d items=%d", store.begins, store.aborts, len(store.items))
	}
}

func TestKnownOversizedResponseDoesNotStartCacheWrite(t *testing.T) {
	item := mustCDN(t, 60)
	store := &fakeCacheStore{items: make(map[string]Entry)}
	origin := &fakeOrigin{responses: []OriginResponse{originResponse("larger-than-limit")}}
	service := NewService(fakeCDNStore{item: item}, store, origin, fakeMetrics{}, 5)

	response := service.Handle(context.Background(), Request{Method: http.MethodGet, Host: item.Domain(), URI: "/asset"})
	if body := readResponseBody(t, response); body != "larger-than-limit" {
		t.Fatalf("response body = %q", body)
	}
	if store.begins != 0 || len(store.items) != 0 {
		t.Fatalf("cache begins=%d items=%d", store.begins, len(store.items))
	}
}

func TestEarlyCloseAbortsCacheFill(t *testing.T) {
	item := mustCDN(t, 60)
	store := &fakeCacheStore{items: make(map[string]Entry)}
	origin := &fakeOrigin{responses: []OriginResponse{originResponse("unfinished")}}
	service := NewService(fakeCDNStore{item: item}, store, origin, fakeMetrics{}, 0)

	response := service.Handle(context.Background(), Request{Method: http.MethodGet, Host: item.Domain(), URI: "/asset"})
	buffer := make([]byte, 1)
	if _, err := response.Body.Read(buffer); err != nil {
		t.Fatalf("read first byte: %v", err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatalf("close response body: %v", err)
	}
	if store.aborts != 1 || len(store.items) != 0 {
		t.Fatalf("cache aborts=%d items=%d", store.aborts, len(store.items))
	}
}

func TestOriginReadFailureAbortsCacheFill(t *testing.T) {
	item := mustCDN(t, 60)
	store := &fakeCacheStore{items: make(map[string]Entry)}
	origin := &fakeOrigin{responses: []OriginResponse{{
		StatusCode:    http.StatusOK,
		Header:        map[string][]string{"Content-Type": {"image/png"}},
		Body:          io.NopCloser(io.MultiReader(strings.NewReader("partial"), errorReader{})),
		ContentLength: -1,
	}}}
	service := NewService(fakeCDNStore{item: item}, store, origin, fakeMetrics{}, 0)

	response := service.Handle(context.Background(), Request{Method: http.MethodGet, Host: item.Domain(), URI: "/asset"})
	if _, err := io.ReadAll(response.Body); err == nil {
		t.Fatal("expected origin read error")
	}
	_ = response.Body.Close()
	if store.aborts != 1 || len(store.items) != 0 {
		t.Fatalf("cache aborts=%d items=%d", store.aborts, len(store.items))
	}
}

func TestCacheWriteFailureDoesNotInterruptOriginResponse(t *testing.T) {
	item := mustCDN(t, 60)
	store := &fakeCacheStore{items: make(map[string]Entry), writeErr: errors.New("disk full")}
	origin := &fakeOrigin{responses: []OriginResponse{originResponse("complete-origin-body")}}
	service := NewService(fakeCDNStore{item: item}, store, origin, fakeMetrics{}, 0)

	response := service.Handle(context.Background(), Request{Method: http.MethodGet, Host: item.Domain(), URI: "/asset"})
	if body := readResponseBody(t, response); body != "complete-origin-body" {
		t.Fatalf("response body = %q", body)
	}
	if store.aborts != 1 || len(store.items) != 0 {
		t.Fatalf("cache aborts=%d items=%d", store.aborts, len(store.items))
	}
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, errors.New("origin read failed") }

func originResponse(body string) OriginResponse {
	return OriginResponse{
		StatusCode:    http.StatusOK,
		Header:        map[string][]string{"Content-Type": {"image/png"}},
		Body:          stream(body),
		ContentLength: int64(len(body)),
	}
}

func stream(body string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(body))
}

func readResponseBody(t *testing.T, response Response) string {
	t.Helper()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatalf("close response body: %v", err)
	}
	return string(body)
}

func mustCDN(t *testing.T, ttl uint) cdn.CDN {
	t.Helper()
	item, err := cdn.New("id", "cdn.example", "http://origin.example", true, ttl)
	if err != nil {
		t.Fatal(err)
	}
	return item
}
