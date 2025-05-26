package balancer

import (
	"errors"

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
	ErrBackendNotFound   = errors.New("backend not found")
	ErrNoHealthyBackends = errors.New("no healthy backends available")
	ErrUnknownAlgorithm  = errors.New("unknown algorithm")
)

// Balancer represents a load balancer
type Balancer interface {
	// Next returns the next backend to use
	Next() (*backend.Backend, error)
	// GetBackend returns a specific backend by ID
	GetBackend(id string) (*backend.Backend, error)
	// AddBackend adds a backend to the balancer
	AddBackend(id string, backend *backend.Backend)
	// RemoveBackend removes a backend from the balancer
	RemoveBackend(id string)
}

// New creates a new balancer with the specified algorithm
func New(algorithm string) Balancer {
	switch algorithm {
	case "round-robin":
		return newRoundRobin()
	case "least-connections":
		return newLeastConnections()
	case "weighted-round-robin":
		return newWeightedRoundRobin()
	default:
		return newRoundRobin()
	}
}
