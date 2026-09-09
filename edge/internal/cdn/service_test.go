package cdn

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeSource struct {
	items []CDN
	err   error
}

func (f *fakeSource) Fetch(context.Context) ([]CDN, error) { return f.items, f.err }

type fakeStore struct {
	items []CDN
}

func (f *fakeStore) Replace(items []CDN) { f.items = append([]CDN(nil), items...) }
func (f *fakeStore) FindByDomain(domain string) (CDN, bool) {
	for _, item := range f.items {
		if item.Domain() == NormalizeDomain(domain) && item.IsActive() {
			return item, true
		}
	}
	return CDN{}, false
}

type fakeSnapshotStore struct {
	items   []CDN
	loadErr error
	saveErr error
}

func (f *fakeSnapshotStore) Load(context.Context) ([]CDN, error) { return f.items, f.loadErr }
func (f *fakeSnapshotStore) Save(_ context.Context, items []CDN) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.items = append([]CDN(nil), items...)
	return nil
}

func TestSyncPersistsBeforeReplacingInMemorySnapshot(t *testing.T) {
	oldItem := mustCDN(t, "old.example", "http://old-origin.example")
	newItem := mustCDN(t, "new.example", "http://new-origin.example")
	store := &fakeStore{items: []CDN{oldItem}}
	snapshots := &fakeSnapshotStore{saveErr: errors.New("disk unavailable")}
	service := NewService(&fakeSource{items: []CDN{newItem}}, store, snapshots, time.Minute)

	if err := service.Sync(context.Background()); err == nil {
		t.Fatal("Sync() error = nil, want persistence error")
	}
	if _, ok := store.FindByDomain(oldItem.Domain()); !ok {
		t.Fatal("last-known-good in-memory snapshot was replaced")
	}
	if _, ok := store.FindByDomain(newItem.Domain()); ok {
		t.Fatal("unpersisted snapshot was published to readers")
	}
}

func TestLoadLocalAllowsMissingSnapshot(t *testing.T) {
	service := NewService(&fakeSource{}, &fakeStore{}, &fakeSnapshotStore{loadErr: ErrSnapshotNotFound}, time.Minute)
	if err := service.LoadLocal(context.Background()); err != nil {
		t.Fatalf("LoadLocal() error = %v", err)
	}
}

func mustCDN(t *testing.T, domain, origin string) CDN {
	t.Helper()
	item, err := New("id", domain, origin, true, 60)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return item
}
