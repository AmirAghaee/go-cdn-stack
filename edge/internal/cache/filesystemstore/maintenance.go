package filesystemstore

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *Store) Load() error {
	files, err := os.ReadDir(s.directory)
	if err != nil {
		return fmt.Errorf("read cache directory: %w", err)
	}
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".cache.json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.directory, file.Name()))
		if err != nil {
			continue
		}
		var item metadata
		if err := json.Unmarshal(data, &item); err != nil || item.StoredAt.IsZero() ||
			item.InitialAgeNanoseconds < 0 || !time.Now().Before(item.ExpiresAt) {
			continue
		}
		key := item.Key
		if key == "" {
			var ok bool
			key, ok = keyFromMetadataName(file.Name())
			if !ok {
				continue
			}
		}
		s.cache.SetWithTTL(key, item, 1, time.Until(item.ExpiresAt))
	}
	s.updateMetrics()
	return nil
}

func (s *Store) RunCleaner(ctx context.Context) {
	ticker := time.NewTicker(s.cleanerInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.cleanExpiredFiles()
		case <-ctx.Done():
			return
		}
	}
}

func (s *Store) cleanExpiredFiles() {
	files, err := os.ReadDir(s.directory)
	if err != nil {
		return
	}
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".cache.json") {
			continue
		}
		metaPath := filepath.Join(s.directory, file.Name())
		data, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		var item metadata
		if json.Unmarshal(data, &item) == nil && time.Now().After(item.ExpiresAt) {
			_ = os.Remove(item.FilePath)
			_ = os.Remove(metaPath)
		}
	}
	s.updateMetrics()
}

func (s *Store) updateMetrics() {
	files, err := os.ReadDir(s.directory)
	if err != nil {
		return
	}
	var size int64
	var count int
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".cache") {
			continue
		}
		if info, err := file.Info(); err == nil {
			size += info.Size()
			count++
		}
	}
	s.metrics.SetCacheStorage(size, count)
}

func keyFromMetadataName(name string) (string, bool) {
	hexName := strings.TrimSuffix(name, ".cache.json")
	decoded, err := hex.DecodeString(hexName)
	if err != nil {
		return "", false
	}
	return string(decoded), true
}
