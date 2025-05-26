package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"load-balancer/internal/backend"
	"load-balancer/internal/balancer"
	"load-balancer/internal/circuitbreaker"
	"load-balancer/internal/config"
	"load-balancer/internal/health"
	"load-balancer/internal/metrics"
	"load-balancer/internal/proxy"
	"load-balancer/internal/retry"
	"load-balancer/pkg/tls"
)

func main() {
	// Parse command line flags
	configFile := flag.String("config", "config.json", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize metrics
	m := metrics.New()

	// Initialize balancer with configured algorithm
	b := balancer.New(cfg.Algorithm)

	// Initialize proxy
	p := proxy.New(m)
	p.SetBalancer(b)

	// Initialize health check scheduler
	scheduler := health.NewScheduler()

	// Add backends from configuration
	for _, backendCfg := range cfg.Backends {
		// Create backend with retry settings
		backend := backend.New(backendCfg.ID, backendCfg.URL, backendCfg.Weight)
		backend.SetRetryConfig(&retry.Config{
			MaxRetries:      cfg.Retry.MaxRetries,
			InitialInterval: time.Duration(cfg.Retry.InitialInterval),
			MaxInterval:     time.Duration(cfg.Retry.MaxInterval),
			Multiplier:      cfg.Retry.Multiplier,
			Randomization:   cfg.Retry.Randomization,
		})

		// Configure circuit breaker
		backend.GetCircuitBreaker().SetConfig(circuitbreaker.Config{
			FailureThreshold: cfg.CircuitBreaker.FailureThreshold,
			ResetTimeout:     time.Duration(cfg.CircuitBreaker.ResetTimeout),
			HalfOpenLimit:    cfg.CircuitBreaker.HalfOpenLimit,
		})

		// Create health checker
		checker := health.NewHTTPChecker(backendCfg.URL, health.Config{
			Timeout:  time.Duration(cfg.HealthCheck.Timeout),
			Path:     cfg.HealthCheck.Path,
			Interval: time.Duration(cfg.HealthCheck.Interval),
		})

		// Add backend to balancer and scheduler
		b.AddBackend(backendCfg.ID, backend)
		scheduler.AddBackend(backendCfg.ID, backend, checker)
	}

	// Start health checks
	scheduler.Start()

	// Create HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: p,
	}

	// Initialize TLS if enabled
	var tlsManager *tls.Manager
	if cfg.Server.TLS.Enabled {
		tlsConfig, err := cfg.GetTLSConfig()
		if err != nil {
			log.Fatalf("Failed to get TLS config: %v", err)
		}

		tlsManager, err = tls.NewManager(*tlsConfig)
		if err != nil {
			log.Fatalf("Failed to initialize TLS manager: %v", err)
		}

		// Set TLS config on server
		server.TLSConfig = tlsManager.GetTLSConfig()
	}

	// Start server in a goroutine
	go func() {
		if cfg.Server.TLS.Enabled {
			log.Printf("Starting server with TLS on port %d", cfg.Server.Port)
			if err := server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Failed to start server: %v", err)
			}
		} else {
			log.Printf("Starting server on port %d", cfg.Server.Port)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Failed to start server: %v", err)
			}
		}
	}()

	// Set up graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Stop health checks
	scheduler.Stop()

	// Stop TLS manager if it exists
	if tlsManager != nil {
		tlsManager.Stop()
	}

	// Shutdown server
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}
