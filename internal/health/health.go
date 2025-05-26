package health

import (
	"context"
	"log"
	"net/http"
	"time"

	"load-balancer/internal/backend"
)

// CheckType represents the type of health check to perform
type CheckType string

const (
	// HTTPCheck performs HTTP GET request to check health
	HTTPCheck CheckType = "http"
	// TCPCheck performs TCP connection check
	TCPCheck CheckType = "tcp"
)

// Config holds the configuration for a health check
type Config struct {
	Type     CheckType
	Endpoint string
	Interval time.Duration
	Timeout  time.Duration
	// HTTP specific
	Method     string
	Path       string
	StatusCode int
	// TCP specific
	Port int
}

// Result represents the result of a health check
type Result struct {
	BackendID string
	Success   bool
	Error     error
	Timestamp time.Time
	Latency   time.Duration
}

// Checker defines the interface for health checks
type Checker interface {
	// Check performs a single health check
	Check(ctx context.Context) Result
	// Type returns the type of health check
	Type() CheckType
}

// HTTPChecker implements the Checker interface for HTTP health checks
type HTTPChecker struct {
	config    Config
	client    *http.Client
	BackendID string
}

// NewHTTPChecker creates a new HTTP health checker
func NewHTTPChecker(endpoint string, config Config) Checker {
	return &HTTPChecker{
		config: Config{
			Type:       HTTPCheck,
			Endpoint:   endpoint,
			Interval:   config.Interval,
			Timeout:    config.Timeout,
			Method:     "GET",
			Path:       config.Path,
			StatusCode: http.StatusOK,
		},
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Check performs an HTTP health check
func (c *HTTPChecker) Check(ctx context.Context) Result {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, c.config.Method, c.config.Endpoint+c.config.Path, nil)
	if err != nil {
		return Result{
			BackendID: c.BackendID,
			Success:   false,
			Error:     err,
			Timestamp: time.Now(),
			Latency:   time.Since(start),
		}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return Result{
			BackendID: c.BackendID,
			Success:   false,
			Error:     err,
			Timestamp: time.Now(),
			Latency:   time.Since(start),
		}
	}
	defer resp.Body.Close()

	success := resp.StatusCode == c.config.StatusCode
	return Result{
		BackendID: c.BackendID,
		Success:   success,
		Error:     nil,
		Timestamp: time.Now(),
		Latency:   time.Since(start),
	}
}

// Type returns the type of health check
func (c *HTTPChecker) Type() CheckType {
	return c.config.Type
}

// Scheduler manages health checks for multiple backends
type Scheduler struct {
	checkers map[string]Checker
	results  chan Result
	stop     chan struct{}
	backends map[string]*backend.Backend
}

// NewScheduler creates a new health check scheduler
func NewScheduler() *Scheduler {
	return &Scheduler{
		checkers: make(map[string]Checker),
		results:  make(chan Result, 100),
		stop:     make(chan struct{}),
		backends: make(map[string]*backend.Backend),
	}
}

// AddBackend adds a backend to be monitored
func (s *Scheduler) AddBackend(backendID string, b *backend.Backend, checker Checker) {
	s.backends[backendID] = b
	s.checkers[backendID] = checker
}

// RemoveBackend removes a backend from monitoring
func (s *Scheduler) RemoveBackend(backendID string) {
	delete(s.backends, backendID)
	delete(s.checkers, backendID)
}

// Start begins the health check scheduling
func (s *Scheduler) Start() {
	for backendID, checker := range s.checkers {
		go s.runChecker(backendID, checker)
	}
}

// Stop stops all health checks
func (s *Scheduler) Stop() {
	close(s.stop)
}

// Results returns the channel for health check results
func (s *Scheduler) Results() <-chan Result {
	return s.results
}

func (s *Scheduler) runChecker(backendID string, checker Checker) {
	ticker := time.NewTicker(time.Second * 10) // Default interval, should be configurable
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			result := checker.Check(context.Background())
			s.results <- result

			// Update backend health status
			if b, exists := s.backends[backendID]; exists {
				log.Printf("Backend %s health check result: %v", backendID, result)
				b.SetHealth(result.Success)
			}
		case <-s.stop:
			return
		}
	}
}
