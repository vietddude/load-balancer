package balancer

import (
	"sync"

	"load-balancer/internal/backend"
)

// roundRobin implements the round-robin load balancing algorithm
type roundRobin struct {
	backends map[string]*backend.Backend
	mu       sync.RWMutex
	current  int
	keys     []string
}

// newRoundRobin creates a new round-robin balancer
func newRoundRobin() *roundRobin {
	return &roundRobin{
		backends: make(map[string]*backend.Backend),
		keys:     make([]string, 0),
	}
}

// Next returns the next backend to use
func (rb *roundRobin) Next() (*backend.Backend, error) {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if len(rb.backends) == 0 {
		return nil, ErrNoBackends
	}

	// Get the next backend
	backend := rb.backends[rb.keys[rb.current]]
	rb.current = (rb.current + 1) % len(rb.keys)

	return backend, nil
}

// GetBackend returns a specific backend by ID
func (rb *roundRobin) GetBackend(id string) (*backend.Backend, error) {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	backend, exists := rb.backends[id]
	if !exists {
		return nil, ErrBackendNotFound
	}

	return backend, nil
}

// AddBackend adds a backend to the balancer
func (rb *roundRobin) AddBackend(id string, backend *backend.Backend) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.backends[id] = backend
	rb.keys = append(rb.keys, id)
}

// RemoveBackend removes a backend from the balancer
func (rb *roundRobin) RemoveBackend(id string) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	delete(rb.backends, id)

	// Remove from keys slice
	for i, key := range rb.keys {
		if key == id {
			rb.keys = append(rb.keys[:i], rb.keys[i+1:]...)
			break
		}
	}

	// Adjust current index if needed
	if rb.current >= len(rb.keys) {
		rb.current = 0
	}
}
