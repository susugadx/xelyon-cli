package gathercontext

import (
	"strings"
	"testing"
)

func TestFormatWithRouteHint_PreservesErrorPrefix(t *testing.T) {
	got := formatWithRouteHint("Direct query", "Error: direct path not found: ./missing.go")
	if !strings.HasPrefix(strings.TrimSpace(got), "Error:") {
		t.Fatalf("expected formatted direct error to preserve Error prefix, got:\n%s", got)
	}
	if !strings.Contains(got, "Route: Direct query") {
		t.Fatalf("expected formatted direct error to keep route hint, got:\n%s", got)
	}
}

func TestFormatWithRouteHint_KeepsRouteFirstForSuccessfulBodies(t *testing.T) {
	got := formatWithRouteHint("Direct read", "body")
	if !strings.HasPrefix(strings.TrimSpace(got), "Route: Direct read") {
		t.Fatalf("expected successful output to keep route hint first, got:\n%s", got)
	}
}
