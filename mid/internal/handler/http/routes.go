package http

import (
	"mid/internal/service"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, cacheSvc service.CacheServiceInterface) {
	NewCacheHandler(cacheSvc).Register(r)
}
