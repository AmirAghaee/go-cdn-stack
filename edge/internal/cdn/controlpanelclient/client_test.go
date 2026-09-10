package controlpanelclient

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestFetchUsesScopedSnapshotCredential(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/edge/v1/snapshot" {
			t.Errorf("path = %s, want /edge/v1/snapshot", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer edge-secret" {
			t.Errorf("Authorization = %q, want scoped service credential", got)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`[{"id":"1","domain":"example.com","origin":"https://origin.example.com","is_active":true,"cache_ttl":60}]`)),
		}, nil
	})}

	items, err := New("https://control-panel.example", "edge-secret", httpClient).Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if len(items) != 1 || items[0].Domain() != "example.com" {
		t.Fatalf("Fetch() items = %#v, want example.com", items)
	}
}
