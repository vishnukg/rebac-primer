package main

import (
	"context"
	"testing"
)

func TestReadPort_UsesDefaultPort(t *testing.T) {
	t.Setenv("PORT", "")

	port, err := readPort()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if port != 4001 {
		t.Fatalf("got port %d, want 4001", port)
	}
}

func TestReadPort_UsesConfiguredPort(t *testing.T) {
	t.Setenv("PORT", "4999")

	port, err := readPort()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if port != 4999 {
		t.Fatalf("got port %d, want 4999", port)
	}
}

func TestReadPort_RejectsInvalidPort(t *testing.T) {
	t.Setenv("PORT", "not-a-port")

	_, err := readPort()

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBuildHandler_ReturnsHandler(t *testing.T) {
	handler, err := buildHandler(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
}
