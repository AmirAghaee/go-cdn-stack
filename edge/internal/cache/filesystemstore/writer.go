package filesystemstore

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/cache"
)

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

func (s *Store) Begin(key string, entry cache.EntryMetadata) (cache.PendingEntry, error) {
	if !time.Now().Before(entry.ExpiresAt) {
		return nil, fmt.Errorf("cache entry is already expired")
	}
	if entry.StoredAt.IsZero() || entry.InitialAge < 0 {
		return nil, fmt.Errorf("cache entry timing metadata is invalid")
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
		oldPath = value.FilePath
	}
	return &pendingEntry{
		store: s,
		key:   key,
		metadata: metadata{
			Key: key, Header: entry.Header, ExpiresAt: entry.ExpiresAt,
			StoredAt: entry.StoredAt, InitialAgeNanoseconds: int64(entry.InitialAge),
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
	if !ok || current.FilePath != p.finalPath {
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
