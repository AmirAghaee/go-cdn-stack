package messaging

import (
	"log"

	"github.com/nats-io/nats.go"
)

type NatsBroker struct {
	conn *nats.Conn
}

func NewNatsBroker(url string) (MessageBrokerInterface, error) {
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

func (p *NatsBroker) Subscribe(subject string, handler func(msg string)) error {
	_, err := p.conn.Subscribe(subject, func(m *nats.Msg) {
		handler(string(m.Data))
	})
	if err != nil {
		log.Printf("failed to subscribe to subject %s: %v", subject, err)
		return err
	}
	return nil
}
