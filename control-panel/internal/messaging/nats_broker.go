package messaging

import (
	"log"

	"github.com/nats-io/nats.go"
)

type NatsBroker struct {
	conn *nats.Conn
}

func NewPublisher(url string) (MessageBrokerInterface, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}
	return &NatsBroker{conn: nc}, nil
}

func (p *NatsBroker) Publish(subject, msg string) error {
	if err := p.conn.Publish(subject, []byte(msg)); err != nil {
		log.Printf("failed to publish message: %v", err)
		return err
	}
	return nil
}
