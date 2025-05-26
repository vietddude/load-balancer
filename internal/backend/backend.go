package backend

import (
	"sync"
	"sync/atomic"
	"time"

	"load-balancer/internal/circuitbreaker"
	"load-balancer/internal/retry"
	"net/url"
)

// Backend represents a backend server
type Backend struct {
	id             string
	url            *url.URL
	weight         int
	IsHealthy      bool
	CurrentConns   int32
	mu             sync.RWMutex
	circuitBreaker *circuitbreaker.CircuitBreaker
	retryConfig    *retry.Config
}

// New creates a new backend
func New(id string, urlStr string, weight int) *Backend {
	parsedURL, _ := url.Parse(urlStr)
	return &Backend{
		id:           id,
		url:          parsedURL,
		weight:       weight,
		IsHealthy:    true,
		CurrentConns: 0,
		circuitBreaker: circuitbreaker.New(circuitbreaker.Config{
			FailureThreshold: 5,
			ResetTimeout:     30 * time.Second,
			HalfOpenLimit:    3,
		}),
	}
}

// ID returns the backend ID
func (b *Backend) ID() string {
	return b.id
}

// URL returns the backend URL
func (b *Backend) URL() *url.URL {
	return b.url
}

// Weight returns the backend weight
func (b *Backend) Weight() int {
	return b.weight
}

// SetWeight sets the backend weight
func (b *Backend) SetWeight(weight int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.weight = weight
}

// SetHealth sets the health status of the backend
func (b *Backend) SetHealth(healthy bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.IsHealthy = healthy
}

// IsAvailable checks if the backend is available for requests
func (b *Backend) IsAvailable() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.IsHealthy && b.circuitBreaker.AllowRequest()
}

// IncrementConnections increments the number of active connections
func (b *Backend) IncrementConnections() {
	atomic.AddInt32(&b.CurrentConns, 1)
}

// DecrementConnections decrements the number of active connections
func (b *Backend) DecrementConnections() {
	if b.GetActiveConnections() == 0 {
		return
	}
	atomic.AddInt32(&b.CurrentConns, -1)
}

// GetActiveConnections returns the number of active connections
func (b *Backend) GetActiveConnections() int {
	return int(atomic.LoadInt32(&b.CurrentConns))
}

// GetWeight returns the weight of the backend
func (b *Backend) GetWeight() int {
	return b.weight
}

// SetRetryConfig sets the retry configuration
func (b *Backend) SetRetryConfig(config *retry.Config) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.retryConfig = config
}

// GetRetryConfig returns the retry configuration
func (b *Backend) GetRetryConfig() *retry.Config {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.retryConfig
}

// GetCircuitBreaker returns the circuit breaker
func (b *Backend) GetCircuitBreaker() *circuitbreaker.CircuitBreaker {
	return b.circuitBreaker
}
