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
}

func (f *fakeCacheStore) Get(key string) (Entry, bool)     { item, ok := f.items[key]; return item, ok }
func (f *fakeCacheStore) Set(key string, item Entry) error { f.items[key] = item; return nil }

type fakeOrigin struct {
	request OriginRequest
}

func (f *fakeOrigin) Fetch(_ context.Context, request OriginRequest) (OriginResponse, error) {
	f.request = request
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
	if _, ok := store.items["cdn.example/asset?id=7"]; !ok {
		t.Fatal("cache entry was not stored with the complete request URI")
	}
}
