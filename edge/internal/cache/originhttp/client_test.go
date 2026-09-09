package originhttp

import (
	"context"
	"io"
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
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatalf("close response body: %v", err)
	}
	if string(body) != "created" {
		t.Fatalf("response body = %q", body)
	}
	if path != "/base/asset" || query != "id=7" || header != "preserved" {
		t.Fatalf("origin request path=%q query=%q header=%q", path, query, header)
	}
}

func TestFetchDoesNotWaitForCompleteOriginBody(t *testing.T) {
	firstChunkWritten := make(chan struct{})
	releaseOrigin := make(chan struct{})
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		flusher := w.(http.Flusher)
		_, _ = w.Write([]byte("first"))
		flusher.Flush()
		close(firstChunkWritten)
		<-releaseOrigin
		_, _ = w.Write([]byte("second"))
	}))
	defer origin.Close()

	client := New(&http.Client{Timeout: time.Second})
	response, err := client.Fetch(context.Background(), cache.OriginRequest{
		Method: http.MethodGet, Origin: origin.URL, URI: "/asset", Host: "cdn.example",
	})
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	defer response.Body.Close()
	select {
	case <-firstChunkWritten:
	default:
		t.Fatal("Fetch returned before origin headers and first chunk were available")
	}
	close(releaseOrigin)
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if string(body) != "firstsecond" {
		t.Fatalf("response body = %q", body)
	}
}
