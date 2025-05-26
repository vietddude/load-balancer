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
		algo Algorithm
	}{
		{"round-robin", RoundRobin},
		{"least-connections", LeastConnections},
		{"random", Random},
		{"weighted-round-robin", WeightedRoundRobin},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := New(string(tt.algo))
			if b.algorithm != tt.algo {
				t.Errorf("Expected algorithm %v, got %v", tt.algo, b.algorithm)
			}
		})
	}
}

func TestAddRemoveBackend(t *testing.T) {
	b := New("round-robin")
	backend := backend.New("test", "http://localhost:8080", 1)

	// Test adding backend
	b.AddBackend("test", backend)
	if len(b.backends) != 1 {
		t.Errorf("Expected 1 backend, got %d", len(b.backends))
	}

	// Test removing backend
	b.RemoveBackend("test")
	if len(b.backends) != 0 {
		t.Errorf("Expected 0 backends, got %d", len(b.backends))
	}
}

func TestGetBackendNoBackends(t *testing.T) {
	b := New("round-robin")
	_, err := b.GetBackend()
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
		backend, err := b.GetBackend()
		if err != nil {
			t.Fatalf("GetBackend failed: %v", err)
		}
		seen[backend.URL]++
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
		if b, exists := b.backends[id]; exists {
			b.IncrementConnections()
		}
	}

	// Get backend 100 times
	for range make([]struct{}, 100) {
		backend, err := b.GetBackend()
		if err != nil {
			t.Fatalf("GetBackend failed: %v", err)
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
		backend := backend.New(id, fmt.Sprintf("http://localhost:808%d", len(b.backends)+1), weight)
		b.AddBackend(id, backend)
	}

	// Test weighted distribution
	seen := make(map[string]int)
	numRequests := 600
	for range make([]struct{}, numRequests) {
		backend, err := b.GetBackend()
		if err != nil {
			t.Fatalf("GetBackend failed: %v", err)
		}
		seen[backend.URL]++
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
		for id, b := range b.backends {
			if b.URL == url {
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
