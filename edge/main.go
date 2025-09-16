package main

import (
	"edge/internal/config"
	"edge/internal/handler"
	"edge/internal/repository"
	"edge/internal/service"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Ensure cache directory exists
	if err := os.MkdirAll(cfg.CacheDir, 0755); err != nil {
		panic(err)
	}

	// Initialize dependencies
	cacheRepo := repository.NewInMemoryCache(cfg)
	edgeService := service.NewEdgeService(cacheRepo, cfg)
	httpHandler := handler.NewHTTPHandler(edgeService)

	// Load existing cache and start cleaner
	cacheRepo.LoadFromDisk()
	cacheRepo.StartCleaner()

	// Setup HTTP server
	gin.SetMode(cfg.GinMode)
	r := gin.Default()

	r.GET("/*path", httpHandler.HandleCacheRequest)
	// All other methods are proxied directly to origin
	r.POST("/*path", httpHandler.HandleProxyRequest)
	r.PUT("/*path", httpHandler.HandleProxyRequest)
	r.DELETE("/*path", httpHandler.HandleProxyRequest)
	r.PATCH("/*path", httpHandler.HandleProxyRequest)
	r.HEAD("/*path", httpHandler.HandleProxyRequest)
	r.OPTIONS("/*path", httpHandler.HandleProxyRequest)

	fmt.Printf("Edge service running on %s\n", cfg.Port)
	if err := r.Run(cfg.Port); err != nil {
		panic(err)
	}
}
