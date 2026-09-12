package edgehealth

import (
	"context"
	"testing"
	"time"
)

type publisherStub struct {
	statuses []Status
}

func (p *publisherStub) Publish(_ context.Context, status Status) error {
	p.statuses = append(p.statuses, status)
	return nil
}

type readinessStub struct{ ready bool }

func (r *readinessStub) Ready() bool { return r.ready }

func TestPublishedStateReflectsReadiness(t *testing.T) {
	publisher := &publisherStub{}
	readiness := &readinessStub{}
	service := NewService(publisher, readiness, "edge", "edge-1", "v1", 0)

	service.publish(context.Background())
	readiness.ready = true
	service.publish(context.Background())

	if len(publisher.statuses) != 2 {
		t.Fatalf("published statuses = %d", len(publisher.statuses))
	}
	if publisher.statuses[0].State != "not_ready" || publisher.statuses[1].State != "ok" {
		t.Fatalf("published states = %q, %q", publisher.statuses[0].State, publisher.statuses[1].State)
	}
	for _, status := range publisher.statuses {
		if status.Timestamp.IsZero() || status.Timestamp.Location() != time.UTC {
			t.Fatalf("timestamp = %v, want UTC", status.Timestamp)
		}
	}
}
