package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"load-balancer/internal/backend"
)

func TestHTTPChecker(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create backend
	b := backend.New("test-backend", server.URL, 1)

	// Create health checker
	checker := NewHTTPChecker(server.URL, Config{
		Timeout:  1 * time.Second,
		Path:     "/",
		Interval: 5 * time.Second,
	})

	// Set backend ID
	if hc, ok := checker.(*HTTPChecker); ok {
		hc.BackendID = b.ID()
	}

	// Perform health check
	result := checker.Check(context.Background())

	// Verify result
	if !result.Success {
		t.Errorf("Expected successful health check, got failure: %v", result.Error)
	}

	if result.BackendID != b.ID() {
		t.Errorf("Expected backend ID '%s', got '%s'", b.ID(), result.BackendID)
	}
}

func TestScheduler(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create a backend
	b := backend.New("test-backend", server.URL, 1)

	// Create health checker with test server URL
	checker := NewHTTPChecker(server.URL, Config{
		Timeout:  1 * time.Second,
		Path:     "/",
		Interval: 5 * time.Second,
	})

	// Set backend ID
	if hc, ok := checker.(*HTTPChecker); ok {
		hc.BackendID = b.ID()
	}

	scheduler := NewScheduler(5 * time.Second)

	// Add backend and checker
	scheduler.AddBackend(b.ID(), b, checker)

	// Start scheduler
	scheduler.Start()

	// Wait for a result
	select {
	case result := <-scheduler.Results():
		if !result.Success {
			t.Errorf("Expected successful health check, got failure: %v", result.Error)
		}
		// Verify backend health status was updated
		if !b.IsHealthy {
			t.Error("Expected backend to be healthy after successful check")
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for health check result")
	}

	// Test removing backend
	scheduler.RemoveBackend(b.ID())

	// Stop the scheduler to prevent any more health checks
	scheduler.Stop()

	// Wait for any pending health checks to complete
	time.Sleep(100 * time.Millisecond)

	// Drain any remaining results
	for {
		select {
		case <-scheduler.Results():
			// Drain results
		default:
			// No more results
			goto done
		}
	}
done:

	// Should not receive any more results for this backend
	select {
	case <-scheduler.Results():
		t.Error("Received result for removed backend")
	case <-time.After(time.Second):
		// Expected timeout
	}
}

func TestSchedulerStop(t *testing.T) {
	scheduler := NewScheduler(5 * time.Second)

	// Create a backend
	b := backend.New("test-backend", "http://slow-server", 1)

	// Create a checker that takes a long time
	config := Config{
		Type:     HTTPCheck,
		Endpoint: "http://slow-server",
		Timeout:  time.Hour,
	}
	checker := NewHTTPChecker("test-backend", config)

	scheduler.AddBackend("test-backend", b, checker)
	scheduler.Start()

	// Stop the scheduler
	scheduler.Stop()

	// Wait a bit to ensure all goroutines are stopped
	time.Sleep(100 * time.Millisecond)

	// Try to add a new backend after stop
	scheduler.AddBackend("new-backend", b, checker)

	// Should not receive any results
	select {
	case <-scheduler.Results():
		t.Error("Received result after scheduler stop")
	case <-time.After(time.Second):
		// Expected timeout
	}
}
