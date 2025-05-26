package tls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// Config holds the TLS configuration
type Config struct {
	CertFile       string
	KeyFile        string
	ReloadInterval time.Duration
	MinVersion     uint16
	MaxVersion     uint16
	CipherSuites   []uint16
}

// Manager handles TLS certificate management and dynamic reloading
type Manager struct {
	config   Config
	cert     *tls.Certificate
	mu       sync.RWMutex
	stopChan chan struct{}
	lastMod  time.Time
	onReload func(*tls.Certificate)
}

// NewManager creates a new TLS certificate manager
func NewManager(config Config) (*Manager, error) {
	if config.ReloadInterval == 0 {
		config.ReloadInterval = 5 * time.Minute
	}
	if config.MinVersion == 0 {
		config.MinVersion = tls.VersionTLS12
	}
	if config.MaxVersion == 0 {
		config.MaxVersion = tls.VersionTLS13
	}
	if len(config.CipherSuites) == 0 {
		config.CipherSuites = []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		}
	}

	manager := &Manager{
		config:   config,
		stopChan: make(chan struct{}),
	}

	// Load initial certificate
	if err := manager.loadCertificate(); err != nil {
		return nil, fmt.Errorf("failed to load initial certificate: %v", err)
	}

	// Start certificate reloader
	go manager.reloadLoop()

	return manager, nil
}

// GetCertificate returns the current TLS certificate
func (m *Manager) GetCertificate() *tls.Certificate {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cert
}

// GetTLSConfig returns a TLS configuration with the current certificate
func (m *Manager) GetTLSConfig() *tls.Config {
	return &tls.Config{
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			return m.GetCertificate(), nil
		},
		MinVersion:   m.config.MinVersion,
		MaxVersion:   m.config.MaxVersion,
		CipherSuites: m.config.CipherSuites,
	}
}

// SetReloadCallback sets a callback function to be called when the certificate is reloaded
func (m *Manager) SetReloadCallback(callback func(*tls.Certificate)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onReload = callback
}

// Stop stops the certificate reloader
func (m *Manager) Stop() {
	close(m.stopChan)
}

// loadCertificate loads the certificate from files
func (m *Manager) loadCertificate() error {
	cert, err := tls.LoadX509KeyPair(m.config.CertFile, m.config.KeyFile)
	if err != nil {
		return fmt.Errorf("failed to load certificate: %v", err)
	}

	// Parse certificate to get expiration
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %v", err)
	}

	// Check if certificate is expired
	if time.Now().After(x509Cert.NotAfter) {
		return fmt.Errorf("certificate is expired")
	}

	m.mu.Lock()
	m.cert = &cert
	m.lastMod = time.Now()
	m.mu.Unlock()

	// Call reload callback if set
	if m.onReload != nil {
		m.onReload(&cert)
	}

	return nil
}

// reloadLoop periodically checks for certificate updates
func (m *Manager) reloadLoop() {
	ticker := time.NewTicker(m.config.ReloadInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check if certificate files have been modified
			certInfo, err := os.Stat(m.config.CertFile)
			if err != nil {
				log.Printf("Failed to stat certificate file: %v", err)
				continue
			}

			keyInfo, err := os.Stat(m.config.KeyFile)
			if err != nil {
				log.Printf("Failed to stat key file: %v", err)
				continue
			}

			// If either file has been modified, reload the certificate
			if certInfo.ModTime().After(m.lastMod) || keyInfo.ModTime().After(m.lastMod) {
				if err := m.loadCertificate(); err != nil {
					log.Printf("Failed to reload certificate: %v", err)
				} else {
					log.Printf("Certificate reloaded successfully")
				}
			}
		case <-m.stopChan:
			return
		}
	}
}
