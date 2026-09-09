package natspublisher

import (
	"context"

	"github.com/AmirAghaee/go-cdn-stack/pkg/messaging"
)

const snapshotSubject = "cdn.snapshot"

type Publisher struct {
	broker messaging.MessageBrokerInterface
}

func New(broker messaging.MessageBrokerInterface) *Publisher {
	return &Publisher{broker: broker}
}

func (p *Publisher) NotifyRefresh(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return p.broker.Publish(snapshotSubject, `{"event":"snapshot"}`)
}
