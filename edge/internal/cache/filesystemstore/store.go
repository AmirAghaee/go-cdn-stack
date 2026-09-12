package filesystemstore

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/cache"
	"github.com/dgraph-io/ristretto/v2"
)

type StorageMetrics interface {
	SetCacheStorage(sizeBytes int64, itemCount int)
}

type Store struct {
	cache           *ristretto.Cache[string, metadata]
	directory       string
	cleanerInterval time.Duration
	metrics         StorageMetrics
}

type metadata struct {
	Key                   string              `json:"key,omitempty"`
	FilePath              string              `json:"file_path"`
	Header                map[string][]string `json:"header"`
	ExpiresAt             time.Time           `json:"expires_at"`
	StoredAt              time.Time           `json:"stored_at"`
	InitialAgeNanoseconds int64               `json:"initial_age_nanoseconds"`
	StatusCode            int                 `json:"status_code,omitempty"`
}

func New(directory string, cleanerInterval time.Duration, metrics StorageMetrics) (*Store, error) {
	if err := os.MkdirAll(directory, 0755); err != nil {
		return nil, fmt.Errorf("create cache directory: %w", err)
	}
	store := &Store{directory: directory, cleanerInterval: cleanerInterval, metrics: metrics}
	memory, err := ristretto.NewCache(&ristretto.Config[string, metadata]{
		NumCounters: 1e7,
		MaxCost:     1 << 28,
		BufferItems: 64,
		OnEvict: func(_ *ristretto.Item[metadata]) {
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
	item := value
	if !time.Now().Before(item.ExpiresAt) {
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
		StoredAt:   item.StoredAt,
		InitialAge: time.Duration(item.InitialAgeNanoseconds),
	}, true
}

func (s *Store) Close() { s.cache.Close() }

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
