package service

import (
	"mid/internal/config"

	"github.com/gin-gonic/gin"
)

type CacheServiceInterface interface {
	CacheRequest(c *gin.Context)
}

type CacheService struct {
	config *config.Config
}

func NewCacheService(config *config.Config) *CacheService {
	return &CacheService{
		config: config,
	}
}

func (s *CacheService) CacheRequest(c *gin.Context) {

}
