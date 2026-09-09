package cache

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
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
	if _, ok := store.items[cacheKey("cdn.example", "/asset?id=7", nil)]; !ok {
		t.Fatal("cache entry was not stored with the complete request URI")
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
		"pragma no cache":         {"Pragma": {"no-cache"}},
	}

	for name, header := range tests {
		t.Run(name, func(t *testing.T) {
			item := mustCDN(t, 60)
			key := cacheKey(item.Domain(), "/private", header)
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
	key := cacheKey(item.Domain(), request.URI, request.Header)
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
