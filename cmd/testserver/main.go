package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var (
	port = flag.Int("port", 8081, "Port to listen on")
)

func main() {
	flag.Parse()

	// Create HTTP server
	server := &http.Server{
		Addr: fmt.Sprintf(":%d", *port),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/health":
				log.Printf("%d /health", *port)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			default:
				log.Printf("%d /", *port)
				w.WriteHeader(http.StatusOK)
				w.Write(fmt.Appendf(nil, "Server running on port %d", *port))
			}
		}),
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting test server on port %d", *port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down test server...")
}
