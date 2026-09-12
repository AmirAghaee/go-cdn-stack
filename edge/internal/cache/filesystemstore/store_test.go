package filesystemstore

import (
	"encoding/json"
	"io"
	"os"
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

	want := cache.EntryMetadata{
		StatusCode: 201,
		Header:     map[string][]string{"Content-Type": {"image/png"}},
		ExpiresAt:  time.Now().Add(time.Minute),
		StoredAt:   time.Now().Add(-20 * time.Second),
		InitialAge: 10 * time.Second,
	}
	pending, err := store.Begin("cdn.example/asset", want)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}
	if _, err := pending.Write([]byte("image")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if _, found := store.Get("cdn.example/asset"); found {
		t.Fatal("cache entry became visible before commit")
	}
	if err := pending.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	store.cache.Wait()
	got, ok := store.Get("cdn.example/asset")
	if !ok {
		t.Fatal("Get() found = false")
	}
	if _, ok := got.Body.(*os.File); !ok {
		t.Fatalf("cached body type = %T, want *os.File", got.Body)
	}
	body, err := io.ReadAll(got.Body)
	if err != nil {
		t.Fatalf("read cached body: %v", err)
	}
	if err := got.Body.Close(); err != nil {
		t.Fatalf("close cached body: %v", err)
	}
	if got.StatusCode != want.StatusCode || string(body) != "image" ||
		!got.StoredAt.Equal(want.StoredAt) || got.InitialAge != want.InitialAge {
		t.Fatalf("Get() entry=%+v body=%q", got, body)
	}
	data, err := os.ReadFile(store.metadataPath("cdn.example/asset"))
	if err != nil {
		t.Fatalf("read persisted metadata: %v", err)
	}
	var persisted metadata
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatalf("decode persisted metadata: %v", err)
	}
	if !persisted.StoredAt.Equal(want.StoredAt) || persisted.InitialAgeNanoseconds != int64(want.InitialAge) {
		t.Fatalf("persisted timing metadata = %+v", persisted)
	}
}

func TestAbortRemovesTemporaryBody(t *testing.T) {
	directory := t.TempDir()
	store, err := New(directory, time.Minute, fakeStorageMetrics{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer store.Close()

	pending, err := store.Begin("cdn.example/asset", cache.EntryMetadata{
		ExpiresAt: time.Now().Add(time.Minute), StoredAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}
	if _, err := pending.Write([]byte("partial")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := pending.Abort(); err != nil {
		t.Fatalf("Abort() error = %v", err)
	}
	files, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("temporary files remain: %v", files)
	}
}
