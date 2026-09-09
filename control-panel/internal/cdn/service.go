package cdn

import (
	"context"
	"errors"
	"log"
)

var ErrExists = errors.New("cdn already exists")

type Store interface {
	Create(context.Context, *CDN) error
	List(context.Context) ([]*CDN, error)
	Get(context.Context, string) (*CDN, error)
	Update(context.Context, string, *CDN) error
	Delete(context.Context, string) error
	FindByOrigin(context.Context, string) (*CDN, error)
}

type RefreshNotifier interface {
	NotifyRefresh(context.Context) error
}

type Service struct {
	store    Store
	notifier RefreshNotifier
}

func NewService(store Store, notifier RefreshNotifier) *Service {
	return &Service{store: store, notifier: notifier}
}

func (s *Service) Create(ctx context.Context, origin, domain string, isActive bool, cacheTTL uint) error {
	if _, err := s.store.FindByOrigin(ctx, origin); err == nil {
		return ErrExists
	}

	item := &CDN{Origin: origin, Domain: domain, IsActive: isActive, CacheTTL: cacheTTL}
	if err := s.store.Create(ctx, item); err != nil {
		return err
	}
	s.notifyRefresh(ctx)
	return nil
}

func (s *Service) List(ctx context.Context) ([]*CDN, error) {
	return s.store.List(ctx)
}

func (s *Service) Get(ctx context.Context, id string) (*CDN, error) {
	return s.store.Get(ctx, id)
}

func (s *Service) Update(ctx context.Context, id, origin, domain string, isActive bool, cacheTTL uint) error {
	item := &CDN{Origin: origin, Domain: domain, IsActive: isActive, CacheTTL: cacheTTL}
	if err := s.store.Update(ctx, id, item); err != nil {
		return err
	}
	s.notifyRefresh(ctx)
	return nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.store.Delete(ctx, id); err != nil {
		return err
	}
	s.notifyRefresh(ctx)
	return nil
}

func (s *Service) Refresh(ctx context.Context) error {
	return s.notifier.NotifyRefresh(ctx)
}

func (s *Service) notifyRefresh(ctx context.Context) {
	if err := s.notifier.NotifyRefresh(ctx); err != nil {
		log.Printf("failed to publish CDN snapshot notification: %v", err)
	}
}
