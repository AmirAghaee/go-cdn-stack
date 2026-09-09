package natspublisher

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/edgehealth"
	"github.com/AmirAghaee/go-cdn-stack/pkg/messaging"
)

type Publisher struct {
	broker messaging.MessageBrokerInterface
}

type statusDTO struct {
	Service   string    `json:"service"`
	Instance  string    `json:"instance"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
}

func New(broker messaging.MessageBrokerInterface) *Publisher { return &Publisher{broker: broker} }

func (p *Publisher) Publish(ctx context.Context, status edgehealth.Status) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	payload, err := json.Marshal(statusDTO{
		Service: status.Service, Instance: status.Instance, Status: status.State,
		Timestamp: status.Timestamp, Version: status.Version,
	})
	if err != nil {
		return fmt.Errorf("encode health status: %w", err)
	}
	if err := p.broker.Publish("health", string(payload)); err != nil {
		return fmt.Errorf("publish health status: %w", err)
	}
	return nil
}
