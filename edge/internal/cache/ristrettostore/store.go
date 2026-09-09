package ristrettostore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/cache"
	"github.com/dgraph-io/ristretto"
)

type StorageMetrics interface {
	SetCacheStorage(sizeBytes int64, itemCount int)
}

type Store struct {
	cache           *ristretto.Cache
	directory       string
	cleanerInterval time.Duration
	metrics         StorageMetrics
}

type metadata struct {
	Key        string              `json:"key,omitempty"`
	FilePath   string              `json:"file_path"`
	Header     map[string][]string `json:"header"`
	ExpiresAt  time.Time           `json:"expires_at"`
	StatusCode int                 `json:"status_code,omitempty"`
}

func New(directory string, cleanerInterval time.Duration, metrics StorageMetrics) (*Store, error) {
	if err := os.MkdirAll(directory, 0755); err != nil {
		return nil, fmt.Errorf("create cache directory: %w", err)
	}
	store := &Store{directory: directory, cleanerInterval: cleanerInterval, metrics: metrics}
	memory, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,
		MaxCost:     1 << 28,
		BufferItems: 64,
		OnEvict: func(_ *ristretto.Item) {
			store.updateMetrics()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create memory cache: %w", err)
	}
	store.cache = memory
	return store, nil
}

func (s *Store) Get(key string) (cache.Entry, bool) {
	value, ok := s.cache.Get(key)
	if !ok {
		return cache.Entry{}, false
	}
	item, ok := value.(metadata)
	if !ok || !time.Now().Before(item.ExpiresAt) {
		s.delete(key, item)
		return cache.Entry{}, false
	}
	body, err := os.ReadFile(item.FilePath)
	if err != nil {
		s.delete(key, item)
		return cache.Entry{}, false
	}
	statusCode := item.StatusCode
	if statusCode == 0 {
		statusCode = 200
	}
	return cache.Entry{StatusCode: statusCode, Header: item.Header, Body: body, ExpiresAt: item.ExpiresAt}, true
}

func (s *Store) Set(key string, entry cache.Entry) error {
	if !time.Now().Before(entry.ExpiresAt) {
		return nil
	}
	digest := sha256.Sum256([]byte(key))
	cacheFile := filepath.Join(s.directory, fmt.Sprintf("%x.cache", digest))
	if err := writeAtomic(cacheFile, entry.Body, 0644); err != nil {
		return fmt.Errorf("write cache body: %w", err)
	}
	item := metadata{Key: key, FilePath: cacheFile, Header: entry.Header, ExpiresAt: entry.ExpiresAt, StatusCode: entry.StatusCode}
	data, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return fmt.Errorf("encode cache metadata: %w", err)
	}
	if err := writeAtomic(cacheFile+".json", data, 0644); err != nil {
		return fmt.Errorf("write cache metadata: %w", err)
	}
	s.cache.SetWithTTL(key, item, 1, time.Until(entry.ExpiresAt))
	s.updateMetrics()
	return nil
}

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
		if err := json.Unmarshal(data, &item); err != nil || !time.Now().Before(item.ExpiresAt) {
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

func (s *Store) Close() { s.cache.Close() }

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

func (s *Store) delete(key string, item metadata) {
	if item.FilePath != "" {
		_ = os.Remove(item.FilePath)
		_ = os.Remove(item.FilePath + ".json")
	}
	s.cache.Del(key)
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

func writeAtomic(path string, data []byte, mode os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".cache-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
