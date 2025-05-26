package backend

import (
	"testing"
	"time"
)

func TestNewBackend(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		weight  int
		wantErr bool
	}{
		{
			name:    "valid url",
			url:     "http://localhost:8080",
			weight:  1,
			wantErr: false,
		},
		{
			name:    "invalid url",
			url:     "://invalid",
			weight:  1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := New(tt.name, tt.url, tt.weight)
			if tt.wantErr && backend != nil {
				t.Error("New() expected nil for invalid input")
			}
			if !tt.wantErr && backend == nil {
				t.Error("New() unexpected nil for valid input")
			}
		})
	}
}

func TestBackendAvailability(t *testing.T) {
	backend := New("test", "http://localhost:8080", 1)
	if backend == nil {
		t.Fatal("Failed to create backend")
	}

	// Test initial state
	if !backend.IsAvailable() {
		t.Error("New backend should be available")
	}

	// Test connection limits
	for range 100 { // Use a reasonable number for testing
		backend.IncrementConnections()
	}
	if backend.IsAvailable() {
		t.Error("Backend should not be available when max connections reached")
	}

	// Test circuit breaker
	backend.DecrementConnections() // Make it available again
	backend.GetCircuitBreaker().RecordFailure()
	if backend.IsAvailable() {
		t.Error("Backend should not be available after failure")
	}

	// Test circuit breaker reset
	time.Sleep(30*time.Second + time.Second) // Use the default reset timeout
	if !backend.IsAvailable() {
		t.Error("Backend should be available after reset timeout")
	}
}

func TestBackendConnectionTracking(t *testing.T) {
	backend := New("test", "http://localhost:8080", 1)
	if backend == nil {
		t.Fatal("Failed to create backend")
	}

	// Test connection increment
	backend.IncrementConnections()
	if backend.GetActiveConnections() != 1 {
		t.Errorf("Expected 1 connection, got %d", backend.GetActiveConnections())
	}

	// Test connection decrement
	backend.DecrementConnections()
	if backend.GetActiveConnections() != 0 {
		t.Errorf("Expected 0 connections, got %d", backend.GetActiveConnections())
	}

	// Test decrement below zero
	backend.DecrementConnections()
	if backend.GetActiveConnections() != 0 {
		t.Errorf("Expected 0 connections after decrement below zero, got %d", backend.GetActiveConnections())
	}
}
