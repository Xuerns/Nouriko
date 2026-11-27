package main

import (
	"testing"
)

func TestPortFromEnv(t *testing.T) {
	t.Setenv("PORT", "")
	if got := portFromEnv(); got != defaultPort {
		t.Fatalf("expected default port %s, got %s", defaultPort, got)
	}

	t.Setenv("PORT", "9090")
	if got := portFromEnv(); got != "9090" {
		t.Fatalf("expected port 9090, got %s", got)
	}
}
