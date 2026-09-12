package cdn

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

var ErrSnapshotNotFound = errors.New("CDN snapshot not found")

type Source interface {
	Fetch(context.Context) ([]CDN, error)
}

type Store interface {
	Replace([]CDN)
	FindByDomain(string) (CDN, bool)
}

type SnapshotStore interface {
	Load(context.Context) ([]CDN, error)
	Save(context.Context, []CDN) error
}

type Service struct {
	source    Source
	store     Store
	snapshots SnapshotStore
	interval  time.Duration
	mu        sync.Mutex
	ready     atomic.Bool
}

func NewService(source Source, store Store, snapshots SnapshotStore, interval time.Duration) *Service {
	return &Service{source: source, store: store, snapshots: snapshots, interval: interval}
}

func (s *Service) LoadLocal(ctx context.Context) error {
	items, err := s.snapshots.Load(ctx)
	if errors.Is(err, ErrSnapshotNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("load local CDN snapshot: %w", err)
	}
	s.store.Replace(items)
	s.ready.Store(true)
	return nil
}

func (s *Service) Sync(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	items, err := s.source.Fetch(ctx)
	if err != nil {
		return fmt.Errorf("fetch CDN snapshot: %w", err)
	}
	if err := s.snapshots.Save(ctx, items); err != nil {
		return fmt.Errorf("persist CDN snapshot: %w", err)
	}
	s.store.Replace(items)
	s.ready.Store(true)
	return nil
}

// Ready reports whether a complete snapshot has been loaded into the in-memory
// store. A successfully loaded empty snapshot is ready; a missing snapshot is not.
func (s *Service) Ready() bool { return s.ready.Load() }

func (s *Service) RunPeriodic(ctx context.Context, onError func(error)) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			syncCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			err := s.Sync(syncCtx)
			cancel()
			if err != nil {
				onError(err)
			}
		case <-ctx.Done():
			return
		}
	}
}
