package ristrettostore

import (
	"testing"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/cache"
)

type fakeStorageMetrics struct{}

func (fakeStorageMetrics) SetCacheStorage(int64, int) {}

func TestSetPersistsAndReturnsResponse(t *testing.T) {
	store, err := New(t.TempDir(), time.Minute, fakeStorageMetrics{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer store.Close()

	want := cache.Entry{
		StatusCode: 201,
		Header:     map[string][]string{"Content-Type": {"image/png"}},
		Body:       []byte("image"),
		ExpiresAt:  time.Now().Add(time.Minute),
	}
	if err := store.Set("cdn.example/asset", want); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	store.cache.Wait()
	got, ok := store.Get("cdn.example/asset")
	if !ok {
		t.Fatal("Get() found = false")
	}
	if got.StatusCode != want.StatusCode || string(got.Body) != string(want.Body) {
		t.Fatalf("Get() = %#v", got)
	}
}
