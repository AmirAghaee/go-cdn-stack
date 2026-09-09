package filesystemstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/cdn"
)

type Store struct {
	path string
}

type snapshotDTO struct {
	ID       string `json:"id"`
	Domain   string `json:"domain"`
	Origin   string `json:"origin"`
	IsActive bool   `json:"is_active"`
	CacheTTL uint   `json:"cache_ttl"`
}

func New(path string) *Store { return &Store{path: path} }

func (s *Store) Load(ctx context.Context) ([]cdn.CDN, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, cdn.ErrSnapshotNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read snapshot: %w", err)
	}

	var payload []snapshotDTO
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("decode snapshot: %w", err)
	}
	items := make([]cdn.CDN, 0, len(payload))
	for _, dto := range payload {
		item, err := cdn.New(dto.ID, dto.Domain, dto.Origin, dto.IsActive, dto.CacheTTL)
		if err != nil {
			return nil, fmt.Errorf("validate CDN %q: %w", dto.Domain, err)
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Store) Save(ctx context.Context, items []cdn.CDN) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	payload := make([]snapshotDTO, 0, len(items))
	for _, item := range items {
		payload = append(payload, snapshotDTO{
			ID: item.ID(), Domain: item.Domain(), Origin: item.Origin(),
			IsActive: item.IsActive(), CacheTTL: item.CacheTTL(),
		})
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode snapshot: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return fmt.Errorf("create snapshot directory: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".cdns-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary snapshot: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write snapshot: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close snapshot: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("replace snapshot: %w", err)
	}
	return nil
}
