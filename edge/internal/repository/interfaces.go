package repository

import "cdneto/internal/domain"

type CacheRepository interface {
	Get(key string) (*domain.CacheItem, bool)
	Set(key string, item *domain.CacheItem)
	Delete(key string)
	LoadFromDisk()
	StartCleaner()
}
