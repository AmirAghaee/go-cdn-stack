package cache

import (
	"context"
	"errors"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/cdn"
)

type PurgeResult struct {
	Domain        string
	EntriesPurged int
	BytesFreed    int64
}

type DomainPurgeStore interface {
	PurgeDomain(context.Context, string) (PurgeResult, error)
}

type PurgeService struct {
	store DomainPurgeStore
}

func NewPurgeService(store DomainPurgeStore) *PurgeService {
	return &PurgeService{store: store}
}

func (s *PurgeService) PurgeDomain(ctx context.Context, domain string) (PurgeResult, error) {
	domain = cdn.NormalizeDomain(domain)
	if domain == "" {
		return PurgeResult{}, errors.New("domain is required")
	}
	result, err := s.store.PurgeDomain(ctx, domain)
	result.Domain = domain
	return result, err
}
