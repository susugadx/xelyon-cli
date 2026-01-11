package api

import (
	"testing"
)

func TestNewGeminiProvider(t *testing.T) {
	apiKey := "test-api-key"
	provider := NewGeminiProvider(apiKey)

	if provider == nil {
		t.Fatal("NewGeminiProvider() returned nil")
	}
}

func TestGeminiProvider_Name(t *testing.T) {
	provider := NewGeminiProvider("test-key")

	name := provider.Name()
	if name != "Gemini" {
		t.Errorf("Name() = %v, want 'Gemini'", name)
	}
}

func TestGeminiProvider_SupportsImages(t *testing.T) {
	provider := NewGeminiProvider("test-key")

	supports := provider.SupportsImages()
	if !supports {
		t.Error("SupportsImages() = false, want true (Gemini supports images)")
	}
}
