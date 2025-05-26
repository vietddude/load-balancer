package balancer

import (
	"errors"
	"math/rand"
	"sync"

	"load-balancer/internal/backend"
)

// Algorithm represents the load balancing algorithm type
type Algorithm string

const (
	// RoundRobin distributes requests in a circular order
	RoundRobin Algorithm = "round-robin"
	// LeastConnections sends requests to the backend with the fewest active connections
	LeastConnections Algorithm = "least-connections"
	// Random distributes requests randomly
	Random Algorithm = "random"
	// WeightedRoundRobin distributes requests based on backend weights
	WeightedRoundRobin Algorithm = "weighted-round-robin"
)

var (
	ErrNoBackends        = errors.New("no backends available")
	ErrNoHealthyBackends = errors.New("no healthy backends available")
	ErrUnknownAlgorithm  = errors.New("unknown algorithm")
)

// Balancer represents a load balancer
type Balancer struct {
	algorithm Algorithm
	backends  map[string]*backend.Backend
	mu        sync.RWMutex
	current   int
}

// New creates a new load balancer with the specified algorithm
func New(algorithm string) *Balancer {
	return &Balancer{
		algorithm: Algorithm(algorithm),
		backends:  make(map[string]*backend.Backend),
		current:   0,
	}
}

// AddBackend adds a backend to the balancer
func (b *Balancer) AddBackend(id string, backend *backend.Backend) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.backends[id] = backend
}

// RemoveBackend removes a backend from the balancer
func (b *Balancer) RemoveBackend(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.backends, id)
}

// GetBackend returns the next backend based on the selected algorithm
func (b *Balancer) GetBackend() (*backend.Backend, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if len(b.backends) == 0 {
		return nil, ErrNoBackends
	}

	var availableBackends []*backend.Backend
	for _, backend := range b.backends {
		if backend.IsAvailable() {
			availableBackends = append(availableBackends, backend)
		}
	}

	if len(availableBackends) == 0 {
		return nil, ErrNoHealthyBackends
	}

	switch b.algorithm {
	case RoundRobin:
		return b.roundRobin(availableBackends)
	case LeastConnections:
		return b.leastConnections(availableBackends)
	case Random:
		return b.random(availableBackends)
	case WeightedRoundRobin:
		return b.weightedRoundRobin(availableBackends)
	default:
		return nil, ErrUnknownAlgorithm
	}
}

func (b *Balancer) roundRobin(backends []*backend.Backend) (*backend.Backend, error) {
	if len(backends) == 0 {
		return nil, ErrNoBackends
	}

	backend := backends[b.current]
	b.current = (b.current + 1) % len(backends)
	return backend, nil
}

func (b *Balancer) leastConnections(backends []*backend.Backend) (*backend.Backend, error) {
	if len(backends) == 0 {
		return nil, ErrNoBackends
	}

	var selected *backend.Backend
	minConns := -1

	for _, backend := range backends {
		conns := backend.GetActiveConnections()
		if minConns == -1 || conns < minConns {
			minConns = conns
			selected = backend
		}
	}

	return selected, nil
}

func (b *Balancer) random(backends []*backend.Backend) (*backend.Backend, error) {
	if len(backends) == 0 {
		return nil, ErrNoBackends
	}

	return backends[rand.Intn(len(backends))], nil
}

func (b *Balancer) weightedRoundRobin(backends []*backend.Backend) (*backend.Backend, error) {
	if len(backends) == 0 {
		return nil, ErrNoBackends
	}

	var totalWeight int
	for _, backend := range backends {
		totalWeight += backend.GetWeight()
	}

	if totalWeight == 0 {
		return nil, errors.New("all backends have zero weight")
	}

	r := rand.Intn(totalWeight)

	var currentWeight int
	for _, backend := range backends {
		currentWeight += backend.GetWeight()
		if r < currentWeight {
			return backend, nil
		}
	}

	return backends[0], nil
}
