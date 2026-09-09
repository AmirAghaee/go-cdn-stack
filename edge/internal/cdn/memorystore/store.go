package memorystore

import (
	"sync/atomic"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/cdn"
)

type Store struct {
	data atomic.Value
}

func New() *Store {
	store := &Store{}
	store.data.Store(make(map[string]cdn.CDN))
	return store
}

func (s *Store) Replace(items []cdn.CDN) {
	next := make(map[string]cdn.CDN, len(items))
	for _, item := range items {
		next[item.Domain()] = item
	}
	s.data.Store(next)
}

func (s *Store) FindByDomain(domain string) (cdn.CDN, bool) {
	items := s.data.Load().(map[string]cdn.CDN)
	item, ok := items[cdn.NormalizeDomain(domain)]
	return item, ok && item.IsActive()
}
