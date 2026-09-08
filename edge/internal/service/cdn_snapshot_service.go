package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/client"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/domain"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/repository"
)

type CdnSnapshotServiceInterface interface {
	LoadLocal() error
	ProcessSnapshot(context.Context) error
	StartPeriodic(<-chan struct{})
}

type cdnSnapshotService struct {
	client       client.ControlPanelClientInterface
	repository   repository.CdnRepositoryInterface
	snapshotFile string
	interval     time.Duration
	mu           sync.Mutex
}

func NewCdnSnapshotService(
	controlPanelClient client.ControlPanelClientInterface,
	repo repository.CdnRepositoryInterface,
	snapshotFile string,
	interval time.Duration,
) CdnSnapshotServiceInterface {
	return &cdnSnapshotService{
		client:       controlPanelClient,
		repository:   repo,
		snapshotFile: snapshotFile,
		interval:     interval,
	}
}

func (s *cdnSnapshotService) LoadLocal() error {
	data, err := os.ReadFile(s.snapshotFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read local CDN snapshot: %w", err)
	}

	var cdns []domain.CDN
	if err := json.Unmarshal(data, &cdns); err != nil {
		return fmt.Errorf("decode local CDN snapshot: %w", err)
	}
	s.repository.Set(cdns)
	log.Printf("loaded %d CDNs from local snapshot", len(cdns))
	return nil
}

func (s *cdnSnapshotService) ProcessSnapshot(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cdns, err := s.client.GetCDNs(ctx)
	if err != nil {
		return err
	}

	// Publish the valid snapshot to memory first. A disk failure must not keep
	// an edge on stale configuration while it is still running.
	s.repository.Set(cdns)
	if err := s.persist(cdns); err != nil {
		return err
	}

	log.Printf("synchronized %d CDNs from control panel", len(cdns))
	return nil
}

func (s *cdnSnapshotService) StartPeriodic(stop <-chan struct{}) {
	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				err := s.ProcessSnapshot(ctx)
				cancel()
				if err != nil {
					log.Printf("periodic CDN synchronization failed: %v", err)
				}
			case <-stop:
				return
			}
		}
	}()
}

func (s *cdnSnapshotService) persist(cdns []domain.CDN) error {
	data, err := json.Marshal(cdns)
	if err != nil {
		return fmt.Errorf("encode CDN snapshot: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.snapshotFile), 0755); err != nil {
		return fmt.Errorf("create CDN snapshot directory: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(s.snapshotFile), ".cdns-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary CDN snapshot: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write CDN snapshot: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close CDN snapshot: %w", err)
	}
	if err := os.Rename(tmpName, s.snapshotFile); err != nil {
		return fmt.Errorf("replace CDN snapshot: %w", err)
	}
	return nil
}
