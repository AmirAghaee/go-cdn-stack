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

type pendingEntry struct {
	store     *Store
	key       string
	metadata  metadata
	file      *os.File
	tempPath  string
	finalPath string
	oldPath   string
	closed    bool
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
	body, err := os.Open(item.FilePath)
	if err != nil {
		s.delete(key, item)
		return cache.Entry{}, false
	}
	statusCode := item.StatusCode
	if statusCode == 0 {
		statusCode = 200
	}
	return cache.Entry{
		StatusCode: statusCode,
		Header:     item.Header,
		Body:       body,
		ExpiresAt:  item.ExpiresAt,
	}, true
}

func (s *Store) Begin(key string, entry cache.EntryMetadata) (cache.PendingEntry, error) {
	if !time.Now().Before(entry.ExpiresAt) {
		return nil, fmt.Errorf("cache entry is already expired")
	}
	digest := sha256.Sum256([]byte(key))
	prefix := fmt.Sprintf("%x-", digest)
	temp, err := os.CreateTemp(s.directory, prefix+"*.tmp")
	if err != nil {
		return nil, fmt.Errorf("create cache body: %w", err)
	}
	tempPath := temp.Name()
	var oldPath string
	if value, ok := s.cache.Get(key); ok {
		if old, ok := value.(metadata); ok {
			oldPath = old.FilePath
		}
	}
	return &pendingEntry{
		store: s,
		key:   key,
		metadata: metadata{
			Key: key, Header: entry.Header, ExpiresAt: entry.ExpiresAt,
			StatusCode: entry.StatusCode,
		},
		file: temp, tempPath: tempPath, oldPath: oldPath,
		finalPath: strings.TrimSuffix(tempPath, ".tmp") + ".cache",
	}, nil
}

func (p *pendingEntry) Write(data []byte) (int, error) {
	if p.closed {
		return 0, os.ErrClosed
	}
	return p.file.Write(data)
}

func (p *pendingEntry) Commit() error {
	if p.closed {
		return os.ErrClosed
	}
	p.closed = true
	if !time.Now().Before(p.metadata.ExpiresAt) {
		_ = p.file.Close()
		_ = os.Remove(p.tempPath)
		return nil
	}
	if err := p.file.Chmod(0644); err != nil {
		_ = p.file.Close()
		_ = os.Remove(p.tempPath)
		return fmt.Errorf("set cache body permissions: %w", err)
	}
	if err := p.file.Close(); err != nil {
		_ = os.Remove(p.tempPath)
		return fmt.Errorf("close cache body: %w", err)
	}
	if err := os.Rename(p.tempPath, p.finalPath); err != nil {
		_ = os.Remove(p.tempPath)
		return fmt.Errorf("publish cache body: %w", err)
	}

	p.metadata.FilePath = p.finalPath
	if accepted := p.store.cache.SetWithTTL(p.key, p.metadata, 1, time.Until(p.metadata.ExpiresAt)); !accepted {
		_ = os.Remove(p.finalPath)
		return fmt.Errorf("admit cache metadata")
	}
	p.store.cache.Wait()
	current, ok := p.store.cache.Get(p.key)
	currentMetadata, metadataOK := current.(metadata)
	if !ok || !metadataOK || currentMetadata.FilePath != p.finalPath {
		_ = os.Remove(p.finalPath)
		return fmt.Errorf("retain cache metadata")
	}

	data, err := json.MarshalIndent(p.metadata, "", "  ")
	if err != nil {
		p.store.cache.Del(p.key)
		p.store.cache.Wait()
		_ = os.Remove(p.finalPath)
		return fmt.Errorf("encode cache metadata: %w", err)
	}
	metaPath := p.store.metadataPath(p.key)
	if err := writeAtomic(metaPath, data, 0644); err != nil {
		p.store.cache.Del(p.key)
		p.store.cache.Wait()
		_ = os.Remove(p.finalPath)
		return fmt.Errorf("write cache metadata: %w", err)
	}

	if p.oldPath != "" && p.oldPath != p.finalPath {
		_ = os.Remove(p.oldPath)
	}
	p.store.updateMetrics()
	return nil
}

func (p *pendingEntry) Abort() error {
	if p.closed {
		return nil
	}
	p.closed = true
	closeErr := p.file.Close()
	removeErr := os.Remove(p.tempPath)
	if closeErr != nil {
		return fmt.Errorf("close temporary cache body: %w", closeErr)
	}
	if removeErr != nil && !os.IsNotExist(removeErr) {
		return fmt.Errorf("remove temporary cache body: %w", removeErr)
	}
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
	}
	_ = os.Remove(s.metadataPath(key))
	s.cache.Del(key)
	s.updateMetrics()
}

func (s *Store) metadataPath(key string) string {
	digest := sha256.Sum256([]byte(key))
	return filepath.Join(s.directory, fmt.Sprintf("%x.cache.json", digest))
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
