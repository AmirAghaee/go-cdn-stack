package subscriber

import (
	"context"
	"log"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/service"
	"github.com/AmirAghaee/go-cdn-stack/pkg/messaging"
)

type CdnSnapshotSubscriber struct {
	broker  messaging.MessageBrokerInterface
	service service.CdnSnapshotServiceInterface
}

func NewCdnSnapshotSubscriber(broker messaging.MessageBrokerInterface, service service.CdnSnapshotServiceInterface) *CdnSnapshotSubscriber {
	return &CdnSnapshotSubscriber{broker: broker, service: service}
}

func (s *CdnSnapshotSubscriber) Register() error {
	return s.broker.Subscribe("cdn.snapshot", func(_ string) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := s.service.ProcessSnapshot(ctx); err != nil {
			log.Printf("event-driven CDN synchronization failed: %v", err)
		}
	})
}
