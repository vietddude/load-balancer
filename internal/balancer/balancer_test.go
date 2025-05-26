package balancer

import (
	"fmt"
	"math"
	"testing"

	"load-balancer/internal/backend"
)

func TestNewBalancer(t *testing.T) {
	tests := []struct {
		name string
		algo string
	}{
		{"round-robin", "round-robin"},
		{"least-connections", "least-connections"},
		{"weighted-round-robin", "weighted-round-robin"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := New(tt.algo)
			if b == nil {
				t.Errorf("Expected non-nil balancer for algorithm %s", tt.algo)
			}
		})
	}
}

func TestAddRemoveBackend(t *testing.T) {
	b := New("round-robin")
	backend := backend.New("test", "http://localhost:8080", 1)

	// Test adding backend
	b.AddBackend("test", backend)
	got, err := b.GetBackend("test")
	if err != nil {
		t.Errorf("Expected no error after adding backend, got %v", err)
	}
	if got == nil {
		t.Error("Expected backend to be found after adding")
	}

	// Test removing backend
	b.RemoveBackend("test")
	_, err = b.GetBackend("test")
	if err == nil {
		t.Error("Expected error after removing backend")
	}
}

func TestGetBackendNoBackends(t *testing.T) {
	b := New("round-robin")
	_, err := b.Next()
	if err != ErrNoBackends {
		t.Errorf("Expected ErrNoBackends, got %v", err)
	}
}

func TestRoundRobin(t *testing.T) {
	b := New("round-robin")

	// Add three backends with different URLs
	for i := 1; i <= 3; i++ {
		backend := backend.New(fmt.Sprintf("backend%d", i), fmt.Sprintf("http://localhost:808%d", i), 1)
		b.AddBackend(fmt.Sprintf("backend%d", i), backend)
	}

	// Test round-robin distribution with more requests
	seen := make(map[string]int)
	numRequests := 1000
	for range make([]struct{}, numRequests) {
		backend, err := b.Next()
		if err != nil {
			t.Fatalf("Next failed: %v", err)
		}
		seen[backend.URL().String()]++
	}

	// Check distribution - allow for 20% deviation from expected value
	expectedCount := float64(numRequests) / float64(len(seen))
	for url, count := range seen {
		deviation := math.Abs(float64(count)-expectedCount) / expectedCount
		if deviation > 0.2 {
			t.Errorf("Uneven distribution for %s: got %d requests, expected around %d (±20%%)",
				url, count, int(expectedCount))
		}
	}
}

func TestLeastConnections(t *testing.T) {
	b := New("least-connections")

	// Add three backends with different URLs
	for i := 1; i <= 3; i++ {
		backend := backend.New(fmt.Sprintf("backend%d", i), fmt.Sprintf("http://localhost:808%d", i), 1)
		b.AddBackend(fmt.Sprintf("backend%d", i), backend)
	}

	// Increment connections on first two backends
	for _, id := range []string{"backend1", "backend2"} {
		if backend, err := b.GetBackend(id); err == nil {
			backend.IncrementConnections()
		}
	}

	// Get backend 100 times
	for range make([]struct{}, 100) {
		backend, err := b.Next()
		if err != nil {
			t.Fatalf("Next failed: %v", err)
		}
		if backend.GetActiveConnections() != 0 {
			t.Errorf("Expected backend with 0 connections, got %d", backend.GetActiveConnections())
		}
	}
}

func TestWeightedRoundRobin(t *testing.T) {
	b := New("weighted-round-robin")

	// Add backends with different weights
	weights := map[string]int{
		"backend1": 1,
		"backend2": 2,
		"backend3": 3,
	}

	for id, weight := range weights {
		backend := backend.New(id, fmt.Sprintf("http://localhost:808%d", weight), weight)
		b.AddBackend(id, backend)
	}

	// Test weighted distribution
	seen := make(map[string]int)
	numRequests := 600
	for range make([]struct{}, numRequests) {
		backend, err := b.Next()
		if err != nil {
			t.Fatalf("Next failed: %v", err)
		}
		seen[backend.URL().String()]++
	}

	// Check distribution (should be roughly proportional to weights)
	total := 0
	for _, count := range seen {
		total += count
	}

	// Calculate expected distribution based on weights
	totalWeight := 0
	for _, weight := range weights {
		totalWeight += weight
	}

	for url, count := range seen {
		// Find the backend ID for this URL
		var backendID string
		for id, weight := range weights {
			expectedURL := fmt.Sprintf("http://localhost:808%d", weight)
			if url == expectedURL {
				backendID = id
				break
			}
		}

		expected := float64(weights[backendID]) / float64(totalWeight) * float64(total)
		deviation := math.Abs(float64(count)-expected) / expected
		if deviation > 0.2 { // Allow 20% deviation
			t.Errorf("Uneven distribution for %s: got %d requests, expected around %d (±20%%)",
				url, count, int(expected))
		}
	}
}
