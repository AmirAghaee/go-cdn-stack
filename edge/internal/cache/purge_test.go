package cache

import (
	"context"
	"errors"
	"testing"
)

type recordingPurgeStore struct {
	domain string
	result PurgeResult
	err    error
}

func (s *recordingPurgeStore) PurgeDomain(_ context.Context, domain string) (PurgeResult, error) {
	s.domain = domain
	return s.result, s.err
}

func TestPurgeServiceNormalizesDomain(t *testing.T) {
	store := &recordingPurgeStore{result: PurgeResult{EntriesPurged: 2, BytesFreed: 7}}
	result, err := NewPurgeService(store).PurgeDomain(context.Background(), " CDN.Example.:443 ")
	if err != nil {
		t.Fatalf("PurgeDomain() error = %v", err)
	}
	if store.domain != "cdn.example" || result.Domain != "cdn.example" || result.EntriesPurged != 2 || result.BytesFreed != 7 {
		t.Fatalf("PurgeDomain() store domain = %q, result = %+v", store.domain, result)
	}
}

func TestPurgeServiceRejectsEmptyDomain(t *testing.T) {
	store := &recordingPurgeStore{}
	_, err := NewPurgeService(store).PurgeDomain(context.Background(), " . ")
	if err == nil {
		t.Fatal("PurgeDomain() error = nil, want validation error")
	}
	if store.domain != "" {
		t.Fatalf("store called with domain %q", store.domain)
	}
}

func TestPurgeServiceReturnsPartialResultAndError(t *testing.T) {
	wantErr := errors.New("partial failure")
	store := &recordingPurgeStore{result: PurgeResult{EntriesPurged: 1}, err: wantErr}
	result, err := NewPurgeService(store).PurgeDomain(context.Background(), "cdn.example")
	if !errors.Is(err, wantErr) || result.Domain != "cdn.example" || result.EntriesPurged != 1 {
		t.Fatalf("PurgeDomain() = %+v, %v", result, err)
	}
}
