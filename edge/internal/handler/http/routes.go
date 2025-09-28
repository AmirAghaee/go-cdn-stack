package http

import (
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/service"
	"github.com/gin-gonic/gin"
)

func RegisterCacheRoutes(r *gin.Engine, cacheSvc service.CacheServiceInterface) {
	NewCacheHandler(cacheSvc).Register(r)
}
