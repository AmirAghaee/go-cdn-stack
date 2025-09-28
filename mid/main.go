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
)

const AppVersion = "v1.0.0"

func main() {
	cfg := config.Load()

	// setup NATS publisher
	natsBroker, err := messaging.NewNatsBroker(cfg.NatsUrl)
	if err != nil {
		log.Fatalf("messaging connect: %v", err)
	}

	// setup health publisher
	healthService := service.NewHealthService(natsBroker, cfg.AppName, cfg.AppCacheUrl, AppVersion)
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
	http.RegisterCacheRoutes(r, cacheService)

	fmt.Printf("Server running on :%s\n", cfg.AppCacheUrl)
	_ = r.Run(cfg.AppCacheUrl)
}

func startInternalPort(cfg *config.Config, cdnRepository repository.CdnRepositoryInterface) {
	edgeRepository := repository.NewEdgeRepository()
	edgeService := service.NewEdgeService(edgeRepository, cdnRepository)

	r := gin.Default()
	http.RegisterInternalRoutes(r, edgeService)

	fmt.Printf("Internal Edge API running on %s\n", cfg.AppInternalUrl)
	if err := r.Run(cfg.AppInternalUrl); err != nil {
		log.Fatalf("internal server failed: %v", err)
	}
}
