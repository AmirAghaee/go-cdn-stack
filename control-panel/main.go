package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/config"
	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/handler/http"
	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/messaging"
	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/repository"
	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/service"
	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/subscriber"

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
	natsBroker, err := messaging.NewNatsBroker(cfg.NatsURL)
	if err != nil {
		log.Fatalf("messaging connect: %v", err)
	}

	// repositories
	userRepo := repository.NewUserRepository(client, cfg.DB)
	cdnRepo := repository.NewCdnRepository(client, cfg.DB)
	healthRepo := repository.NewHealthRepository(client, cfg.DB)

	// services
	userService := service.NewUserService(userRepo)
	cdnService := service.NewCdnService(cdnRepo)

	// subscribe to health events
	healthSub := subscriber.NewHealthSubscriber(natsBroker, healthRepo)
	if err := healthSub.Register(); err != nil {
		log.Fatalf("failed to register health subscriber: %v", err)
	}

	// http handler
	r := gin.Default()
	http.RegisterRoutes(r, cdnService, userService, natsBroker)

	fmt.Printf("Server running on :%s\n", cfg.AppUrl)
	_ = r.Run(cfg.AppUrl)
}
