package natshandler

import (
	"context"
	"log"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/pkg/messaging"
)

type Synchronizer interface {
	Sync(context.Context) error
}

type Handler struct {
	broker       messaging.MessageBrokerInterface
	synchronizer Synchronizer
}

func New(broker messaging.MessageBrokerInterface, synchronizer Synchronizer) *Handler {
	return &Handler{broker: broker, synchronizer: synchronizer}
}

func (h *Handler) Register(ctx context.Context) error {
	return h.broker.Subscribe("cdn.snapshot", func(_ string) {
		syncCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		if err := h.synchronizer.Sync(syncCtx); err != nil {
			log.Printf("event-driven CDN synchronization failed: %v", err)
		}
	})
}
