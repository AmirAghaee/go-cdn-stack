package repository

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/mid/internal/config"
	"github.com/AmirAghaee/go-cdn-stack/mid/internal/domain"
	"github.com/dgraph-io/ristretto"
)

type CacheItemRepositoryInterface interface {
	Get(key string) (*domain.CacheItem, bool)
	Set(key string, item *domain.CacheItem)
	Delete(key string)
	LoadFromDisk()
	StartCleaner()
}

type cacheItemRepository struct {
	cache  *ristretto.Cache
	config *config.Config
}

func NewCacheItemRepository(cfg *config.Config) CacheItemRepositoryInterface {
	// Ristretto config
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // Adjust based on expected key count
		MaxCost:     1 << 28, // 256MB max (adjust depending on memory)
		BufferItems: 64,
	})
	if err != nil {
		panic(err)
	}

	return &cacheItemRepository{
		cache:  cache,
		config: cfg,
	}
}

func (r *cacheItemRepository) Get(key string) (*domain.CacheItem, bool) {
	value, ok := r.cache.Get(key)
	if !ok {
		return nil, false
	}

	return value.(*domain.CacheItem), true
}

func (r *cacheItemRepository) Set(key string, item *domain.CacheItem) {
	ttl := time.Until(item.ExpiresAt)

	// If already expired, do nothing
	if ttl <= 0 {
		return
	}

	r.cache.SetWithTTL(key, item, 1, ttl)
}

func (r *cacheItemRepository) Delete(key string) {
	r.cache.Del(key)
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
		ticker := time.NewTicker(r.config.CleanerIntervalDuration)
		defer ticker.Stop()

		for range ticker.C {
			files, _ := os.ReadDir(r.config.CacheDir)

			for _, f := range files {
				if !strings.HasSuffix(f.Name(), ".cache.json") {
					continue
				}

				metaFile := filepath.Join(r.config.CacheDir, f.Name())
				data, err := os.ReadFile(metaFile)
				if err != nil {
					continue
				}

				var item domain.CacheItem
				if json.Unmarshal(data, &item) != nil {
					continue
				}

				if time.Now().After(item.ExpiresAt) {
					_ = os.Remove(item.FilePath)
					_ = os.Remove(metaFile)
					log.Printf("Deleted expired cache (disk): %s", item.FilePath)
				}
			}
		}
	}()
}
