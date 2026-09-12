package filesystemstore

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/cache"
)

type fakeStorageMetrics struct{}

func (fakeStorageMetrics) SetCacheStorage(int64, int) {}

type recordingStorageMetrics struct {
	mu        sync.Mutex
	sizeBytes int64
	itemCount int
}

func (m *recordingStorageMetrics) SetCacheStorage(sizeBytes int64, itemCount int) {
	m.mu.Lock()
	m.sizeBytes = sizeBytes
	m.itemCount = itemCount
	m.mu.Unlock()
}

func (m *recordingStorageMetrics) current() (int64, int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sizeBytes, m.itemCount
}

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
	if !persisted.StoredAt.Equal(want.StoredAt) || persisted.InitialAgeNanoseconds != int64(want.InitialAge) ||
		persisted.BodySize != int64(len("image")) || persisted.FormatVersion != metadataFormatVersion {
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

func TestGetInvalidatesBodyTruncatedAfterLookup(t *testing.T) {
	metrics := &recordingStorageMetrics{}
	store, err := New(t.TempDir(), time.Minute, metrics)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer store.Close()

	writeEntry(t, store, "cdn.example/truncated", "complete")
	item, ok := store.cache.Get("cdn.example/truncated")
	if !ok {
		t.Fatal("committed metadata not found")
	}
	entry, ok := store.Get("cdn.example/truncated")
	if !ok {
		t.Fatal("Get() found = false")
	}
	if err := os.Truncate(item.FilePath, 3); err != nil {
		t.Fatalf("Truncate() error = %v", err)
	}
	if _, err := io.ReadAll(entry.Body); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("ReadAll() error = %v, want %v", err, io.ErrUnexpectedEOF)
	}
	if err := entry.Body.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if _, found := store.Get("cdn.example/truncated"); found {
		t.Fatal("corrupt cache entry remained available")
	}
	if _, err := os.Stat(item.FilePath); !os.IsNotExist(err) {
		t.Fatalf("body stat error = %v, want not exist", err)
	}
	if _, err := os.Stat(store.metadataPath("cdn.example/truncated")); !os.IsNotExist(err) {
		t.Fatalf("metadata stat error = %v, want not exist", err)
	}
	if size, count := metrics.current(); size != 0 || count != 0 {
		t.Fatalf("storage metrics = (%d, %d), want (0, 0)", size, count)
	}
}

func TestLoadReconcilesCacheOwnedArtifacts(t *testing.T) {
	directory := t.TempDir()
	first, err := New(directory, time.Minute, fakeStorageMetrics{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	writeEntry(t, first, "cdn.example/valid", "valid")
	first.Close()

	malformedMetadata := filepath.Join(directory, strings.Repeat("d", 64)+".cache.json")
	bodyTemp := filepath.Join(directory, strings.Repeat("a", 64)+"-abandoned.tmp")
	metadataTemp := filepath.Join(directory, ".cache-abandoned.tmp")
	orphanBody := filepath.Join(directory, strings.Repeat("b", 64)+"-orphan.cache")
	snapshotTemp := filepath.Join(directory, ".cdns-keep.tmp")
	unrelatedBody := filepath.Join(directory, "notes.cache")
	for path, contents := range map[string]string{
		malformedMetadata: "{", bodyTemp: "partial", metadataTemp: "partial",
		orphanBody: "orphan", snapshotTemp: "snapshot", unrelatedBody: "unrelated",
	} {
		if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", path, err)
		}
	}

	legacyKey := "cdn.example/legacy"
	legacyDigest := sha256.Sum256([]byte(legacyKey))
	legacyBody := filepath.Join(directory, fmt.Sprintf("%x-legacy.cache", legacyDigest))
	if err := os.WriteFile(legacyBody, []byte("legacy"), 0600); err != nil {
		t.Fatalf("write legacy body: %v", err)
	}
	legacy := metadata{
		Key: legacyKey, FilePath: legacyBody, BodySize: int64(len("legacy")),
		StoredAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute),
	}
	legacyData, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	legacyMetadata := filepath.Join(directory, fmt.Sprintf("%x.cache.json", legacyDigest))
	if err := os.WriteFile(legacyMetadata, legacyData, 0600); err != nil {
		t.Fatalf("write legacy metadata: %v", err)
	}

	metrics := &recordingStorageMetrics{}
	recovered, err := New(directory, time.Minute, metrics)
	if err != nil {
		t.Fatalf("New() recovery error = %v", err)
	}
	defer recovered.Close()
	if err := recovered.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	entry, ok := recovered.Get("cdn.example/valid")
	if !ok {
		t.Fatal("valid entry was not recovered")
	}
	body, err := io.ReadAll(entry.Body)
	if err != nil {
		t.Fatalf("read recovered body: %v", err)
	}
	_ = entry.Body.Close()
	if string(body) != "valid" {
		t.Fatalf("recovered body = %q, want valid", body)
	}
	for _, path := range []string{malformedMetadata, bodyTemp, metadataTemp, orphanBody, legacyBody, legacyMetadata} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("artifact %q stat error = %v, want not exist", path, err)
		}
	}
	if _, err := os.Stat(snapshotTemp); err != nil {
		t.Fatalf("unrelated snapshot temporary file was removed: %v", err)
	}
	if _, err := os.Stat(unrelatedBody); err != nil {
		t.Fatalf("unrelated cache-suffixed file was removed: %v", err)
	}
	if size, count := metrics.current(); size != int64(len("valid")) || count != 1 {
		t.Fatalf("storage metrics = (%d, %d), want (%d, 1)", size, count, len("valid"))
	}
}

func TestCleanerPreservesActiveWrite(t *testing.T) {
	store, err := New(t.TempDir(), time.Minute, fakeStorageMetrics{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer store.Close()

	pending, err := store.Begin("cdn.example/active", cache.EntryMetadata{
		ExpiresAt: time.Now().Add(time.Minute), StoredAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}
	concrete := pending.(*pendingEntry)
	if _, err := pending.Write([]byte("partial")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	store.cleanExpiredFiles()
	if _, err := os.Stat(concrete.tempPath); err != nil {
		t.Fatalf("active temporary body was removed: %v", err)
	}
	if err := pending.Abort(); err != nil {
		t.Fatalf("Abort() error = %v", err)
	}
}

func TestReplacementUpdatesAccountingWithoutDirectoryScan(t *testing.T) {
	directory := t.TempDir()
	metrics := &recordingStorageMetrics{}
	store, err := New(directory, time.Minute, metrics)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer store.Close()

	writeEntry(t, store, "cdn.example/replaced", "first")
	old, ok := store.cache.Get("cdn.example/replaced")
	if !ok {
		t.Fatal("first metadata not found")
	}
	orphan := filepath.Join(directory, strings.Repeat("c", 64)+"-orphan.cache")
	if err := os.WriteFile(orphan, []byte("unaccounted"), 0600); err != nil {
		t.Fatalf("write orphan: %v", err)
	}
	writeEntry(t, store, "cdn.example/replaced", "replacement")
	if _, err := os.Stat(old.FilePath); !os.IsNotExist(err) {
		t.Fatalf("replaced body stat error = %v, want not exist", err)
	}
	if _, err := os.Stat(orphan); err != nil {
		t.Fatalf("commit unexpectedly scanned the directory: %v", err)
	}
	if size, count := metrics.current(); size != int64(len("replacement")) || count != 1 {
		t.Fatalf("storage metrics = (%d, %d), want (%d, 1)", size, count, len("replacement"))
	}
	store.cleanExpiredFiles()
	if _, err := os.Stat(orphan); !os.IsNotExist(err) {
		t.Fatalf("orphan stat error = %v, want not exist", err)
	}
}

func writeEntry(t *testing.T, store *Store, key, body string) {
	t.Helper()
	pending, err := store.Begin(key, cache.EntryMetadata{
		StatusCode: 200, ExpiresAt: time.Now().Add(time.Minute), StoredAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}
	if _, err := pending.Write([]byte(body)); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := pending.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	store.cache.Wait()
}
