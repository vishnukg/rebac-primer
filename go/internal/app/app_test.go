package app

import (
	"context"
	"testing"
)

func TestConfigFromEnv_UsesDefaultPort(t *testing.T) {
	// Arrange
	lookup := func(string) string { return "" }

	// Act
	cfg, err := ConfigFromEnv(lookup)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 4001 {
		t.Fatalf("got port %d, want 4001", cfg.Port)
	}
}

func TestConfigFromEnv_UsesConfiguredPort(t *testing.T) {
	// Arrange
	lookup := func(key string) string {
		if key == "PORT" {
			return "4999"
		}
		return ""
	}

	// Act
	cfg, err := ConfigFromEnv(lookup)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 4999 {
		t.Fatalf("got port %d, want 4999", cfg.Port)
	}
}

func TestConfigFromEnv_RejectsInvalidPort(t *testing.T) {
	// Arrange
	lookup := func(key string) string {
		if key == "PORT" {
			return "not-a-port"
		}
		return ""
	}

	// Act
	_, err := ConfigFromEnv(lookup)

	// Assert
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNewWithConfig_UsesConfiguredPort(t *testing.T) {
	// Act
	a, err := NewWithConfig(context.Background(), Config{Port: 4999})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Port != 4999 {
		t.Fatalf("got port %d, want 4999", a.Port)
	}
	if a.Handler == nil {
		t.Fatal("expected handler")
	}
}
