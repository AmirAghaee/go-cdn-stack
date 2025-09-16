package main

import (
	"cdneto/internal/config"
	"cdneto/internal/handler"
	"cdneto/internal/repository"
	"cdneto/internal/service"
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
	r := gin.Default()
	r.GET("/*path", httpHandler.HandleRequest)

	fmt.Printf("Edge service running on %s\n", cfg.Port)
	if err := r.Run(cfg.Port); err != nil {
		panic(err)
	}
}
