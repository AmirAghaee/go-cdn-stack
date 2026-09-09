package edgehealth

import (
	"context"
	"log"
	"time"
)

type Publisher interface {
	Publish(context.Context, Status) error
}

type Service struct {
	publisher Publisher
	status    Status
	interval  time.Duration
}

func NewService(publisher Publisher, service, instance, version string, interval time.Duration) *Service {
	return &Service{
		publisher: publisher,
		status:    Status{Service: service, Instance: instance, State: "ok", Version: version},
		interval:  interval,
	}
}

func (s *Service) Run(ctx context.Context) {
	s.publish(ctx)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.publish(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (s *Service) publish(ctx context.Context) {
	status := s.status
	status.Timestamp = time.Now().UTC()
	if err := s.publisher.Publish(ctx, status); err != nil {
		log.Printf("publish edge health: %v", err)
	}
}
