package http

import (
	"github.com/AmirAghaee/go-cdn-stack/mid/internal/service"

	"github.com/gin-gonic/gin"
)

func RegisterCacheRoutes(r *gin.Engine, cacheSvc service.CacheServiceInterface) {
	NewCacheHandler(cacheSvc).Register(r)
}

func RegisterInternalRoutes(r *gin.Engine, cacheSvc service.EdgeServiceInterface) {
	NewEdgeHandler(cacheSvc).Register(r)
}
