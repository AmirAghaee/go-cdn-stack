package originhttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/cache"
)

func TestFetchPreservesOriginBasePathQueryAndHeaders(t *testing.T) {
	var path, query, header string
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, query, header = r.URL.Path, r.URL.RawQuery, r.Header.Get("X-Test-Header")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("created"))
	}))
	defer origin.Close()

	client := New(&http.Client{Timeout: time.Second})
	response, err := client.Fetch(context.Background(), cache.OriginRequest{
		Method: http.MethodGet, Origin: origin.URL + "/base", URI: "/asset?id=7",
		Host: "cdn.example", Header: map[string][]string{"X-Test-Header": {"preserved"}}, ClientIP: "192.0.2.1",
	})
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d", response.StatusCode)
	}
	if path != "/base/asset" || query != "id=7" || header != "preserved" {
		t.Fatalf("origin request path=%q query=%q header=%q", path, query, header)
	}
}
