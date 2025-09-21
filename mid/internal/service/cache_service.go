package service

import (
	"mid/internal/config"
	"mid/internal/repository"
	"net/http"

	"github.com/gin-gonic/gin"
)

type CacheServiceInterface interface {
	CacheRequest(c *gin.Context)
}

type cacheService struct {
	config        *config.Config
	cdnRepository repository.CdnRepositoryInterface
}

func NewCacheService(config *config.Config, cdnRepo repository.CdnRepositoryInterface) CacheServiceInterface {
	return &cacheService{
		config:        config,
		cdnRepository: cdnRepo,
	}
}

func (s *cacheService) CacheRequest(c *gin.Context) {

	host := c.Request.Host
	cdn, ok := s.cdnRepository.GetByDomain(host)
	if !ok {
		c.String(http.StatusBadGateway, "Unknown host: %s", host)
		return
	}

	// Example: just return CDNs in JSON
	c.JSON(200, gin.H{
		"cdn": cdn,
	})
}
