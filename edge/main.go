package main

import (
	"fmt"
	"os"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/client"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/config"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/handler"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/repository"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/service"

	"github.com/gin-gonic/gin"
)

const AppVersion = "v1.0.0"

func main() {
	// Load configuration
	cfg := config.Load()

	// Ensure cache directory exists
	if err := os.MkdirAll(cfg.CacheDir, 0755); err != nil {
		panic(err)
	}

	// setup clients
	midClient := client.NewMidClient(cfg.MidInternalUrl)

	// Initialize dependencies
	cacheRepo := repository.NewInMemoryCache(cfg)
	cdnRepo := repository.NewCdnRepository()
	edgeService := service.NewEdgeService(cacheRepo, cfg)
	httpHandler := handler.NewHTTPHandler(edgeService)

	// Load existing cache and start cleaner
	cacheRepo.LoadFromDisk()
	cacheRepo.StartCleaner()

	//  setup services
	midService := service.NewMidService(midClient, cdnRepo, cfg, cfg.AppName, cfg.AppUrl, AppVersion)
	midService.StartSubmitHeartbeat()

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

	fmt.Printf("Edge service running on %s\n", cfg.AppUrl)
	if err := r.Run(cfg.AppUrl); err != nil {
		panic(err)
	}
}
