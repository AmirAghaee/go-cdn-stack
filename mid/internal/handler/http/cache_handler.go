package http

import (
	"mid/internal/service"

	"github.com/gin-gonic/gin"
)

type CacheHandler struct {
	cacheService service.CacheServiceInterface
}

func NewCacheHandler(cacheService service.CacheServiceInterface) *CacheHandler {
	return &CacheHandler{
		cacheService: cacheService,
	}
}

func (h *CacheHandler) Register(r *gin.Engine) {
	r.Any("/*path", h.cacheRequest)
}

func (h *CacheHandler) cacheRequest(c *gin.Context) {
	h.cacheService.CacheRequest(c)
}
