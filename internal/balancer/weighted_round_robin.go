package balancer

import (
	"sync"

	"load-balancer/internal/backend"
)

// weightedRoundRobin implements the weighted round-robin load balancing algorithm
type weightedRoundRobin struct {
	backends map[string]*backend.Backend
	mu       sync.RWMutex
	current  int
	weights  []int
	keys     []string
	// Track the current weight for each backend
	currentWeights []int
	// Track the maximum weight
	maxWeight int
}

// newWeightedRoundRobin creates a new weighted round-robin balancer
func newWeightedRoundRobin() *weightedRoundRobin {
	return &weightedRoundRobin{
		backends:       make(map[string]*backend.Backend),
		weights:        make([]int, 0),
		keys:           make([]string, 0),
		currentWeights: make([]int, 0),
		maxWeight:      0,
	}
}

// Next returns the next backend based on weights
func (wrr *weightedRoundRobin) Next() (*backend.Backend, error) {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()

	if len(wrr.backends) == 0 {
		return nil, ErrNoBackends
	}

	var (
		totalWeight int
		selectedIdx int = -1
		maxWeight   int = -1
	)

	for i, b := range wrr.keys {
		if !wrr.backends[b].IsAvailable() {
			continue
		}

		// Increase current weight
		wrr.currentWeights[i] += wrr.weights[i]
		totalWeight += wrr.weights[i]

		// Pick the backend with highest current weight
		if selectedIdx == -1 || wrr.currentWeights[i] > maxWeight {
			selectedIdx = i
			maxWeight = wrr.currentWeights[i]
		}
	}

	if selectedIdx == -1 {
		return nil, ErrNoHealthyBackends
	}

	// Adjust the selected backend's current weight
	wrr.currentWeights[selectedIdx] -= totalWeight
	return wrr.backends[wrr.keys[selectedIdx]], nil
}

// GetBackend returns a specific backend by ID
func (wrr *weightedRoundRobin) GetBackend(id string) (*backend.Backend, error) {
	wrr.mu.RLock()
	defer wrr.mu.RUnlock()

	backend, exists := wrr.backends[id]
	if !exists {
		return nil, ErrBackendNotFound
	}

	return backend, nil
}

// AddBackend adds a backend to the balancer
func (wrr *weightedRoundRobin) AddBackend(id string, backend *backend.Backend) {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()

	weight := backend.Weight()
	if weight > wrr.maxWeight {
		wrr.maxWeight = weight
	}

	wrr.backends[id] = backend
	wrr.keys = append(wrr.keys, id)
	wrr.weights = append(wrr.weights, weight)
	wrr.currentWeights = append(wrr.currentWeights, 0)
}

// RemoveBackend removes a backend from the balancer
func (wrr *weightedRoundRobin) RemoveBackend(id string) {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()

	delete(wrr.backends, id)

	// Remove from keys, weights, and currentWeights slices
	for i, key := range wrr.keys {
		if key == id {
			wrr.keys = append(wrr.keys[:i], wrr.keys[i+1:]...)
			wrr.weights = append(wrr.weights[:i], wrr.weights[i+1:]...)
			wrr.currentWeights = append(wrr.currentWeights[:i], wrr.currentWeights[i+1:]...)
			break
		}
	}

	// Recalculate maxWeight
	wrr.maxWeight = 0
	for _, weight := range wrr.weights {
		if weight > wrr.maxWeight {
			wrr.maxWeight = weight
		}
	}

	// Adjust current index if needed
	if wrr.current >= len(wrr.keys) {
		wrr.current = 0
	}
}
