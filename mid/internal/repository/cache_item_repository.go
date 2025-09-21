package repository

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"mid/internal/config"
	"mid/internal/domain"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type CacheItemRepositoryInterface interface {
	Get(key string) (*domain.CacheItem, bool)
	Set(key string, item *domain.CacheItem)
	Delete(key string)
	LoadFromDisk()
	StartCleaner()
}

type cacheItemRepository struct {
	cache  map[string]*domain.CacheItem
	mutex  sync.RWMutex
	config *config.Config
}

func NewCacheItemRepository(config *config.Config) CacheItemRepositoryInterface {
	return &cacheItemRepository{
		cache:  make(map[string]*domain.CacheItem),
		config: config,
	}
}

func (r *cacheItemRepository) Get(key string) (*domain.CacheItem, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	item, found := r.cache[key]
	if !found || time.Now().After(item.ExpiresAt) {
		return nil, false
	}
	return item, true
}

func (r *cacheItemRepository) Set(key string, item *domain.CacheItem) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.cache[key] = item
}

func (r *cacheItemRepository) Delete(key string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.cache, key)
}

func (r *cacheItemRepository) LoadFromDisk() {
	files, err := os.ReadDir(r.config.CacheDir)
	if err != nil {
		fmt.Println("Error reading cache dir:", err)
		return
	}

	count := 0
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".cache.json") {
			continue
		}

		metaFile := filepath.Join(r.config.CacheDir, f.Name())
		data, err := os.ReadFile(metaFile)
		if err != nil {
			continue
		}

		var item domain.CacheItem
		if err := json.Unmarshal(data, &item); err != nil {
			continue
		}

		if time.Now().Before(item.ExpiresAt) {
			hexName := strings.TrimSuffix(f.Name(), ".cache.json")
			if keyBytes, err := hex.DecodeString(hexName); err == nil {
				r.Set(string(keyBytes), &item)
				count++
			}
		}
	}
	fmt.Printf("Loaded %d cache items from disk\n", count)
}

func (r *cacheItemRepository) StartCleaner() {
	go func() {
		ticker := time.NewTicker(r.config.CleanerInterval)
		defer ticker.Stop()

		for range ticker.C {
			now := time.Now()
			r.mutex.Lock()
			for key, item := range r.cache {
				if now.After(item.ExpiresAt) {
					_ = os.Remove(item.FilePath)
					_ = os.Remove(item.FilePath + ".json")
					delete(r.cache, key)
					log.Printf("Deleted expired cache: %s", item.FilePath)
				}
			}
			r.mutex.Unlock()
		}
	}()
}
