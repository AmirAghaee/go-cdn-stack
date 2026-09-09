package main

import (
	"context"
	"log"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/cdn"
	cdnhttp "github.com/AmirAghaee/go-cdn-stack/control-panel/internal/cdn/httpapi"
	cdnmongo "github.com/AmirAghaee/go-cdn-stack/control-panel/internal/cdn/mongostore"
	cdnnats "github.com/AmirAghaee/go-cdn-stack/control-panel/internal/cdn/natspublisher"
	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/edgehealth"
	healthmongo "github.com/AmirAghaee/go-cdn-stack/control-panel/internal/edgehealth/mongostore"
	healthnats "github.com/AmirAghaee/go-cdn-stack/control-panel/internal/edgehealth/natshandler"
	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/identity"
	identityhttp "github.com/AmirAghaee/go-cdn-stack/control-panel/internal/identity/httpapi"
	identityjwt "github.com/AmirAghaee/go-cdn-stack/control-panel/internal/identity/jwttoken"
	identitymongo "github.com/AmirAghaee/go-cdn-stack/control-panel/internal/identity/mongostore"
	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/platform/config"
	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/platform/httpserver"
	"github.com/AmirAghaee/go-cdn-stack/pkg/jwt"
	"github.com/AmirAghaee/go-cdn-stack/pkg/messaging"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatalf("mongo connect: %v", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("mongo ping: %v", err)
	}

	broker, err := messaging.NewNatsBroker(cfg.NatsURL)
	if err != nil {
		log.Fatalf("messaging connect: %v", err)
	}

	tokenManager := identityjwt.New(jwt.NewJWTManager(cfg.JWTSecret, cfg.JWTDuration))

	identityService := identity.NewService(identitymongo.New(client.Database(cfg.DB)), tokenManager)
	refreshNotifier := cdnnats.New(broker)
	cdnService := cdn.NewService(cdnmongo.New(client.Database(cfg.DB)), refreshNotifier)
	healthRecorder := edgehealth.NewRecorder(healthmongo.New(client.Database(cfg.DB)))

	healthHandler := healthnats.New(broker, healthRecorder)
	if err := healthHandler.Register(); err != nil {
		log.Fatalf("register health subscriber: %v", err)
	}

	router := httpserver.New(
		identityhttp.New(identityService),
		cdnhttp.New(cdnService),
		identityhttp.Auth(tokenManager),
	)

	log.Printf("control panel listening on %s", cfg.AppURL)
	if err := router.Run(cfg.AppURL); err != nil {
		log.Fatalf("run HTTP server: %v", err)
	}
}
