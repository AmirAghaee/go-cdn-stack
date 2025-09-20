package main

import (
	"fmt"
	"log"
	"mid/internal/config"
	"mid/internal/messaging"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	// setup NATS publisher
	natsBroker, err := messaging.NewNatsBroker(cfg.NatsUrl)
	if err != nil {
		log.Fatalf("messaging connect: %v", err)
	}

	err = natsBroker.Subscribe("cdn.snapshot", func(msg string) {
		fmt.Println("📩 Received message:", msg)
	})
	if err != nil {
		log.Fatalf("subscribe failed: %v", err)
	}

	r := gin.Default()
	//	h := http.NewHTTPHandler(cdnService, userService, natsPublisher)
	//h.Register(r)

	fmt.Printf("Server running on :%s\n", cfg.Port)
	_ = r.Run(":" + cfg.Port)
}
