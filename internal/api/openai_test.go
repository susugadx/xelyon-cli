package api

import (
	"testing"
)

func TestNewOpenAIProvider(t *testing.T) {
	apiKey := "test-api-key"
	provider := NewOpenAIProvider(apiKey)

	if provider == nil {
		t.Fatal("NewOpenAIProvider() returned nil")
	}
}

func TestOpenAIProvider_Name(t *testing.T) {
	provider := NewOpenAIProvider("test-key")

	name := provider.Name()
	if name != "OpenAI" {
		t.Errorf("Name() = %v, want 'OpenAI'", name)
	}
}

func TestOpenAIProvider_SupportsImages(t *testing.T) {
	provider := NewOpenAIProvider("test-key")

	supports := provider.SupportsImages()
	if !supports {
		t.Error("SupportsImages() = false, want true (OpenAI supports images)")
	}
}
