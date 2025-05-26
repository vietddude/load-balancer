package balancer

import (
	"sync"

	"load-balancer/internal/backend"
)

// leastConnections implements the least connections load balancing algorithm
type leastConnections struct {
	backends map[string]*backend.Backend
	mu       sync.RWMutex
}

// newLeastConnections creates a new least connections balancer
func newLeastConnections() *leastConnections {
	return &leastConnections{
		backends: make(map[string]*backend.Backend),
	}
}

// Next returns the backend with the least active connections
func (lc *leastConnections) Next() (*backend.Backend, error) {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	if len(lc.backends) == 0 {
		return nil, ErrNoBackends
	}

	var selected *backend.Backend
	minConns := -1

	for _, backend := range lc.backends {
		if !backend.IsAvailable() {
			continue
		}
		conns := backend.GetActiveConnections()
		if minConns == -1 || conns < minConns {
			minConns = conns
			selected = backend
		}
	}

	if selected == nil {
		return nil, ErrNoHealthyBackends
	}

	return selected, nil
}

// GetBackend returns a specific backend by ID
func (lc *leastConnections) GetBackend(id string) (*backend.Backend, error) {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	backend, exists := lc.backends[id]
	if !exists {
		return nil, ErrBackendNotFound
	}

	return backend, nil
}

// AddBackend adds a backend to the balancer
func (lc *leastConnections) AddBackend(id string, backend *backend.Backend) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.backends[id] = backend
}

// RemoveBackend removes a backend from the balancer
func (lc *leastConnections) RemoveBackend(id string) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	delete(lc.backends, id)
}
