package main

import (
	"fmt"
	"log"
	"os"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/client"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/config"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/handler/http"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/repository"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/service"
	"github.com/prometheus/client_golang/prometheus/promhttp"

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
	midService := service.NewMidService(midClient, cdnRepository, cfg, cfg.AppName, cfg.AppCacheURL, AppVersion)
	midService.StartSubmitHeartbeat()

	go startInternalPort(cfg)

	// Setup HTTP server
	gin.SetMode(cfg.GinMode)
	r := gin.Default()
	http.RegisterCacheRoutes(r, cacheService)

	fmt.Printf("Edge service running on %s\n", cfg.AppCacheURL)
	if err := r.Run(cfg.AppCacheURL); err != nil {
		panic(err)
	}
}

func startInternalPort(cfg *config.Config) {
	r := gin.Default()
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	fmt.Printf("Internal Edge API running on %s\n", cfg.AppInternalURL)
	if err := r.Run(cfg.AppInternalURL); err != nil {
		log.Fatalf("internal server failed: %v", err)
	}
}
