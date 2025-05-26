package proxy

import (
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"load-balancer/internal/backend"
	"load-balancer/internal/balancer"
	"load-balancer/internal/metrics"
	"load-balancer/internal/retry"
	"load-balancer/internal/session"
)

// Proxy represents a load balancer proxy
type Proxy struct {
	balancer balancer.Balancer
	metrics  *metrics.Metrics
	session  *session.Manager
	client   *http.Client
}

// New creates a new proxy
func New(m *metrics.Metrics) *Proxy {
	return &Proxy{
		metrics: m,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetBalancer sets the load balancer
func (p *Proxy) SetBalancer(b balancer.Balancer) {
	p.balancer = b
}

// SetSessionManager sets the session manager
func (p *Proxy) SetSessionManager(s *session.Manager) {
	p.session = s
}

// ServeHTTP implements the http.Handler interface
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Handle metrics request
	if r.URL.Path == "/metrics" {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		w.Write([]byte(p.metrics.GetPrometheusMetrics()))
		return
	}

	// Increment total requests
	p.metrics.IncrementTotalRequests()

	// Check for existing session
	var backendID string
	if p.session != nil {
		backendID = p.session.GetBackendID(r)
	}

	// Get backend from balancer
	var backend *backend.Backend
	var err error
	if backendID != "" {
		backend, err = p.balancer.GetBackend(backendID)
		if err != nil {
			backend = nil
		}
	}
	if backend == nil {
		backend, err = p.balancer.Next()
		if err != nil {
			p.metrics.IncrementFailedRequests()
			http.Error(w, "No available backends", http.StatusServiceUnavailable)
			return
		}
	}

	// Increment backend requests
	p.metrics.IncrementBackendRequests(backend.ID())
	log.Printf("Incoming request %s to backend %s", r.URL.Path, backend.ID())
	// Forward request to backend
	err = p.forwardRequest(w, r, backend)
	if err != nil {
		p.metrics.IncrementBackendFailures(backend.ID())
		switch err {
		case ErrBackendUnavailable:
			http.Error(w, "Backend unavailable", http.StatusServiceUnavailable)
		case ErrBackendError:
			http.Error(w, "Backend error", http.StatusBadGateway)
		default:
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Set session if enabled
	if p.session != nil {
		p.session.SetBackendID(r, w, backend.ID())
	}
}

// forwardRequest forwards a request to a backend
func (p *Proxy) forwardRequest(w http.ResponseWriter, r *http.Request, b *backend.Backend) error {
	// Increment active connections
	p.metrics.IncrementActiveConnections(b.ID())
	defer p.metrics.DecrementActiveConnections(b.ID())

	// Create request to backend
	req, err := http.NewRequest(r.Method, b.URL().String()+r.URL.Path, r.Body)
	if err != nil {
		return err
	}

	// Copy headers
	for k, v := range r.Header {
		req.Header[k] = v
	}

	// Set host header
	req.Host = r.Host

	// Create retry config
	retryConfig := b.GetRetryConfig()

	// Execute request with retries
	var resp *http.Response
	err = retry.Do(r.Context(), retryConfig, func() error {
		var err error
		resp, err = p.client.Do(req)
		if err != nil {
			return err
		}

		// Check if response indicates failure
		if resp.StatusCode >= 500 {
			return errors.New("backend returned error status code")
		}

		return nil
	})

	if err != nil {
		// Record failure in circuit breaker
		b.GetCircuitBreaker().RecordFailure()
		return err
	}

	// Record success in circuit breaker
	b.GetCircuitBreaker().RecordSuccess()

	// Copy response headers
	for k, v := range resp.Header {
		w.Header()[k] = v
	}

	// Set status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	if _, err := io.Copy(w, resp.Body); err != nil {
		return err
	}

	return nil
}

// ErrBackendUnavailable is returned when the backend is not available
var ErrBackendUnavailable = &proxyError{"backend unavailable"}

// ErrBackendError is returned when the backend returns an error
var ErrBackendError = &proxyError{"backend error"}

type proxyError struct {
	msg string
}

func (e *proxyError) Error() string {
	return e.msg
}
