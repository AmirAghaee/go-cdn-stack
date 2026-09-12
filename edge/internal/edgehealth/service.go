package edgehealth

import (
	"context"
	"log"
	"time"
)

type Publisher interface {
	Publish(context.Context, Status) error
}

type Readiness interface {
	Ready() bool
}

type Service struct {
	publisher Publisher
	readiness Readiness
	status    Status
	interval  time.Duration
}

func NewService(publisher Publisher, readiness Readiness, service, instance, version string, interval time.Duration) *Service {
	return &Service{
		publisher: publisher,
		readiness: readiness,
		status:    Status{Service: service, Instance: instance, Version: version},
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
	status.State = "not_ready"
	if s.readiness.Ready() {
		status.State = "ok"
	}
	status.Timestamp = time.Now().UTC()
	if err := s.publisher.Publish(ctx, status); err != nil {
		log.Printf("publish edge health: %v", err)
	}
}
