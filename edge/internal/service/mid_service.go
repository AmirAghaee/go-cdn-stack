package service

import (
	"log"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/client"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/config"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/domain"
)

type MidServiceInterface interface {
	StartSubmitHeartbeat()
}

type midService struct {
	client   client.MidClientInterface
	config   *config.Config
	service  string
	instance string
	version  string
}

func NewMidService(client client.MidClientInterface, config *config.Config, service, instance, version string) MidServiceInterface {
	return &midService{
		client:   client,
		config:   config,
		service:  service,
		instance: instance,
		version:  version,
	}
}

func (s *midService) StartSubmitHeartbeat() {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			edge := domain.Edge{
				Service:   s.service,
				Instance:  s.instance,
				Status:    "ok",
				Timestamp: time.Now().UTC(),
				Version:   s.version,
			}

			if err := s.client.Submit(edge); err != nil {
				log.Printf("❌ failed to submit heartbeat: %v\n", err)
			} else {
				log.Println("✅ heartbeat sent")
			}
		}
	}()
}
