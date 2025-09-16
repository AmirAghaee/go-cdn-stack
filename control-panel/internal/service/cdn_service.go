package service

import (
	"context"
	"control-panel/internal/domain"
	"control-panel/internal/repository"
)

type CdnServiceInterface interface {
	Create(ctx context.Context, origin, domain string, isActive bool) error
	List(ctx context.Context) ([]*domain.CDN, error)
	Get(ctx context.Context, id string) (*domain.CDN, error)
	Update(ctx context.Context, id, origin, domain string, isActive bool) error
	Delete(ctx context.Context, id string) error
}

type CdnService struct {
	repo repository.CdnRepositoryInterface
}

// NewCdnService returns a new CdnService
func NewCdnService(r repository.CdnRepositoryInterface) *CdnService {
	return &CdnService{
		repo: r,
	}
}

func (c *CdnService) Create(ctx context.Context, origin, domainName string, isActive bool) error {
	cdn := &domain.CDN{Origin: origin, Domain: domainName, IsActive: isActive}
	return c.repo.CreateCDN(ctx, cdn)
}

func (c *CdnService) List(ctx context.Context) ([]*domain.CDN, error) {
	return c.repo.ListCDNs(ctx)
}

func (c *CdnService) Get(ctx context.Context, id string) (*domain.CDN, error) {
	return c.repo.GetCDN(ctx, id)
}

func (c *CdnService) Update(ctx context.Context, id, origin, domainName string, isActive bool) error {
	cdn := &domain.CDN{Origin: origin, Domain: domainName, IsActive: isActive}
	return c.repo.UpdateCDN(ctx, id, cdn)
}

func (c *CdnService) Delete(ctx context.Context, id string) error {
	return c.repo.DeleteCDN(ctx, id)
}
