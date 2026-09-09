package cdn

import (
	"context"
	"errors"
	"testing"
)

type storeFake struct {
	found     *CDN
	findErr   error
	createErr error
	updateErr error
	deleteErr error
	created   *CDN
}

func (f *storeFake) Create(_ context.Context, item *CDN) error {
	f.created = item
	return f.createErr
}

func (f *storeFake) List(context.Context) ([]*CDN, error)          { return nil, nil }
func (f *storeFake) Get(context.Context, string) (*CDN, error)    { return nil, nil }
func (f *storeFake) Update(context.Context, string, *CDN) error   { return f.updateErr }
func (f *storeFake) Delete(context.Context, string) error         { return f.deleteErr }
func (f *storeFake) FindByOrigin(context.Context, string) (*CDN, error) {
	return f.found, f.findErr
}

type notifierFake struct {
	calls int
	err   error
}

func (f *notifierFake) NotifyRefresh(context.Context) error {
	f.calls++
	return f.err
}

func TestCreateStoresCDNAndNotifiesEdges(t *testing.T) {
	store := &storeFake{findErr: errors.New("not found")}
	notifier := &notifierFake{}
	service := NewService(store, notifier)

	err := service.Create(context.Background(), "https://origin.example", "cdn.example", true, 60)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if store.created == nil {
		t.Fatal("Create() did not persist the CDN")
	}
	if store.created.CacheTTL != 60 || !store.created.IsActive {
		t.Fatalf("persisted CDN = %#v", store.created)
	}
	if notifier.calls != 1 {
		t.Fatalf("notification calls = %d, want 1", notifier.calls)
	}
}

func TestCreateDoesNotNotifyWhenStoreFails(t *testing.T) {
	wantErr := errors.New("insert failed")
	store := &storeFake{findErr: errors.New("not found"), createErr: wantErr}
	notifier := &notifierFake{}
	service := NewService(store, notifier)

	err := service.Create(context.Background(), "https://origin.example", "cdn.example", true, 60)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Create() error = %v, want %v", err, wantErr)
	}
	if notifier.calls != 0 {
		t.Fatalf("notification calls = %d, want 0", notifier.calls)
	}
}

func TestCreateReportsExistingCDN(t *testing.T) {
	store := &storeFake{found: &CDN{Origin: "https://origin.example"}}
	service := NewService(store, &notifierFake{})

	err := service.Create(context.Background(), "https://origin.example", "cdn.example", true, 60)
	if !errors.Is(err, ErrExists) {
		t.Fatalf("Create() error = %v, want %v", err, ErrExists)
	}
}

func TestMutationRemainsSuccessfulWhenNotificationFails(t *testing.T) {
	store := &storeFake{findErr: errors.New("not found")}
	notifier := &notifierFake{err: errors.New("NATS unavailable")}
	service := NewService(store, notifier)

	if err := service.Create(context.Background(), "https://origin.example", "cdn.example", true, 60); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
}

func TestManualRefreshReportsNotificationFailure(t *testing.T) {
	wantErr := errors.New("NATS unavailable")
	service := NewService(&storeFake{}, &notifierFake{err: wantErr})

	if err := service.Refresh(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("Refresh() error = %v, want %v", err, wantErr)
	}
}

func TestUpdateAndDeleteNotifyAfterSuccessfulPersistence(t *testing.T) {
	store := &storeFake{}
	notifier := &notifierFake{}
	service := NewService(store, notifier)

	if err := service.Update(context.Background(), "cdn-id", "https://origin.example", "cdn.example", true, 60); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if err := service.Delete(context.Background(), "cdn-id"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if notifier.calls != 2 {
		t.Fatalf("notification calls = %d, want 2", notifier.calls)
	}
}

func TestFailedUpdateAndDeleteDoNotNotify(t *testing.T) {
	store := &storeFake{
		updateErr: errors.New("update failed"),
		deleteErr: errors.New("delete failed"),
	}
	notifier := &notifierFake{}
	service := NewService(store, notifier)

	_ = service.Update(context.Background(), "cdn-id", "https://origin.example", "cdn.example", true, 60)
	_ = service.Delete(context.Background(), "cdn-id")
	if notifier.calls != 0 {
		t.Fatalf("notification calls = %d, want 0", notifier.calls)
	}
}
