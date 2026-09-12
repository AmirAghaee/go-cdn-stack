package filesystemstore

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/cache"
	"github.com/dgraph-io/ristretto/v2"
)

const metadataFormatVersion = 1

type StorageMetrics interface {
	SetCacheStorage(sizeBytes int64, itemCount int)
}

type Store struct {
	cache           *ristretto.Cache[string, metadata]
	directory       string
	cleanerInterval time.Duration
	metrics         StorageMetrics
	mu              sync.Mutex
	activeTemps     map[string]struct{}
	accountedBodies map[string]int64
	sizeBytes       int64
	itemCount       int
}

type metadata struct {
	Key                   string              `json:"key,omitempty"`
	FilePath              string              `json:"file_path"`
	Header                map[string][]string `json:"header"`
	ExpiresAt             time.Time           `json:"expires_at"`
	StoredAt              time.Time           `json:"stored_at"`
	InitialAgeNanoseconds int64               `json:"initial_age_nanoseconds"`
	BodySize              int64               `json:"body_size"`
	FormatVersion         int                 `json:"format_version"`
	StatusCode            int                 `json:"status_code,omitempty"`
}

func New(directory string, cleanerInterval time.Duration, metrics StorageMetrics) (*Store, error) {
	absoluteDirectory, err := filepath.Abs(directory)
	if err != nil {
		return nil, fmt.Errorf("resolve cache directory: %w", err)
	}
	if err := os.MkdirAll(absoluteDirectory, 0755); err != nil {
		return nil, fmt.Errorf("create cache directory: %w", err)
	}
	store := &Store{
		directory: absoluteDirectory, cleanerInterval: cleanerInterval, metrics: metrics,
		activeTemps: make(map[string]struct{}), accountedBodies: make(map[string]int64),
	}
	memory, err := ristretto.NewCache(&ristretto.Config[string, metadata]{
		NumCounters: 1e7,
		MaxCost:     1 << 28,
		BufferItems: 64,
	})
	if err != nil {
		return nil, fmt.Errorf("create memory cache: %w", err)
	}
	store.cache = memory
	return store, nil
}

func (s *Store) Get(key string) (cache.Entry, bool) {
	s.mu.Lock()
	value, ok := s.cache.Get(key)
	if !ok {
		s.mu.Unlock()
		return cache.Entry{}, false
	}
	item := value
	if !s.validMetadata(key, item) || !time.Now().Before(item.ExpiresAt) {
		s.deleteLocked(key, item)
		s.mu.Unlock()
		s.updateMetrics()
		return cache.Entry{}, false
	}
	info, err := os.Lstat(item.FilePath)
	if err != nil || !info.Mode().IsRegular() || info.Size() != item.BodySize {
		s.deleteLocked(key, item)
		s.mu.Unlock()
		s.updateMetrics()
		return cache.Entry{}, false
	}
	body, err := os.Open(item.FilePath)
	if err != nil {
		s.deleteLocked(key, item)
		s.mu.Unlock()
		s.updateMetrics()
		return cache.Entry{}, false
	}
	s.mu.Unlock()
	statusCode := item.StatusCode
	if statusCode == 0 {
		statusCode = 200
	}
	return cache.Entry{
		StatusCode: statusCode,
		Header:     item.Header,
		Body:       &validatedBody{body: body, remaining: item.BodySize, onCorrupt: func() { s.invalidate(key, item) }},
		ExpiresAt:  item.ExpiresAt,
		StoredAt:   item.StoredAt,
		InitialAge: time.Duration(item.InitialAgeNanoseconds),
	}, true
}

func (s *Store) Close() { s.cache.Close() }

func (s *Store) invalidate(key string, item metadata) {
	s.mu.Lock()
	current, ok := s.cache.Get(key)
	if ok && current.FilePath == item.FilePath {
		s.deleteLocked(key, item)
	}
	s.mu.Unlock()
	s.updateMetrics()
}

func (s *Store) deleteLocked(key string, item metadata) {
	if s.bodyPathMatchesKey(key, item.FilePath) {
		s.removeBodyLocked(item.FilePath)
	}
	_ = os.Remove(s.metadataPath(key))
	s.cache.Del(key)
}

func (s *Store) removeBodyLocked(path string) {
	if !s.safeBodyPath(path) {
		return
	}
	if err := os.Remove(path); err == nil || os.IsNotExist(err) {
		s.removeAccountingLocked(path)
	}
}

func (s *Store) metadataPath(key string) string {
	return metadataPath(s.directory, key)
}

func metadataPath(directory, key string) string {
	digest := sha256.Sum256([]byte(key))
	return filepath.Join(directory, fmt.Sprintf("%x.cache.json", digest))
}

func (s *Store) validMetadata(key string, item metadata) bool {
	return validMetadata(s.directory, key, item)
}

func validMetadata(directory, key string, item metadata) bool {
	if item.FormatVersion != metadataFormatVersion || item.StoredAt.IsZero() ||
		item.InitialAgeNanoseconds < 0 || item.BodySize < 0 || item.Key != key || !safeBodyPath(directory, item.FilePath) {
		return false
	}
	return bodyPathMatchesKey(directory, key, item.FilePath)
}

func (s *Store) bodyPathMatchesKey(key, path string) bool {
	return bodyPathMatchesKey(s.directory, key, path)
}

func bodyPathMatchesKey(directory, key, path string) bool {
	if !safeBodyPath(directory, path) {
		return false
	}
	digest := sha256.Sum256([]byte(key))
	return strings.HasPrefix(filepath.Base(path), fmt.Sprintf("%x-", digest))
}

func (s *Store) safeBodyPath(path string) bool {
	return safeBodyPath(s.directory, path)
}

func safeBodyPath(directory, path string) bool {
	cleaned := filepath.Clean(path)
	return filepath.Dir(cleaned) == directory && isCacheBody(filepath.Base(cleaned))
}

func (s *Store) addAccountingLocked(path string, size int64) {
	if previous, ok := s.accountedBodies[path]; ok {
		s.sizeBytes += size - previous
		s.accountedBodies[path] = size
		return
	}
	s.accountedBodies[path] = size
	s.sizeBytes += size
	s.itemCount++
}

func (s *Store) removeAccountingLocked(path string) {
	size, ok := s.accountedBodies[path]
	if !ok {
		return
	}
	delete(s.accountedBodies, path)
	s.sizeBytes -= size
	s.itemCount--
}

func (s *Store) updateMetrics() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metrics.SetCacheStorage(s.sizeBytes, s.itemCount)
}

type validatedBody struct {
	body      *os.File
	remaining int64
	onCorrupt func()
	once      sync.Once
}

func (b *validatedBody) Read(buffer []byte) (int, error) {
	count, err := b.body.Read(buffer)
	b.remaining -= int64(count)
	if b.remaining < 0 {
		b.once.Do(b.onCorrupt)
		return count, fmt.Errorf("cache body exceeds expected size")
	}
	if err == io.EOF && b.remaining > 0 {
		b.once.Do(b.onCorrupt)
		return count, io.ErrUnexpectedEOF
	}
	return count, err
}

func (b *validatedBody) Close() error { return b.body.Close() }
