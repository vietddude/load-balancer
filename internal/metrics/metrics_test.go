package metrics

import (
	"testing"
)

func TestMetrics(t *testing.T) {
	m := New()

	// Backend1: Moderate usage
	m.IncrementTotalRequests()
	m.IncrementFailedRequests()
	m.IncrementBackendRequests("backend1")
	m.IncrementBackendRequests("backend1")
	m.IncrementBackendFailures("backend1")
	m.IncrementHealthCheckFailures("backend1")
	m.IncrementActiveConnections("backend1")
	m.DecrementActiveConnections("backend1")

	// Backend2: Heavier usage
	for i := 0; i < 5; i++ {
		m.IncrementTotalRequests()
		m.IncrementBackendRequests("backend2")
	}
	m.IncrementFailedRequests()
	m.IncrementBackendFailures("backend2")
	m.IncrementBackendFailures("backend2")
	m.IncrementHealthCheckFailures("backend2")
	m.IncrementHealthCheckFailures("backend2")
	m.IncrementActiveConnections("backend2")
	m.IncrementActiveConnections("backend2")
	m.DecrementActiveConnections("backend2")

	// Backend3: Light usage, no failures
	m.IncrementTotalRequests()
	m.IncrementBackendRequests("backend3")
	m.IncrementActiveConnections("backend3")

	// Get stats
	stats := m.GetStats()

	t.Run("Global Stats", func(t *testing.T) {
		if got := stats["total_requests"].(int64); got != 7 {
			t.Errorf("Expected 7 total requests, got %d", got)
		}
		if got := stats["failed_requests"].(int64); got != 2 {
			t.Errorf("Expected 2 failed requests, got %d", got)
		}
	})

	t.Run("Backend Requests", func(t *testing.T) {
		reqs := stats["backend_requests"].(map[string]int64)
		expect := map[string]int64{
			"backend1": 2,
			"backend2": 5,
			"backend3": 1,
		}
		for backend, expected := range expect {
			if got := reqs[backend]; got != expected {
				t.Errorf("Expected %d requests for %s, got %d", expected, backend, got)
			}
		}
	})

	t.Run("Backend Failures", func(t *testing.T) {
		failures := stats["backend_failures"].(map[string]int64)
		expect := map[string]int64{
			"backend1": 1,
			"backend2": 2,
			"backend3": 0,
		}
		for backend, expected := range expect {
			if got := failures[backend]; got != expected {
				t.Errorf("Expected %d failures for %s, got %d", expected, backend, got)
			}
		}
	})

	t.Run("Health Check Failures", func(t *testing.T) {
		hcf := stats["health_check_failures"].(map[string]int64)
		expect := map[string]int64{
			"backend1": 1,
			"backend2": 2,
			"backend3": 0,
		}
		for backend, expected := range expect {
			if got := hcf[backend]; got != expected {
				t.Errorf("Expected %d health check failures for %s, got %d", expected, backend, got)
			}
		}
	})

	t.Run("Active Connections", func(t *testing.T) {
		conns := stats["active_connections"].(map[string]int64)
		expect := map[string]int64{
			"backend1": 0, // increment then decrement
			"backend2": 1, // increment twice, decrement once
			"backend3": 1, // single increment
		}
		for backend, expected := range expect {
			if got := conns[backend]; got != expected {
				t.Errorf("Expected %d active connections for %s, got %d", expected, backend, got)
			}
		}
	})
}
