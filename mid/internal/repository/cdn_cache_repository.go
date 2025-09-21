package repository

import (
	"mid/internal/domain"
	"sync/atomic"
)

type CdnCacheRepositoryInterface interface {
	Set([]domain.CDN)
	GetAll() []domain.CDN
	GetByDomain(domain string) (domain.CDN, bool)
}

type cdnCacheRepository struct {
	data atomic.Value
}

func NewCdnCacheRepository() CdnCacheRepositoryInterface {
	repo := &cdnCacheRepository{}
	repo.data.Store(make(map[string]domain.CDN))
	return repo
}

func (c *cdnCacheRepository) Set(cdns []domain.CDN) {
	newMap := make(map[string]domain.CDN, len(cdns))
	for _, cdn := range cdns {
		newMap[cdn.Domain] = cdn
	}
	c.data.Store(newMap)
}

func (c *cdnCacheRepository) GetAll() []domain.CDN {
	m := c.data.Load().(map[string]domain.CDN)
	result := make([]domain.CDN, 0, len(m))
	for _, cdn := range m {
		result = append(result, cdn)
	}
	return result
}

func (c *cdnCacheRepository) GetByDomain(domainName string) (domain.CDN, bool) {
	m := c.data.Load().(map[string]domain.CDN)
	cdn, ok := m[domainName]
	return cdn, ok
}
