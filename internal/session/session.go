package session

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Type represents the type of sticky session
type Type string

const (
	// IPBased uses client IP for sticky sessions
	IPBased Type = "ip"
	// CookieBased uses cookies for sticky sessions
	CookieBased Type = "cookie"
)

// Config holds the sticky session configuration
type Config struct {
	Enabled         bool          `json:"enabled"`
	Type            Type          `json:"type"`
	CookieName      string        `json:"cookie_name"`
	TTL             time.Duration `json:"ttl"`
	MaxSessions     int           `json:"max_sessions"`
	CleanupInterval time.Duration `json:"cleanup_interval"`
}

// Session represents a sticky session
type Session struct {
	BackendID string
	ExpiresAt time.Time
}

// Manager handles sticky session management
type Manager struct {
	config   Config
	sessions map[string]*Session
	mu       sync.RWMutex
	stopChan chan struct{}
}

// NewManager creates a new session manager
func NewManager(config Config) *Manager {
	if config.CookieName == "" {
		config.CookieName = "lb_session"
	}
	if config.TTL == 0 {
		config.TTL = 24 * time.Hour
	}
	if config.MaxSessions == 0 {
		config.MaxSessions = 10000
	}
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 1 * time.Hour
	}

	manager := &Manager{
		config:   config,
		sessions: make(map[string]*Session),
		stopChan: make(chan struct{}),
	}

	// Start cleanup routine
	go manager.cleanupLoop()

	return manager
}

// GetBackendID returns the backend ID for a request based on sticky session configuration
func (m *Manager) GetBackendID(r *http.Request) string {
	if !m.config.Enabled {
		return ""
	}

	var sessionKey string
	switch m.config.Type {
	case IPBased:
		sessionKey = m.getIPKey(r)
	case CookieBased:
		sessionKey = m.getCookieKey(r)
	default:
		return ""
	}

	if sessionKey == "" {
		return ""
	}

	m.mu.RLock()
	session, exists := m.sessions[sessionKey]
	m.mu.RUnlock()

	if !exists || time.Now().After(session.ExpiresAt) {
		return ""
	}

	return session.BackendID
}

// SetBackendID sets the backend ID for a request
func (m *Manager) SetBackendID(r *http.Request, w http.ResponseWriter, backendID string) {
	if !m.config.Enabled || backendID == "" {
		return
	}

	var sessionKey string
	switch m.config.Type {
	case IPBased:
		sessionKey = m.getIPKey(r)
	case CookieBased:
		sessionKey = m.getCookieKey(r)
		if sessionKey == "" {
			sessionKey = m.generateSessionKey()
			http.SetCookie(w, &http.Cookie{
				Name:     m.config.CookieName,
				Value:    sessionKey,
				Path:     "/",
				HttpOnly: true,
				Secure:   r.TLS != nil,
				SameSite: http.SameSiteLaxMode,
				Expires:  time.Now().Add(m.config.TTL),
			})
		}
	}

	if sessionKey == "" {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if we need to evict old sessions
	if len(m.sessions) >= m.config.MaxSessions {
		m.evictOldestSession()
	}

	m.sessions[sessionKey] = &Session{
		BackendID: backendID,
		ExpiresAt: time.Now().Add(m.config.TTL),
	}
}

// Stop stops the session manager
func (m *Manager) Stop() {
	close(m.stopChan)
}

// getIPKey returns a session key based on the client IP
func (m *Manager) getIPKey(r *http.Request) string {
	// Get IP from X-Forwarded-For header if available
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.RemoteAddr
	}
	return ip
}

// getCookieKey returns a session key from the cookie
func (m *Manager) getCookieKey(r *http.Request) string {
	cookie, err := r.Cookie(m.config.CookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// generateSessionKey generates a new session key
func (m *Manager) generateSessionKey() string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	return hex.EncodeToString(hash[:])
}

// evictOldestSession removes the oldest session
func (m *Manager) evictOldestSession() {
	var oldestKey string
	var oldestTime time.Time

	for key, session := range m.sessions {
		if oldestKey == "" || session.ExpiresAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = session.ExpiresAt
		}
	}

	if oldestKey != "" {
		delete(m.sessions, oldestKey)
	}
}

// cleanupLoop periodically removes expired sessions
func (m *Manager) cleanupLoop() {
	ticker := time.NewTicker(m.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.cleanup()
		case <-m.stopChan:
			return
		}
	}
}

// cleanup removes expired sessions
func (m *Manager) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for key, session := range m.sessions {
		if now.After(session.ExpiresAt) {
			delete(m.sessions, key)
		}
	}
}
