package circuitbreaker

import (
	"sync"
	"time"
)

// State represents the state of the circuit breaker
type State int

const (
	// Closed means the circuit is closed and requests are allowed
	Closed State = iota
	// Open means the circuit is open and requests are blocked
	Open
	// HalfOpen means the circuit is half-open and limited requests are allowed
	HalfOpen
)

// Config represents the circuit breaker configuration
type Config struct {
	FailureThreshold int
	ResetTimeout     time.Duration
	HalfOpenLimit    int
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	config          Config
	state           State
	failureCount    int
	successCount    int
	lastFailureTime time.Time
	mu              sync.RWMutex
}

// New creates a new circuit breaker
func New(config Config) *CircuitBreaker {
	return &CircuitBreaker{
		config: config,
		state:  Closed,
	}
}

// SetConfig updates the circuit breaker configuration
func (cb *CircuitBreaker) SetConfig(config Config) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.config = config
}

// AllowRequest checks if a request should be allowed
func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case Closed:
		return true
	case Open:
		// Check if we should transition to half-open
		if time.Since(cb.lastFailureTime) > cb.config.ResetTimeout {
			cb.mu.RUnlock()
			cb.mu.Lock()
			cb.state = HalfOpen
			cb.mu.Unlock()
			cb.mu.RLock()
			return true
		}
		return false
	case HalfOpen:
		// Allow limited requests in half-open state
		return cb.successCount < cb.config.HalfOpenLimit
	default:
		return false
	}
}

// RecordSuccess records a successful request
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case Closed:
		// Reset failure count on success
		cb.failureCount = 0
	case HalfOpen:
		cb.successCount++
		// If we've had enough successes, close the circuit
		if cb.successCount >= cb.config.HalfOpenLimit {
			cb.state = Closed
			cb.failureCount = 0
			cb.successCount = 0
		}
	}
}

// RecordFailure records a failed request
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case Closed:
		// If we've exceeded the failure threshold, open the circuit
		if cb.failureCount >= cb.config.FailureThreshold {
			cb.state = Open
		}
	case HalfOpen:
		// Any failure in half-open state opens the circuit
		cb.state = Open
		cb.successCount = 0
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetFailureCount returns the current failure count
func (cb *CircuitBreaker) GetFailureCount() int {
	return cb.failureCount
}

// GetLastFailure returns the time of the last failure
func (cb *CircuitBreaker) GetLastFailure() time.Time {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.lastFailureTime
}

// GetLastSuccess returns the time of the last success
func (cb *CircuitBreaker) GetLastSuccess() time.Time {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.lastFailureTime
}
