package proxy

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"load-balancer/internal/backend"
	"load-balancer/internal/balancer"
	"load-balancer/internal/metrics"
	"load-balancer/internal/retry"
	"load-balancer/internal/session"
)

func TestProxy(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Echo back the request path
		w.Write([]byte(r.URL.Path))
	}))
	defer server.Close()

	// Create a backend pointing to our test server
	b := backend.New("test-backend", server.URL, 1)
	b.SetRetryConfig(&retry.Config{
		MaxRetries:      3,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     time.Second,
		Multiplier:      2,
		Randomization:   0.1,
	})

	// Create balancer and add backend
	bal := balancer.New("round-robin")
	bal.AddBackend("test-backend", b)

	sessionConfig := session.Config{
		Enabled:         true,
		Type:            session.IPBased,
		CookieName:      "session",
		TTL:             10 * time.Second,
		MaxSessions:     100,
		CleanupInterval: 10 * time.Second,
	}
	// Create a session manager
	sessionManager := session.NewManager(sessionConfig)

	// Create proxy and set balancer
	proxy := New(metrics.New())
	proxy.SetBalancer(bal)
	proxy.SetSessionManager(sessionManager)

	fmt.Println(proxy)
	// Test cases
	tests := []struct {
		name          string
		path          string
		expectedBody  string
		expectedError bool
	}{
		{
			name:          "root path",
			path:          "/",
			expectedBody:  "/",
			expectedError: false,
		},
		{
			name:          "test path",
			path:          "/test",
			expectedBody:  "/test",
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test request
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			// Forward request using the HTTP handler interface
			proxy.ServeHTTP(w, req)

			// Check response
			if w.Code != http.StatusOK {
				t.Errorf("ServeHTTP() status = %v, want %v", w.Code, http.StatusOK)
			}

			// Check body
			if got := w.Body.String(); got != tt.expectedBody {
				t.Errorf("ServeHTTP() body = %v, want %v", got, tt.expectedBody)
			}
		})
	}
}
