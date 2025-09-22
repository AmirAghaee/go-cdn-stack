package service

import (
	"encoding/json"
	"log"
	"mid/internal/domain"
	"mid/internal/messaging"
	"time"
)

type HealthServiceInterface interface {
	Start(stopChan <-chan struct{})
}

type healthService struct {
	broker   messaging.MessageBrokerInterface
	service  string
	instance string
	version  string
}

func NewHealthService(broker messaging.MessageBrokerInterface, service, instance, version string) HealthServiceInterface {
	return &healthService{
		broker:   broker,
		service:  service,
		instance: instance,
		version:  version,
	}
}

func (h *healthService) Start(stopChan <-chan struct{}) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.publishHealth()
		case <-stopChan:
			log.Println("stopping health publisher")
			return
		}
	}
}

func (h *healthService) publishHealth() {
	health := domain.HealthStatus{
		Service:   h.service,
		Instance:  h.instance,
		Status:    "ok",
		Timestamp: time.Now().UTC(),
		Version:   h.version,
	}

	payload, err := json.Marshal(health)
	if err != nil {
		log.Printf("failed to marshal health payload: %v", err)
		return
	}

	if err := h.broker.Publish("health", string(payload)); err != nil {
		log.Printf("failed to publish health: %v", err)
	}

}
