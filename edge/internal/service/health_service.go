package service

import (
	"encoding/json"
	"log"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/domain"
	"github.com/AmirAghaee/go-cdn-stack/pkg/messaging"
)

type HealthServiceInterface interface {
	Start(<-chan struct{})
}

type healthService struct {
	broker   messaging.MessageBrokerInterface
	service  string
	instance string
	version  string
}

func NewHealthService(broker messaging.MessageBrokerInterface, service, instance, version string) HealthServiceInterface {
	return &healthService{broker: broker, service: service, instance: instance, version: version}
}

func (h *healthService) Start(stop <-chan struct{}) {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		h.publish()
		for {
			select {
			case <-ticker.C:
				h.publish()
			case <-stop:
				return
			}
		}
	}()
}

func (h *healthService) publish() {
	status := domain.Edge{
		Service: h.service, Instance: h.instance, Status: "ok",
		Timestamp: time.Now().UTC(), Version: h.version,
	}
	payload, err := json.Marshal(status)
	if err != nil {
		log.Printf("failed to marshal edge health: %v", err)
		return
	}
	if err := h.broker.Publish("health", string(payload)); err != nil {
		log.Printf("failed to publish edge health: %v", err)
	}
}
