package filesystemstore

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *Store) Load() error {
	if err := s.reconcile(); err != nil {
		return err
	}
	s.updateMetrics()
	return nil
}

func (s *Store) reconcile() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	files, err := os.ReadDir(s.directory)
	if err != nil {
		return fmt.Errorf("read cache directory: %w", err)
	}
	now := time.Now()
	referencedBodies := make(map[string]int64)
	for _, file := range files {
		if file.IsDir() || !isCacheMetadata(file.Name()) {
			continue
		}
		metadataPath := filepath.Join(s.directory, file.Name())
		data, err := os.ReadFile(metadataPath)
		if err != nil {
			_ = os.Remove(metadataPath)
			continue
		}
		var item metadata
		if err := json.Unmarshal(data, &item); err != nil ||
			filepath.Clean(metadataPath) != s.metadataPath(item.Key) ||
			!s.validMetadata(item.Key, item) || !now.Before(item.ExpiresAt) {
			if item.Key != "" {
				s.cache.Del(item.Key)
			}
			_ = os.Remove(metadataPath)
			if s.bodyPathMatchesKey(item.Key, item.FilePath) {
				_ = os.Remove(item.FilePath)
			}
			continue
		}
		info, err := os.Lstat(item.FilePath)
		if err != nil || !info.Mode().IsRegular() || info.Size() != item.BodySize {
			s.cache.Del(item.Key)
			_ = os.Remove(metadataPath)
			_ = os.Remove(item.FilePath)
			continue
		}
		referencedBodies[item.FilePath] = item.BodySize
		s.cache.SetWithTTL(item.Key, item, 1, time.Until(item.ExpiresAt))
	}
	s.cache.Wait()

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		path := filepath.Join(s.directory, file.Name())
		switch {
		case isCacheTemp(file.Name()):
			if _, active := s.activeTemps[path]; !active {
				_ = os.Remove(path)
			}
		case isCacheBody(file.Name()):
			if _, referenced := referencedBodies[path]; !referenced {
				if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
					if info, infoErr := file.Info(); infoErr == nil {
						referencedBodies[path] = info.Size()
					}
				}
			}
		}
	}

	s.accountedBodies = referencedBodies
	s.sizeBytes = 0
	for _, size := range referencedBodies {
		s.sizeBytes += size
	}
	s.itemCount = len(referencedBodies)
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
	if err := s.reconcile(); err == nil {
		s.updateMetrics()
	}
}

func isCacheTemp(name string) bool {
	if strings.HasPrefix(name, ".cache-") && strings.HasSuffix(name, ".tmp") {
		return true
	}
	return hasCacheBodyPrefix(name) && strings.HasSuffix(name, ".tmp")
}

func isCacheBody(name string) bool {
	return hasCacheBodyPrefix(name) && strings.HasSuffix(name, ".cache")
}

func isCacheMetadata(name string) bool {
	const suffix = ".cache.json"
	return len(name) == 64+len(suffix) && strings.HasSuffix(name, suffix) && isLowerHex(name[:64])
}

func hasCacheBodyPrefix(name string) bool {
	return len(name) > 65 && name[64] == '-' && isLowerHex(name[:64])
}

func isLowerHex(value string) bool {
	for _, character := range value {
		if !strings.ContainsRune("0123456789abcdef", character) {
			return false
		}
	}
	return value != ""
}
