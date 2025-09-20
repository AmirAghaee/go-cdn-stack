package nats

import (
	"log"

	"github.com/nats-io/nats.go"
)

type PublisherInterface interface {
	Publish(subject, msg string) error
}

type Publisher struct {
	conn *nats.Conn
}

func NewPublisher(url string) (*Publisher, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}
	return &Publisher{conn: nc}, nil
}

func (p *Publisher) Publish(subject, msg string) error {
	if err := p.conn.Publish(subject, []byte(msg)); err != nil {
		log.Printf("failed to publish message: %v", err)
		return err
	}
	return nil
}
