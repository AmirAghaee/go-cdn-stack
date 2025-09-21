package main

import (
	"fmt"
	"log"
	"mid/internal/client"
	"mid/internal/config"
	"mid/internal/handler/http"
	"mid/internal/messaging"
	"mid/internal/service"
	"mid/internal/subscriber"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	// setup NATS publisher
	natsBroker, err := messaging.NewNatsBroker(cfg.NatsUrl)
	if err != nil {
		log.Fatalf("messaging connect: %v", err)
	}

	// setup clients
	controlPanelClient := client.NewControlPanelClient(cfg.ControlPanelURL)

	// setup services
	cdnSnapshotService := service.NewCdnSnapshotService(controlPanelClient)
	cacheService := service.NewCacheService(cfg)

	// first time sync with control panel
	if err := cdnSnapshotService.ProcessSnapshot(); err != nil {
		panic(err)
	}

	// setup subscribers
	cdnSnapshotSub := subscriber.NewCdnSnapshotSubscriber(natsBroker, cdnSnapshotService)
	if err := cdnSnapshotSub.Register(); err != nil {
		log.Fatalf("failed to register cdn snapshot subscriber: %v", err)
	}

	r := gin.Default()
	http.RegisterRoutes(r, cacheService)

	fmt.Printf("Server running on :%s\n", cfg.Port)
	_ = r.Run(":" + cfg.Port)
}
