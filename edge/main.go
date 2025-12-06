package main

import (
	"fmt"
	"os"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/client"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/config"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/handler/http"
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
	midClient := client.NewMidClient(cfg.MidInternalURL)

	// setup repository
	cdnRepository := repository.NewCdnRepository()
	cacheItemRepository := repository.NewCacheItemRepository(cfg)

	// setup services
	cacheService := service.NewCacheService(cfg, cdnRepository, cacheItemRepository)

	// Load existing cache and start cleaner
	cacheItemRepository.LoadFromDisk()
	cacheItemRepository.StartCleaner()

	//  setup services
	midService := service.NewMidService(midClient, cdnRepository, cfg, cfg.AppName, cfg.AppURL, AppVersion)
	midService.StartSubmitHeartbeat()

	// Setup HTTP server
	gin.SetMode(cfg.GinMode)
	r := gin.Default()
	http.RegisterCacheRoutes(r, cacheService)

	fmt.Printf("Edge service running on %s\n", cfg.AppURL)
	if err := r.Run(cfg.AppURL); err != nil {
		panic(err)
	}
}
