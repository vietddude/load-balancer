package metrics

import (
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics tracks various load balancer metrics
type Metrics struct {
	mu sync.RWMutex

	// Prometheus metrics
	totalRequests       atomic.Int64
	failedRequests      atomic.Int64
	activeConnections   map[string]int64
	backendRequests     map[string]int64
	backendFailures     map[string]int64
	backendLatencies    map[string]int64
	healthCheckFailures map[string]int64
}

// New creates a new Metrics instance
func New() *Metrics {
	return &Metrics{
		activeConnections:   make(map[string]int64),
		backendRequests:     make(map[string]int64),
		backendFailures:     make(map[string]int64),
		backendLatencies:    make(map[string]int64),
		healthCheckFailures: make(map[string]int64),
	}
}

// IncrementTotalRequests increments the total request counter
func (m *Metrics) IncrementTotalRequests() {
	m.totalRequests.Add(1)
}

// IncrementFailedRequests increments the failed request counter
func (m *Metrics) IncrementFailedRequests() {
	m.failedRequests.Add(1)
}

// IncrementActiveConnections increments the active connections for a backend
func (m *Metrics) IncrementActiveConnections(backendID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeConnections[backendID]++
}

// DecrementActiveConnections decrements the active connections for a backend
func (m *Metrics) DecrementActiveConnections(backendID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.activeConnections[backendID] > 0 {
		m.activeConnections[backendID]--
	}
}

// IncrementBackendRequests increments the request counter for a backend
func (m *Metrics) IncrementBackendRequests(backendID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.backendRequests[backendID]++
}

// IncrementBackendFailures increments the failure counter for a backend
func (m *Metrics) IncrementBackendFailures(backendID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.backendFailures[backendID]++
}

// RecordBackendLatency records the latency for a backend
func (m *Metrics) RecordBackendLatency(backendID string, latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.backendLatencies[backendID] = latency.Microseconds()
}

// IncrementHealthCheckFailures increments the health check failure counter for a backend
func (m *Metrics) IncrementHealthCheckFailures(backendID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healthCheckFailures[backendID]++
}

// GetStats returns the current metrics
func (m *Metrics) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"total_requests":        m.totalRequests.Load(),
		"failed_requests":       m.failedRequests.Load(),
		"active_connections":    m.activeConnections,
		"backend_requests":      m.backendRequests,
		"backend_failures":      m.backendFailures,
		"backend_latencies":     m.backendLatencies,
		"health_check_failures": m.healthCheckFailures,
	}
}

// GetPrometheusMetrics returns metrics in Prometheus format
func (m *Metrics) GetPrometheusMetrics() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var metrics string

	// Total requests
	metrics += "# HELP load_balancer_total_requests Total number of requests processed\n"
	metrics += "# TYPE load_balancer_total_requests counter\n"
	metrics += "load_balancer_total_requests " + strconv.FormatInt(m.totalRequests.Load(), 10) + "\n"

	// Failed requests
	metrics += "# HELP load_balancer_failed_requests Total number of failed requests\n"
	metrics += "# TYPE load_balancer_failed_requests counter\n"
	metrics += "load_balancer_failed_requests " + strconv.FormatInt(m.failedRequests.Load(), 10) + "\n"

	// Active connections
	metrics += "# HELP load_balancer_active_connections Number of active connections per backend\n"
	metrics += "# TYPE load_balancer_active_connections gauge\n"
	for backend, count := range m.activeConnections {
		metrics += "load_balancer_active_connections{backend=\"" + backend + "\"} " + strconv.FormatInt(count, 10) + "\n"
	}

	// Backend requests
	metrics += "# HELP load_balancer_backend_requests Number of requests per backend\n"
	metrics += "# TYPE load_balancer_backend_requests counter\n"
	for backend, count := range m.backendRequests {
		metrics += "load_balancer_backend_requests{backend=\"" + backend + "\"} " + strconv.FormatInt(count, 10) + "\n"
	}

	// Backend failures
	metrics += "# HELP load_balancer_backend_failures Number of failures per backend\n"
	metrics += "# TYPE load_balancer_backend_failures counter\n"
	for backend, count := range m.backendFailures {
		metrics += "load_balancer_backend_failures{backend=\"" + backend + "\"} " + strconv.FormatInt(count, 10) + "\n"
	}

	// Backend latencies
	metrics += "# HELP load_balancer_backend_latency_microseconds Latency per backend in microseconds\n"
	metrics += "# TYPE load_balancer_backend_latency_microseconds gauge\n"
	for backend, latency := range m.backendLatencies {
		metrics += "load_balancer_backend_latency_microseconds{backend=\"" + backend + "\"} " + strconv.FormatInt(latency, 10) + "\n"
	}

	// Health check failures
	metrics += "# HELP load_balancer_health_check_failures Number of health check failures per backend\n"
	metrics += "# TYPE load_balancer_health_check_failures counter\n"
	for backend, count := range m.healthCheckFailures {
		metrics += "load_balancer_health_check_failures{backend=\"" + backend + "\"} " + strconv.FormatInt(count, 10) + "\n"
	}

	return metrics
}
