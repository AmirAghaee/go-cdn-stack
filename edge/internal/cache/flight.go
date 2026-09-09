package cache

import (
	"sync"
)

type flight struct {
	done chan struct{}
}

type flightGroup struct {
	mu      sync.Mutex
	flights map[string]*flight
}

func (g *flightGroup) join(key string) (*flight, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if existing := g.flights[key]; existing != nil {
		return existing, false
	}
	if g.flights == nil {
		g.flights = make(map[string]*flight)
	}
	created := &flight{done: make(chan struct{})}
	g.flights[key] = created
	return created, true
}

func (g *flightGroup) finish(key string, completed *flight) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.flights[key] != completed {
		return
	}
	delete(g.flights, key)
	close(completed.done)
}
