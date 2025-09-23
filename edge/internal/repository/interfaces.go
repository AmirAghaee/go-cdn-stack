package repository

import "github.com/AmirAghaee/go-cdn-stack/edge/internal/domain"

type CacheRepository interface {
	Get(key string) (*domain.CacheItem, bool)
	Set(key string, item *domain.CacheItem)
	Delete(key string)
	LoadFromDisk()
	StartCleaner()
}
