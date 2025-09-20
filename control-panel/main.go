package main

import (
	"context"
	"control-panel/internal/config"
	"control-panel/internal/handler/http"
	"control-panel/internal/messaging"
	"control-panel/internal/repository"
	"control-panel/internal/service"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOpts := options.Client().ApplyURI(cfg.MongoURI)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		log.Fatalf("mongo connect: %v", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("mongo ping: %v", err)
	}

	// setup NATS publisher
	natsPublisher, err := messaging.NewPublisher(cfg.NatsURL)
	if err != nil {
		log.Fatalf("messaging connect: %v", err)
	}

	// repositories
	userRepo := repository.NewUserRepository(client, cfg.DB)
	cdnRepo := repository.NewCdnRepository(client, cfg.DB)

	// services
	userService := service.NewUserService(userRepo)
	cdnService := service.NewCdnService(cdnRepo)

	// http handler
	r := gin.Default()
	http.RegisterRoutes(r, cdnService, userService, natsPublisher)

	port := cfg.Port
	if port == "" {
		port = "8090"
	}
	fmt.Printf("Server running on :%s\n", port)
	_ = r.Run(":" + port)
}
