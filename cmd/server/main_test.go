package main

import (
	"testing"
)

func TestReadPort_UsesDefaultPort(t *testing.T) {
	// Arrange
	t.Setenv("PORT", "")

	// Act
	port, err := readPort()

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if port != 4001 {
		t.Fatalf("got port %d, want 4001", port)
	}
}

func TestReadPort_UsesConfiguredPort(t *testing.T) {
	// Arrange
	t.Setenv("PORT", "4999")

	// Act
	port, err := readPort()

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if port != 4999 {
		t.Fatalf("got port %d, want 4999", port)
	}
}

func TestReadPort_RejectsInvalidPort(t *testing.T) {
	// Arrange
	t.Setenv("PORT", "not-a-port")

	// Act
	_, err := readPort()

	// Assert
	if err == nil {
		t.Fatal("expected error")
	}
}
