package natsbroker

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	defaultMinReconnectDelay = 500 * time.Millisecond
	defaultMaxReconnectDelay = 30 * time.Second
)

type Broker struct {
	connection  *nats.Conn
	connected   chan struct{}
	connectOnce sync.Once
}

func New(url string) (*Broker, error) {
	backoff := newReconnectBackoff(defaultMinReconnectDelay, defaultMaxReconnectDelay, rand.NewSource(time.Now().UnixNano()))
	return newBroker(url, backoff.delay)
}

func newBroker(url string, reconnectDelay nats.ReconnectDelayHandler, options ...nats.Option) (*Broker, error) {
	broker := &Broker{connected: make(chan struct{})}
	markConnected := func(*nats.Conn) {
		broker.connectOnce.Do(func() { close(broker.connected) })
	}
	connectionOptions := []nats.Option{
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.CustomReconnectDelay(reconnectDelay),
		nats.ConnectHandler(markConnected),
		nats.ReconnectHandler(markConnected),
	}
	connectionOptions = append(connectionOptions, options...)
	connection, err := nats.Connect(url, connectionOptions...)
	if err != nil {
		return nil, err
	}
	broker.connection = connection
	if connection.IsConnected() {
		markConnected(connection)
	}
	return broker, nil
}

func (b *Broker) Publish(subject, message string) error {
	return b.connection.Publish(subject, []byte(message))
}

func (b *Broker) Subscribe(subject string, handler func(message string)) error {
	_, err := b.connection.Subscribe(subject, func(message *nats.Msg) {
		handler(string(message.Data))
	})
	return err
}

func (b *Broker) Connected() bool { return b.connection.IsConnected() }

func (b *Broker) WaitForConnection(ctx context.Context) error {
	select {
	case <-b.connected:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (b *Broker) Close() { b.connection.Close() }

type reconnectBackoff struct {
	minimum time.Duration
	maximum time.Duration
	random  *rand.Rand
	mu      sync.Mutex
}

func newReconnectBackoff(minimum, maximum time.Duration, source rand.Source) *reconnectBackoff {
	return &reconnectBackoff{minimum: minimum, maximum: maximum, random: rand.New(source)}
}

func (b *reconnectBackoff) delay(attempt int) time.Duration {
	delay := b.minimum
	for step := 0; step < attempt && delay < b.maximum; step++ {
		if delay > b.maximum/2 {
			delay = b.maximum
			break
		}
		delay *= 2
	}
	if delay >= b.maximum {
		return b.maximum
	}

	jitterLimit := delay / 2
	if remaining := b.maximum - delay; jitterLimit > remaining {
		jitterLimit = remaining
	}
	if jitterLimit <= 0 {
		return delay
	}
	b.mu.Lock()
	jitter := time.Duration(b.random.Int63n(int64(jitterLimit) + 1))
	b.mu.Unlock()
	return delay + jitter
}
