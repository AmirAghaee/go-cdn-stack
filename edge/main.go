package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/client"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/config"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/handler/http"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/repository"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/service"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/subscriber"
	"github.com/AmirAghaee/go-cdn-stack/pkg/messaging"
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

	// setup repository
	cdnRepository := repository.NewCdnRepository()
	cacheItemRepository := repository.NewCacheItemRepository(cfg)

	controlPanelClient := client.NewControlPanelClient(cfg.ControlPanelURL, cfg.JWTSecret)
	snapshotService := service.NewCdnSnapshotService(
		controlPanelClient,
		cdnRepository,
		cfg.SnapshotFile,
		cfg.SyncIntervalDuration,
	)
	if err := snapshotService.LoadLocal(); err != nil {
		log.Printf("failed to load local CDN snapshot: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := snapshotService.ProcessSnapshot(ctx); err != nil {
		log.Printf("initial CDN synchronization failed; using last-known-good snapshot: %v", err)
	}
	cancel()

	cacheService := service.NewCacheService(cfg, cdnRepository, cacheItemRepository)

	// Load existing cache and start cleaner
	cacheItemRepository.LoadFromDisk()
	cacheItemRepository.StartCleaner()

	stop := make(chan struct{})
	defer close(stop)
	snapshotService.StartPeriodic(stop)

	// NATS accelerates configuration refresh and carries health updates. The
	// periodic control-panel sync remains available if NATS is temporarily down.
	if broker, err := messaging.NewNatsBroker(cfg.NatsURL); err != nil {
		log.Printf("NATS unavailable; continuing with periodic CDN sync: %v", err)
	} else {
		if err := subscriber.NewCdnSnapshotSubscriber(broker, snapshotService).Register(); err != nil {
			log.Printf("failed to subscribe to CDN snapshots: %v", err)
		}
		service.NewHealthService(broker, "edge", cfg.AppName, AppVersion).Start(stop)
	}

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
