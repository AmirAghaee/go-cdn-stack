package repository

import (
	"sync"

	"github.com/AmirAghaee/go-cdn-stack/mid/internal/domain"
)

type EdgeRepositoryInterface interface {
	Set(edge domain.Edge)
	SetAll([]domain.Edge)
	GetAll() []domain.Edge
}

type edgeRepository struct {
	mu   sync.RWMutex
	data map[string]domain.Edge
}

func NewEdgeRepository() EdgeRepositoryInterface {
	return &edgeRepository{
		data: make(map[string]domain.Edge),
	}
}

func (r *edgeRepository) Set(edge domain.Edge) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[edge.Service] = edge
}

func (r *edgeRepository) SetAll(edges []domain.Edge) {
	r.mu.Lock()
	defer r.mu.Unlock()

	newMap := make(map[string]domain.Edge, len(edges))
	for _, e := range edges {
		newMap[e.Instance] = e
	}
	r.data = newMap
}

func (r *edgeRepository) GetAll() []domain.Edge {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]domain.Edge, 0, len(r.data))
	for _, e := range r.data {
		result = append(result, e)
	}
	return result
}
