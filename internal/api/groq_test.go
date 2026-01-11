package api

import (
	"testing"
)

func TestNewGroqProvider(t *testing.T) {
	apiKey := "test-api-key"
	provider := NewGroqProvider(apiKey)

	if provider == nil {
		t.Fatal("NewGroqProvider() returned nil")
	}
}

func TestGroqProvider_Name(t *testing.T) {
	provider := NewGroqProvider("test-key")

	name := provider.Name()
	if name != "Groq" {
		t.Errorf("Name() = %v, want 'Groq'", name)
	}
}

func TestGroqProvider_SupportsImages(t *testing.T) {
	provider := NewGroqProvider("test-key")

	supports := provider.SupportsImages()
	if supports {
		t.Error("SupportsImages() = true, want false (Groq does not support images)")
	}
}
