package subscriber

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/domain"
	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/repository"
	"github.com/AmirAghaee/go-cdn-stack/pkg/messaging"
)

type HealthSubscriberInterface interface {
	Register() error
}

type healthSubscriber struct {
	broker messaging.MessageBrokerInterface
	repo   repository.HealthRepositoryInterface
}

func NewHealthSubscriber(broker messaging.MessageBrokerInterface, repo repository.HealthRepositoryInterface) HealthSubscriberInterface {
	return &healthSubscriber{
		broker: broker,
		repo:   repo,
	}
}

func (s *healthSubscriber) Register() error {
	return s.broker.Subscribe("health", func(msg string) {
		var status domain.HealthStatus
		if err := json.Unmarshal([]byte(msg), &status); err != nil {
			log.Printf("failed to unmarshal health message: %v", err)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.repo.Upsert(ctx, status); err != nil {
			log.Printf("failed to upsert health status: %v", err)
		} else {
			log.Printf("health updated: %s [%s] -> %s", status.Service, status.Instance, status.Status)
		}
	})
}
