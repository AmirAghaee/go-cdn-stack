package service

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/domain"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/repository"
)

type fakeControlPanelClient struct {
	cdns []domain.CDN
	err  error
}

func (f *fakeControlPanelClient) GetCDNs(context.Context) ([]domain.CDN, error) {
	return f.cdns, f.err
}

func TestSnapshotPersistsAndLoadsLastKnownGoodConfiguration(t *testing.T) {
	snapshotFile := filepath.Join(t.TempDir(), "config", "cdns.json")
	cdn := domain.CDN{Domain: "cdn.example", Origin: "http://origin.example", IsActive: true, CacheTTL: 60}

	firstRepo := repository.NewCdnRepository()
	first := NewCdnSnapshotService(
		&fakeControlPanelClient{cdns: []domain.CDN{cdn}},
		firstRepo,
		snapshotFile,
		time.Minute,
	)
	if err := first.ProcessSnapshot(context.Background()); err != nil {
		t.Fatalf("process snapshot: %v", err)
	}

	secondRepo := repository.NewCdnRepository()
	second := NewCdnSnapshotService(
		&fakeControlPanelClient{},
		secondRepo,
		snapshotFile,
		time.Minute,
	)
	if err := second.LoadLocal(); err != nil {
		t.Fatalf("load local snapshot: %v", err)
	}
	got, ok := secondRepo.GetByDomain(cdn.Domain)
	if !ok || got.Origin != cdn.Origin {
		t.Fatalf("loaded CDN = %#v, found = %v", got, ok)
	}
}
