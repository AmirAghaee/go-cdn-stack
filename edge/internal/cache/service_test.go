package cache

import (
	"context"
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
	items map[string]Entry
	gets  int
	sets  int
}

func (f *fakeCacheStore) Get(key string) (Entry, bool) {
	f.gets++
	item, ok := f.items[key]
	return item, ok
}

func (f *fakeCacheStore) Set(key string, item Entry) error {
	f.sets++
	f.items[key] = item
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
	return OriginResponse{StatusCode: http.StatusOK, Header: map[string][]string{"Content-Type": {"image/png"}}, Body: []byte("image")}, nil
}

type fakeMetrics struct{}

func (fakeMetrics) RecordRequest(string, string, string, time.Duration) {}
func (fakeMetrics) RecordCacheHit(string)                               {}
func (fakeMetrics) RecordCacheMiss(string)                              {}
func (fakeMetrics) RecordOriginRequest(string, string, time.Duration)   {}
func (fakeMetrics) RecordBytesReceived(string, int)                     {}
func (fakeMetrics) RecordBytesSent(string, string, int)                 {}
func (fakeMetrics) RecordError(string, string)                          {}

func TestCacheMissPreservesRequestURIAndHeaders(t *testing.T) {
	item, err := cdn.New("id", "cdn.example", "http://origin.example/base", true, 60)
	if err != nil {
		t.Fatal(err)
	}
	origin := &fakeOrigin{}
	store := &fakeCacheStore{items: make(map[string]Entry)}
	service := NewService(fakeCDNStore{item: item}, store, origin, fakeMetrics{})

	response := service.Handle(context.Background(), Request{
		Method: http.MethodGet, Host: "cdn.example", URI: "/asset?id=7",
		Header: map[string][]string{"X-Test-Header": {"preserved"}}, Body: io.NopCloser(strings.NewReader("")),
	})

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", response.StatusCode)
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
		"authorization":          {"Authorization": {"Bearer secret"}},
		"authorization lowercase": {"authorization": {"Bearer secret"}},
		"cookie":                 {"Cookie": {"session=secret"}},
		"range":                  {"Range": {"bytes=0-99"}},
		"cache control no cache": {"Cache-Control": {"no-cache"}},
		"cache control no store": {"Cache-Control": {"max-age=60, NO-STORE"}},
		"cache control multiple": {"cache-control": {"max-age=60", "no-cache"}},
		"pragma no cache":        {"Pragma": {"no-cache"}},
	}

	for name, header := range tests {
		t.Run(name, func(t *testing.T) {
			item := mustCDN(t, 60)
			key := cacheKey(item.Domain(), "/private", header)
			store := &fakeCacheStore{items: map[string]Entry{
				key: {
					StatusCode: http.StatusOK,
					Header:     map[string][]string{"Content-Type": {"image/png"}},
					Body:       []byte("cached-private-response"),
					ExpiresAt:  time.Now().Add(time.Minute),
				},
			}}
			origin := &fakeOrigin{}
			service := NewService(fakeCDNStore{item: item}, store, origin, fakeMetrics{})

			response := service.Handle(context.Background(), Request{
				Method: http.MethodGet,
				Host:   item.Domain(),
				URI:    "/private",
				Header: header,
			})

			if response.CacheStatus != "bypass" || string(response.Body) != "image" {
				t.Fatalf("response cache status=%q body=%q", response.CacheStatus, response.Body)
			}
			if store.gets != 0 || store.sets != 0 {
				t.Fatalf("cache operations: gets=%d sets=%d", store.gets, store.sets)
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
			originResponse.Body = []byte("origin")
			origin := &fakeOrigin{responses: []OriginResponse{originResponse}}
			service := NewService(fakeCDNStore{item: item}, store, origin, fakeMetrics{})

			response := service.Handle(context.Background(), Request{
				Method: http.MethodGet,
				Host:   item.Domain(),
				URI:    "/asset",
			})

			if response.CacheStatus != "miss" || store.sets != 0 {
				t.Fatalf("cache status=%q sets=%d", response.CacheStatus, store.sets)
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
			Body: []byte("gzip-response"),
		},
		{
			StatusCode: http.StatusOK,
			Header:     map[string][]string{"Content-Type": {"text/css"}, "Vary": {"accept-encoding"}},
			Body:       []byte("identity-response"),
		},
	}}
	service := NewService(fakeCDNStore{item: item}, store, origin, fakeMetrics{})

	gzipRequest := Request{
		Method: http.MethodGet,
		Host:   item.Domain(),
		URI:    "/asset.css",
		Header: map[string][]string{"Accept-Encoding": {"GZIP"}},
	}
	identityRequest := Request{Method: http.MethodGet, Host: item.Domain(), URI: "/asset.css"}

	if response := service.Handle(context.Background(), gzipRequest); string(response.Body) != "gzip-response" {
		t.Fatalf("gzip response body = %q", response.Body)
	}
	if response := service.Handle(context.Background(), identityRequest); string(response.Body) != "identity-response" {
		t.Fatalf("identity response body = %q", response.Body)
	}
	response := service.Handle(context.Background(), gzipRequest)
	if response.CacheStatus != "hit" || string(response.Body) != "gzip-response" {
		t.Fatalf("cached gzip response status=%q body=%q", response.CacheStatus, response.Body)
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
			Body:       []byte("legacy-private-response"),
			ExpiresAt:  time.Now().Add(time.Minute),
		},
	}}
	origin := &fakeOrigin{}
	service := NewService(fakeCDNStore{item: item}, store, origin, fakeMetrics{})

	response := service.Handle(context.Background(), Request{Method: http.MethodGet, Host: item.Domain(), URI: "/asset"})

	if response.CacheStatus != "miss" || string(response.Body) != "image" {
		t.Fatalf("response cache status=%q body=%q", response.CacheStatus, response.Body)
	}
	if origin.calls != 1 {
		t.Fatalf("origin calls = %d", origin.calls)
	}
	if _, found := store.items[item.Domain()+"/asset"]; !found {
		t.Fatal("legacy entry should be left for normal cleanup")
	}
}

func mustCDN(t *testing.T, ttl uint) cdn.CDN {
	t.Helper()
	item, err := cdn.New("id", "cdn.example", "http://origin.example", true, ttl)
	if err != nil {
		t.Fatal(err)
	}
	return item
}
