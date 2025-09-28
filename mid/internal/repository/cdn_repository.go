package repository

import (
	"sync/atomic"

	"github.com/AmirAghaee/go-cdn-stack/mid/internal/domain"
	"github.com/google/uuid"
)

type CdnRepositoryInterface interface {
	Set([]domain.CDN)
	GetAll() []domain.CDN
	GetByDomain(domain string) (domain.CDN, bool)
	GetVersion() string
}

type cdnRepository struct {
	data    atomic.Value
	version string
}

func NewCdnRepository() CdnRepositoryInterface {
	repo := &cdnRepository{}
	repo.data.Store(make(map[string]domain.CDN))
	repo.version = uuid.NewString()
	return repo
}

func (c *cdnRepository) Set(cdns []domain.CDN) {
	newMap := make(map[string]domain.CDN, len(cdns))
	for _, cdn := range cdns {
		newMap[cdn.Domain] = cdn
	}
	c.data.Store(newMap)
	c.version = uuid.NewString()
}

func (c *cdnRepository) GetAll() []domain.CDN {
	m := c.data.Load().(map[string]domain.CDN)
	result := make([]domain.CDN, 0, len(m))
	for _, cdn := range m {
		result = append(result, cdn)
	}
	return result
}

func (c *cdnRepository) GetByDomain(domainName string) (domain.CDN, bool) {
	m := c.data.Load().(map[string]domain.CDN)
	cdn, ok := m[domainName]
	return cdn, ok
}

func (c *cdnRepository) GetVersion() string {
	return c.version
}
