package config

import (
	"testing"
)

func TestLoad(t *testing.T) {
	cfg, err := Load("./../../config.json")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", cfg.Server.Port)
	}
}
