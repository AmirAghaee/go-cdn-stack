package filesystemstore

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPurgeDomainRemovesOnlyMatchingCommittedEntries(t *testing.T) {
	directory := t.TempDir()
	store, err := New(directory, time.Minute, fakeStorageMetrics{})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	targetKeys := []string{
		purgeTestKey("cdn.example", "/first", ""),
		purgeTestKey("cdn.example", "/second", "gzip"),
	}
	otherKey := purgeTestKey("other.example", "/first", "")
	for _, key := range append(targetKeys, otherKey) {
		writeEntry(t, store, key, "body")
	}

	malformed := filepath.Join(directory, strings.Repeat("a", 64)+".cache.json")
	snapshot := filepath.Join(directory, "cdns.json")
	temporary := filepath.Join(directory, ".cdn-snapshot.tmp")
	unrelated := filepath.Join(directory, "notes.cache")
	for path, contents := range map[string]string{
		malformed: "{", snapshot: "[]", temporary: "partial", unrelated: "keep",
	} {
		if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}

	result, err := mustPurger(t, directory).PurgeDomain(context.Background(), "cdn.example")
	if err != nil {
		t.Fatalf("PurgeDomain() error = %v", err)
	}
	if result.EntriesPurged != 2 || result.BytesFreed != 8 {
		t.Fatalf("PurgeDomain() result = %+v, want 2 entries and 8 bytes", result)
	}
	for _, key := range targetKeys {
		if _, found := store.Get(key); found {
			t.Fatalf("target key %q remains cached", key)
		}
		if _, err := os.Stat(store.metadataPath(key)); !os.IsNotExist(err) {
			t.Fatalf("target metadata stat error = %v, want not exist", err)
		}
	}
	if _, found := store.Get(otherKey); !found {
		t.Fatal("other domain entry was purged")
	}
	for _, path := range []string{malformed, snapshot, temporary, unrelated} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("preserved artifact %q stat error = %v", path, err)
		}
	}
}

func TestPurgeDomainTreatsMissingBodyAsInvalidatedEntry(t *testing.T) {
	directory := t.TempDir()
	store, err := New(directory, time.Minute, fakeStorageMetrics{})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	key := purgeTestKey("cdn.example", "/missing", "")
	writeEntry(t, store, key, "body")
	item, _ := store.cache.Get(key)
	if err := os.Remove(item.FilePath); err != nil {
		t.Fatal(err)
	}

	result, err := mustPurger(t, directory).PurgeDomain(context.Background(), "cdn.example")
	if err != nil || result.EntriesPurged != 1 || result.BytesFreed != 0 {
		t.Fatalf("PurgeDomain() = %+v, %v", result, err)
	}
}

func TestPurgeDomainZeroMatchesIsSuccessful(t *testing.T) {
	result, err := mustPurger(t, t.TempDir()).PurgeDomain(context.Background(), "cdn.example")
	if err != nil || result.EntriesPurged != 0 || result.BytesFreed != 0 {
		t.Fatalf("PurgeDomain() = %+v, %v", result, err)
	}
}

func TestPurgeDomainHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := mustPurger(t, t.TempDir()).PurgeDomain(ctx, "cdn.example")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("PurgeDomain() error = %v, want context.Canceled", err)
	}
}

func TestPurgeDomainReportsPartialRemovalFailure(t *testing.T) {
	directory := t.TempDir()
	key := purgeTestKey("cdn.example", "/blocked", "")
	digest := sha256.Sum256([]byte(key))
	bodyPath := filepath.Join(directory, fmt.Sprintf("%x-blocked.cache", digest))
	if err := os.Mkdir(bodyPath, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bodyPath, "child"), []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	item := metadata{
		Key: key, FilePath: bodyPath, Header: map[string][]string{},
		ExpiresAt: time.Now().Add(time.Hour), StoredAt: time.Now(), BodySize: 4,
		FormatVersion: metadataFormatVersion, StatusCode: 200,
	}
	data, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(metadataPath(directory, key), data, 0600); err != nil {
		t.Fatal(err)
	}

	result, err := mustPurger(t, directory).PurgeDomain(context.Background(), "cdn.example")
	if err == nil || result.EntriesPurged != 1 || result.BytesFreed != 0 {
		t.Fatalf("PurgeDomain() = %+v, %v; want partial failure", result, err)
	}
	if _, statErr := os.Stat(metadataPath(directory, key)); !os.IsNotExist(statErr) {
		t.Fatalf("metadata stat error = %v, want not exist", statErr)
	}
}

func mustPurger(t *testing.T, directory string) *Purger {
	t.Helper()
	purger, err := NewPurger(directory)
	if err != nil {
		t.Fatal(err)
	}
	return purger
}

func purgeTestKey(domain, uri, encoding string) string {
	return fmt.Sprintf(
		"v3|configuration=%s|host=%d:%s|uri=%d:%s|accept-encoding=%d:%s",
		strings.Repeat("0", 64), len(domain), domain, len(uri), uri, len(encoding), encoding,
	)
}
