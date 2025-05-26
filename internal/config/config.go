package config

import (
	"encoding/json"
	"os"
	"time"
)

// Duration is a custom type for time.Duration that supports JSON unmarshaling
type Duration time.Duration

// UnmarshalJSON implements custom JSON unmarshaling for Duration
func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	duration, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(duration)
	return nil
}

// MarshalJSON implements custom JSON marshaling for Duration
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// Config represents the load balancer configuration
type Config struct {
	Server struct {
		Port     int    `json:"port"`
		CertFile string `json:"cert_file"`
		KeyFile  string `json:"key_file"`
	} `json:"server"`

	// Load balancer configuration
	Algorithm string `json:"algorithm"`

	// Health check configuration
	HealthCheck struct {
		Interval Duration `json:"interval"`
		Timeout  Duration `json:"timeout"`
		Path     string   `json:"path"`
	} `json:"health_check"`

	// Circuit breaker configuration
	CircuitBreaker struct {
		FailureThreshold int      `json:"failure_threshold"`
		ResetTimeout     Duration `json:"reset_timeout"`
		HalfOpenLimit    int      `json:"half_open_limit"`
	} `json:"circuit_breaker"`

	// Retry configuration
	Retry struct {
		MaxRetries      int      `json:"max_retries"`
		InitialInterval Duration `json:"initial_interval"`
		MaxInterval     Duration `json:"max_interval"`
		Multiplier      float64  `json:"multiplier"`
		Randomization   float64  `json:"randomization"`
	} `json:"retry"`

	// Backend configuration
	Backends []BackendConfig `json:"backends"`
}

// BackendConfig represents a backend configuration
type BackendConfig struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	Weight int    `json:"weight"`
}

// Load loads the configuration from a file
func Load(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return nil, err
	}

	// Set default values if not specified
	if config.Server.Port == 0 {
		config.Server.Port = 8080
	}

	if config.HealthCheck.Interval == 0 {
		config.HealthCheck.Interval = Duration(5 * time.Second)
	}

	if config.HealthCheck.Timeout == 0 {
		config.HealthCheck.Timeout = Duration(2 * time.Second)
	}

	if config.HealthCheck.Path == "" {
		config.HealthCheck.Path = "/health"
	}

	if config.Algorithm == "" {
		config.Algorithm = "round-robin"
	}

	return &config, nil
}

// Save saves the configuration to a file
func (c *Config) Save(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(c)
}
