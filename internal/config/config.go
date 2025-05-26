package config

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"load-balancer/internal/session"
	tlsmanager "load-balancer/pkg/tls"
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

// TLSConfig represents TLS configuration
type TLSConfig struct {
	Enabled        bool     `json:"enabled"`
	CertFile       string   `json:"cert_file"`
	KeyFile        string   `json:"key_file"`
	ReloadInterval Duration `json:"reload_interval"`
	MinVersion     string   `json:"min_version"`
	MaxVersion     string   `json:"max_version"`
	CipherSuites   []string `json:"cipher_suites"`
}

// Config represents the load balancer configuration
type Config struct {
	Server struct {
		Port int       `json:"port"`
		TLS  TLSConfig `json:"tls"`
	} `json:"server"`

	// Load balancer configuration
	Algorithm string `json:"algorithm"`

	// Sticky session configuration
	StickySession struct {
		Enabled         bool     `json:"enabled"`
		Type            string   `json:"type"`
		CookieName      string   `json:"cookie_name"`
		TTL             Duration `json:"ttl"`
		MaxSessions     int      `json:"max_sessions"`
		CleanupInterval Duration `json:"cleanup_interval"`
	} `json:"sticky_session"`

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

	// Set default TLS configuration
	if config.Server.TLS.ReloadInterval == 0 {
		config.Server.TLS.ReloadInterval = Duration(5 * time.Minute)
	}

	if config.Server.TLS.MinVersion == "" {
		config.Server.TLS.MinVersion = "TLS12"
	}

	if config.Server.TLS.MaxVersion == "" {
		config.Server.TLS.MaxVersion = "TLS13"
	}

	// Set default sticky session configuration
	if config.StickySession.CookieName == "" {
		config.StickySession.CookieName = "lb_session"
	}
	if config.StickySession.TTL == 0 {
		config.StickySession.TTL = Duration(24 * time.Hour)
	}
	if config.StickySession.MaxSessions == 0 {
		config.StickySession.MaxSessions = 10000
	}
	if config.StickySession.CleanupInterval == 0 {
		config.StickySession.CleanupInterval = Duration(1 * time.Hour)
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

// GetTLSConfig converts the TLS configuration to a tls.Config
func (c *Config) GetTLSConfig() (*tlsmanager.Config, error) {
	if !c.Server.TLS.Enabled {
		return nil, nil
	}

	// Convert TLS version strings to constants
	minVersion, err := parseTLSVersion(c.Server.TLS.MinVersion)
	if err != nil {
		return nil, err
	}

	maxVersion, err := parseTLSVersion(c.Server.TLS.MaxVersion)
	if err != nil {
		return nil, err
	}

	// Convert cipher suite strings to constants
	cipherSuites, err := parseCipherSuites(c.Server.TLS.CipherSuites)
	if err != nil {
		return nil, err
	}

	return &tlsmanager.Config{
		CertFile:       c.Server.TLS.CertFile,
		KeyFile:        c.Server.TLS.KeyFile,
		ReloadInterval: time.Duration(c.Server.TLS.ReloadInterval),
		MinVersion:     minVersion,
		MaxVersion:     maxVersion,
		CipherSuites:   cipherSuites,
	}, nil
}

// GetSessionConfig converts the sticky session configuration to a session.Config
func (c *Config) GetSessionConfig() session.Config {
	return session.Config{
		Enabled:         c.StickySession.Enabled,
		Type:            session.Type(c.StickySession.Type),
		CookieName:      c.StickySession.CookieName,
		TTL:             time.Duration(c.StickySession.TTL),
		MaxSessions:     c.StickySession.MaxSessions,
		CleanupInterval: time.Duration(c.StickySession.CleanupInterval),
	}
}

// parseTLSVersion converts a TLS version string to a constant
func parseTLSVersion(version string) (uint16, error) {
	switch version {
	case "TLS10":
		return tls.VersionTLS10, nil
	case "TLS11":
		return tls.VersionTLS11, nil
	case "TLS12":
		return tls.VersionTLS12, nil
	case "TLS13":
		return tls.VersionTLS13, nil
	default:
		return 0, fmt.Errorf("unsupported TLS version: %s", version)
	}
}

// parseCipherSuites converts cipher suite strings to constants
func parseCipherSuites(suites []string) ([]uint16, error) {
	if len(suites) == 0 {
		return nil, nil
	}

	result := make([]uint16, len(suites))
	for i, suite := range suites {
		switch suite {
		case "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384":
			result[i] = tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
		case "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":
			result[i] = tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
		case "TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305":
			result[i] = tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305
		case "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305":
			result[i] = tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305
		case "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256":
			result[i] = tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256
		case "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":
			result[i] = tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
		default:
			return nil, fmt.Errorf("unsupported cipher suite: %s", suite)
		}
	}
	return result, nil
}
