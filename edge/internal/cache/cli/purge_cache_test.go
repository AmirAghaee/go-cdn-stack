package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/cache"
)

type fakePurger struct {
	domain string
	result cache.PurgeResult
	err    error
}

func (p *fakePurger) PurgeDomain(_ context.Context, domain string) (cache.PurgeResult, error) {
	p.domain = domain
	return p.result, p.err
}

func TestPurgeCacheCommandNormalizesAndReportsResult(t *testing.T) {
	store := &fakePurgeStore{result: cache.PurgeResult{EntriesPurged: 2, BytesFreed: 42}}
	var stdout bytes.Buffer
	command := NewPurgeCacheCommand(cache.NewPurgeService(store))
	command.stdout = &stdout

	if err := command.Run(context.Background(), []string{"--domain", " CDN.Example.:443 "}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if store.domain != "cdn.example" {
		t.Fatalf("purged domain = %q, want cdn.example", store.domain)
	}
	if got := stdout.String(); got != "purged 2 cached entries (42 bytes) for domain cdn.example\n" {
		t.Fatalf("stdout = %q", got)
	}
}

func TestPurgeCacheCommandRequiresDomain(t *testing.T) {
	command := NewPurgeCacheCommand(&fakePurger{})
	if err := command.Run(context.Background(), nil); err == nil || !strings.Contains(err.Error(), "--domain is required") {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestPurgeCacheCommandRejectsUnexpectedArguments(t *testing.T) {
	command := NewPurgeCacheCommand(&fakePurger{})
	if err := command.Run(context.Background(), []string{"--domain", "cdn.example", "extra"}); err == nil || !strings.Contains(err.Error(), "unexpected arguments") {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestPurgeCacheCommandReportsZeroResult(t *testing.T) {
	purger := &fakePurger{result: cache.PurgeResult{Domain: "cdn.example"}}
	var stdout bytes.Buffer
	command := NewPurgeCacheCommand(purger)
	command.stdout = &stdout
	if err := command.Run(context.Background(), []string{"--domain", "cdn.example"}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := stdout.String(); got != "purged 0 cached entries (0 bytes) for domain cdn.example\n" {
		t.Fatalf("stdout = %q", got)
	}
}

func TestPurgeCacheCommandPropagatesPurgeError(t *testing.T) {
	wantErr := errors.New("remove failed")
	command := NewPurgeCacheCommand(&fakePurger{err: wantErr})
	err := command.Run(context.Background(), []string{"--domain", "cdn.example"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Run() error = %v, want wrapped purge error", err)
	}
}

type fakePurgeStore struct {
	domain string
	result cache.PurgeResult
}

func (s *fakePurgeStore) PurgeDomain(_ context.Context, domain string) (cache.PurgeResult, error) {
	s.domain = domain
	return s.result, nil
}
