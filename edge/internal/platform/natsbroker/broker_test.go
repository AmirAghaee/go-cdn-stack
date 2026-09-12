package natsbroker

import (
	"bufio"
	"context"
	"errors"
	"io"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
)

func TestReconnectBackoffIsBounded(t *testing.T) {
	backoff := newReconnectBackoff(10*time.Millisecond, 100*time.Millisecond, rand.NewSource(1))
	for attempt := 0; attempt < 20; attempt++ {
		delay := backoff.delay(attempt)
		if delay < 10*time.Millisecond || delay > 100*time.Millisecond {
			t.Fatalf("delay(%d) = %v, want within [10ms, 100ms]", attempt, delay)
		}
	}
	if delay := backoff.delay(100); delay != 100*time.Millisecond {
		t.Fatalf("delay(100) = %v, want 100ms", delay)
	}
}

func TestInitialFailureRetriesAndActivatesSubscription(t *testing.T) {
	dialer := &recoveringDialer{subscribed: make(chan struct{}), published: make(chan struct{})}
	broker, err := newBroker(
		"nats://retry.test:4222",
		func(int) time.Duration { return time.Millisecond },
		nats.SetCustomDialer(dialer),
		nats.Timeout(20*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("newBroker() error = %v", err)
	}
	defer broker.Close()
	if broker.Connected() {
		t.Fatal("broker unexpectedly connected on its first attempt")
	}
	received := make(chan string, 1)
	if err := broker.Subscribe("cdn.snapshot", func(message string) { received <- message }); err != nil {
		t.Fatalf("Subscribe() while reconnecting error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := broker.WaitForConnection(ctx); err != nil {
		t.Fatalf("WaitForConnection() error = %v", err)
	}
	select {
	case <-dialer.subscribed:
	case <-ctx.Done():
		t.Fatal("subscription was not established after recovery")
	}
	select {
	case message := <-received:
		if message != `{}` {
			t.Fatalf("received message = %q, want %q", message, `{}`)
		}
	case <-ctx.Done():
		t.Fatal("notification was not delivered after recovery")
	}
	if attempts := dialer.attempts.Load(); attempts < 3 {
		t.Fatalf("dial attempts = %d, want at least 3", attempts)
	}
	if err := broker.Publish("health", `{}`); err != nil {
		t.Fatalf("Publish() after recovery error = %v", err)
	}
	select {
	case <-dialer.published:
	case <-ctx.Done():
		t.Fatal("health publish was not sent after recovery")
	}
}

func TestWaitForConnectionHonorsCancellation(t *testing.T) {
	dialer := &failingDialer{}
	broker, err := newBroker(
		"nats://unavailable.test:4222",
		func(int) time.Duration { return time.Millisecond },
		nats.SetCustomDialer(dialer),
		nats.Timeout(time.Millisecond),
	)
	if err != nil {
		t.Fatalf("newBroker() error = %v", err)
	}
	defer broker.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := broker.WaitForConnection(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("WaitForConnection() error = %v, want deadline exceeded", err)
	}
	if attempts := dialer.attempts.Load(); attempts < 2 {
		t.Fatalf("dial attempts = %d, want at least 2", attempts)
	}
}

type failingDialer struct{ attempts atomic.Int32 }

func (d *failingDialer) Dial(string, string) (net.Conn, error) {
	d.attempts.Add(1)
	return nil, errors.New("NATS unavailable")
}

type recoveringDialer struct {
	attempts   atomic.Int32
	subscribed chan struct{}
	published  chan struct{}
	subOnce    sync.Once
	pubOnce    sync.Once
}

func (d *recoveringDialer) Dial(string, string) (net.Conn, error) {
	if d.attempts.Add(1) < 3 {
		return nil, errors.New("NATS unavailable")
	}
	client, server := net.Pipe()
	go d.serve(server)
	return client, nil
}

func (d *recoveringDialer) serve(connection net.Conn) {
	defer connection.Close()
	_, _ = io.WriteString(connection, "INFO {\"server_id\":\"test\",\"server_name\":\"test\",\"version\":\"2.10.0\",\"proto\":1,\"host\":\"127.0.0.1\",\"port\":4222,\"headers\":true,\"max_payload\":1048576}\r\n")
	responses := make(chan string, 8)
	defer close(responses)
	go func() {
		for response := range responses {
			if _, err := io.WriteString(connection, response); err != nil {
				return
			}
		}
	}()
	reader := bufio.NewReader(connection)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		switch fields[0] {
		case "PING":
			responses <- "PONG\r\n"
		case "SUB":
			if len(fields) > 2 && fields[1] == "cdn.snapshot" {
				d.subOnce.Do(func() { close(d.subscribed) })
				responses <- "MSG cdn.snapshot " + fields[len(fields)-1] + " 2\r\n{}\r\n"
			}
		case "PUB":
			if len(fields) < 3 {
				return
			}
			size, err := strconv.Atoi(fields[len(fields)-1])
			if err != nil {
				return
			}
			if _, err := io.CopyN(io.Discard, reader, int64(size+2)); err != nil {
				return
			}
			if fields[1] == "health" {
				d.pubOnce.Do(func() { close(d.published) })
			}
		}
	}
}
