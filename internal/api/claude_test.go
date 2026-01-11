package api

import (
	"testing"
)

func TestNewClaudeProvider(t *testing.T) {
	apiKey := "test-api-key"
	provider := NewClaudeProvider(apiKey)

	if provider == nil {
		t.Fatal("NewClaudeProvider() returned nil")
	}
}

func TestClaudeProvider_Name(t *testing.T) {
	provider := NewClaudeProvider("test-key")

	name := provider.Name()
	if name != "Claude" {
		t.Errorf("Name() = %v, want 'Claude'", name)
	}
}

func TestClaudeProvider_SupportsImages(t *testing.T) {
	provider := NewClaudeProvider("test-key")

	supports := provider.SupportsImages()
	if !supports {
		t.Error("SupportsImages() = false, want true (Claude supports images)")
	}
}
