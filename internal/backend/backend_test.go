package backend

import (
	"testing"
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

// func TestBackendAvailability(t *testing.T) {
// 	backend := New("test", "http://localhost:8080", 1)
// 	if backend == nil {
// 		t.Fatal("Failed to create backend")
// 	}

// 	// Test initial state
// 	if !backend.IsAvailable() {
// 		t.Error("New backend should be available")
// 	}

// 	// Test circuit breaker
// 	for i := 0; i < 5; i++ { // Assuming failure threshold is 5
// 		backend.GetCircuitBreaker().RecordFailure()
// 	}
// 	if backend.IsAvailable() {
// 		t.Error("Backend should not be available after reaching failure threshold")
// 	}

// 	// Wait for reset timeout to transition to half-open state
// 	time.Sleep(31 * time.Second) // ResetTimeout is 30s

// 	// Test circuit breaker recovery
// 	// Need multiple successes to transition from HalfOpen to Closed
// 	for i := 0; i < 3; i++ { // HalfOpenLimit is 3
// 		backend.GetCircuitBreaker().RecordSuccess()
// 	}
// 	if !backend.IsAvailable() {
// 		t.Error("Backend should be available after circuit breaker recovery")
// 	}
// }

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
