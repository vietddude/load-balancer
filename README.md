# Load Balancer

[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/vietddude/load-balancer)
[![Go Report Card](https://goreportcard.com/badge/github.com/vietddude/load-balancer)](https://goreportcard.com/report/github.com/vietddude/load-balancer)
![CI](https://github.com/vietddude/load-balancer/actions/workflows/go.yml/badge.svg)
![GitHub last commit](https://img.shields.io/github/last-commit/vietddude/load-balancer)

A high-performance load balancer written in Go with support for:

- Multiple load balancing algorithms (Round Robin, Least Connections, Random, Weighted Round Robin)
- Health checks with configurable intervals and failure thresholds
- Circuit breaking with configurable failure thresholds and reset timeouts
- Sticky sessions based on IP or cookies
- Metrics collection and Prometheus integration
- TLS termination with dynamic certificate loading
- Dynamic backend management with hot reloading
- Docker support with Prometheus and Grafana monitoring

## DeepWiki

For detailed documentation, visit our [DeepWiki](https://deepwiki.com/vietddude/load-balancer).

## Project Structure

```
load-balancer/
├── cmd/
│   └── loadbalancer/
│       └── main.go           # Entry point for the load balancer app
├── internal/
│   ├── backend/
│   │   ├── backend.go        # Backend struct, health checks, circuit breaker
│   │   └── backend_test.go
│   ├── balancer/
│   │   ├── balancer.go       # LoadBalancer struct, routing, sticky sessions, weighted balancing
│   │   └── balancer_test.go
│   ├── proxy/
│   │   ├── proxy.go          # Reverse proxy wrapper, retries, request forwarding
│   │   └── proxy_test.go
│   ├── health/
│   │   ├── health.go         # Health check implementation & scheduling
│   │   └── health_test.go
│   ├── metrics/
│   │   ├── metrics.go        # Metrics collection & exporting (Prometheus integration)
│   │   └── metrics_test.go
│   ├── session/
│   │   ├── session.go        # Sticky session logic (based on IP or cookie)
│   │   └── session_test.go
│   └── config/
│       ├── config.go         # Configuration loading & dynamic backend management
│       └── config_test.go
├── pkg/
│   └── tls/
│       └── tls.go            # TLS termination helper (cert loading, config)
├── scripts/
│   └── run_backends.sh       # Helper script to spin up dummy backend servers for testing
├── prometheus/
│   └── prometheus.yml        # Prometheus configuration
├── grafana/
│   └── provisioning/         # Grafana provisioning configuration
│       └── datasources/
│           └── prometheus.yml
├── docker-compose.yml        # Docker Compose configuration
├── Dockerfile               # Docker build configuration
├── go.mod
├── go.sum
└── README.md
```

## Features

### Load Balancing Algorithms

- **Round Robin**: Distributes requests evenly across backends
- **Least Connections**: Routes to the backend with the fewest active connections
- **Random**: Randomly selects a backend for each request
- **Weighted Round Robin**: Distributes requests based on backend weights

### Health Checks

- Configurable check intervals
- Custom health check endpoints
- Failure threshold configuration
- Automatic backend removal on repeated failures

### Circuit Breaking

- Configurable failure thresholds
- Automatic circuit opening on repeated failures
- Half-open state for gradual recovery
- Configurable reset timeouts

### Sticky Sessions

- IP-based session affinity
- Cookie-based session tracking
- Configurable session timeouts
- Fallback to normal load balancing when sessions expire

### Metrics & Monitoring

- Prometheus metrics integration
- Request counts and latencies
- Backend health status
- Circuit breaker states
- Active connections per backend
- Grafana dashboards for visualization
- Real-time monitoring and alerting

### TLS Support

- Basic TLS termination with certificate files
- Configuration via cert_file and key_file in config
- Note: Certificate files must be provided by the user
- Note: Dynamic certificate reloading and SNI support are planned features

### Dynamic Configuration

- Hot reloading of backend configuration
- Runtime backend addition/removal
- Weight adjustment without restart
- Health check parameter updates

### Docker Support

- Containerized deployment
- Multi-stage builds for smaller images
- Docker Compose for easy setup
- Integrated monitoring stack

## Getting Started

### Local Development

1. Clone the repository:

   ```bash
   git clone https://github.com/yourusername/load-balancer.git
   cd load-balancer
   ```

2. Install dependencies:

   ```bash
   go mod tidy
   ```

3. Build the application:

   ```bash
   go build ./cmd/loadbalancer
   ```

4. Start test backend servers:

   ```bash
   ./scripts/run_backends.sh
   ```

5. Run the load balancer:
   ```bash
   ./loadbalancer -config config.yaml
   ```

### Docker Deployment

1. Build and start all services:

   ```bash
   docker-compose up -d
   ```

2. Access the services:

   - Load Balancer: http://localhost:8080
   - Prometheus: http://localhost:9090
   - Grafana: http://localhost:3000 (login with admin/admin)

3. View metrics:
   - Prometheus metrics are available at http://localhost:8080/metrics
   - Grafana dashboards are pre-configured with Prometheus data source

## Configuration

The load balancer can be configured using a YAML file. Here's an example configuration:

```yaml
listen:
  port: 8080
  tls:
    enabled: true
    cert_file: "cert.pem"
    key_file: "key.pem"

algorithm: "round-robin"

backends:
  - id: "backend1"
    url: "http://localhost:8081"
    weight: 1
  - id: "backend2"
    url: "http://localhost:8082"
    weight: 2

health_check:
  interval: "30s"
  timeout: "5s"
  path: "/health"
  failure_threshold: 3

circuit_breaker:
  failure_threshold: 5
  reset_timeout: "30s"
  half_open_limit: 3

sticky_session:
  enabled: true
  type: "cookie"
  cookie_name: "session_id"
  timeout: "1h"

metrics:
  prometheus:
    enabled: true
    path: "/metrics"
```

## Monitoring

The project includes a complete monitoring stack:

### Prometheus

- Scrapes metrics every 15 seconds
- Stores metrics data persistently
- Accessible at http://localhost:9090

### Grafana

- Pre-configured with Prometheus data source
- Default credentials: admin/admin
- Accessible at http://localhost:3000
- Includes dashboards for:
  - Request rates and latencies
  - Backend health status
  - Circuit breaker states
  - Active connections
  - Error rates

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
