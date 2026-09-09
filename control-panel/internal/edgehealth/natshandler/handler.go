package natshandler

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/edgehealth"
	"github.com/AmirAghaee/go-cdn-stack/pkg/messaging"
)

type Handler struct {
	broker   messaging.MessageBrokerInterface
	recorder *edgehealth.Recorder
}

type statusMessage struct {
	Service   string    `json:"service"`
	Instance  string    `json:"instance"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
}

func New(broker messaging.MessageBrokerInterface, recorder *edgehealth.Recorder) *Handler {
	return &Handler{broker: broker, recorder: recorder}
}

func (h *Handler) Register() error {
	return h.broker.Subscribe("health", h.handle)
}

func (h *Handler) handle(message string) {
	var input statusMessage
	if err := json.Unmarshal([]byte(message), &input); err != nil {
		log.Printf("failed to unmarshal health message: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status := edgehealth.Status{
		Service: input.Service, Instance: input.Instance, Status: input.Status,
		Timestamp: input.Timestamp, Version: input.Version,
	}
	if err := h.recorder.Record(ctx, status); err != nil {
		log.Printf("failed to upsert health status: %v", err)
		return
	}
	log.Printf("health updated: %s [%s] -> %s", status.Service, status.Instance, status.Status)
}
