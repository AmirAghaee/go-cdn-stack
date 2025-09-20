package subscriber

import (
	"log"
	"mid/internal/messaging"
	"mid/internal/service"
)

type CdnSnapshotSubscriberInterface interface {
	Register() error
}

type CdnSnapshotSubscriber struct {
	broker  messaging.MessageBrokerInterface
	service service.CdnSnapshotServiceInterface
}

func NewCdnSnapshotSubscriber(broker messaging.MessageBrokerInterface, service service.CdnSnapshotServiceInterface) CdnSnapshotSubscriberInterface {
	return &CdnSnapshotSubscriber{
		broker:  broker,
		service: service,
	}
}

func (s *CdnSnapshotSubscriber) Register() error {
	return s.broker.Subscribe("cdn.snapshot", func(msg string) {
		if err := s.service.ProcessSnapshot(); err != nil {
			log.Printf("❌ failed processing snapshot: %v", err)
		}
	})
}
