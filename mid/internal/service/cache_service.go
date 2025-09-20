package service

import (
	"github.com/gin-gonic/gin"
)

type CacheServiceInterface interface {
	CacheRequest(c *gin.Context)
}

type CacheService struct {
}

func NewCacheService() *CacheService {
	return &CacheService{}
}

func (s *CacheService) CacheRequest(c *gin.Context) {

}
