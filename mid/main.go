package main

import (
	"fmt"
	"log"

	"github.com/AmirAghaee/go-cdn-stack/mid/internal/client"
	"github.com/AmirAghaee/go-cdn-stack/mid/internal/config"
	"github.com/AmirAghaee/go-cdn-stack/mid/internal/handler/http"
	"github.com/AmirAghaee/go-cdn-stack/mid/internal/repository"
	"github.com/AmirAghaee/go-cdn-stack/mid/internal/service"
	"github.com/AmirAghaee/go-cdn-stack/mid/internal/subscriber"
	"github.com/AmirAghaee/go-cdn-stack/pkg/messaging"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const AppVersion = "v1.0.0"

func main() {
	cfg := config.Load()

	// setup NATS publisher
	natsBroker, err := messaging.NewNatsBroker(cfg.NatsURL)
	if err != nil {
		log.Fatalf("messaging connect: %v", err)
	}

	// setup health publisher
	healthService := service.NewHealthService(natsBroker, cfg.AppName, cfg.AppCacheURL, AppVersion)
	stopChan := make(chan struct{})
	go healthService.Start(stopChan)
	defer close(stopChan)

	// setup clients
	controlPanelClient := client.NewControlPanelClient(cfg.ControlPanelURL)

	// setup repository
	cdnRepository := repository.NewCdnRepository()
	cacheItemRepository := repository.NewCacheItemRepository(cfg)

	// setup services
	cdnSnapshotService := service.NewCdnSnapshotService(controlPanelClient, cdnRepository)
	cacheService := service.NewCacheService(cfg, cdnRepository, cacheItemRepository)

	// first time sync with control panel
	if err := cdnSnapshotService.ProcessSnapshot(); err != nil {
		panic(err)
	}

	// setup subscribers
	cdnSnapshotSub := subscriber.NewCdnSnapshotSubscriber(natsBroker, cdnSnapshotService)
	if err := cdnSnapshotSub.Register(); err != nil {
		log.Fatalf("failed to register cdn snapshot subscriber: %v", err)
	}

	// Load existing cache and start cleaner
	cacheItemRepository.LoadFromDisk()
	cacheItemRepository.StartCleaner()

	go startInternalPort(cfg, cdnRepository)

	r := gin.Default()

	// Add metrics and health endpoints via middleware (before catch-all routes)
	r.Use(func(c *gin.Context) {
		// Handle metrics endpoint
		if c.Request.URL.Path == "/metrics" {
			promhttp.Handler().ServeHTTP(c.Writer, c.Request)
			c.Abort()
			return
		}

		// Handle health endpoint
		if c.Request.URL.Path == "/health" {
			c.JSON(200, gin.H{
				"status":  "healthy",
				"version": AppVersion,
			})
			c.Abort()
			return
		}

		c.Next()
	})

	http.RegisterCacheRoutes(r, cacheService)

	fmt.Printf("Mid cache server running on %s\n", cfg.AppCacheURL)
	fmt.Printf("Metrics available at %s/metrics\n", cfg.AppCacheURL)
	_ = r.Run(cfg.AppCacheURL)
}

func startInternalPort(cfg *config.Config, cdnRepository repository.CdnRepositoryInterface) {
	edgeRepository := repository.NewEdgeRepository()
	edgeService := service.NewEdgeService(edgeRepository, cdnRepository)

	r := gin.Default()

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	http.RegisterInternalRoutes(r, edgeService)

	fmt.Printf("Internal Mid API running on %s\n", cfg.AppInternalURL)
	if err := r.Run(cfg.AppInternalURL); err != nil {
		log.Fatalf("internal server failed: %v", err)
	}
}
