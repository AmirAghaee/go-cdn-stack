package service

import (
	"mid/internal/config"
	"mid/internal/repository"

	"github.com/gin-gonic/gin"
)

type CacheServiceInterface interface {
	CacheRequest(c *gin.Context)
}

type cacheService struct {
	config             *config.Config
	cdnCacheRepository repository.CdnCacheRepositoryInterface
}

func NewCacheService(config *config.Config, cache repository.CdnCacheRepositoryInterface) CacheServiceInterface {
	return &cacheService{
		config:             config,
		cdnCacheRepository: cache,
	}
}

func (s *cacheService) CacheRequest(c *gin.Context) {

	cdns := s.cdnCacheRepository.GetAll()

	// Example: just return CDNs in JSON
	c.JSON(200, gin.H{
		"cdns": cdns,
	})
}
