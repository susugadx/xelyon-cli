package api

import (
	"testing"
)

func TestNewOllamaProvider(t *testing.T) {
	baseURL := "http://localhost:11434"
	provider := NewOllamaProvider(baseURL)

	if provider == nil {
		t.Fatal("NewOllamaProvider() returned nil")
	}
}

func TestOllamaProvider_Name(t *testing.T) {
	provider := NewOllamaProvider("http://localhost:11434")

	name := provider.Name()
	if name != "Ollama" {
		t.Errorf("Name() = %v, want 'Ollama'", name)
	}
}

func TestOllamaProvider_SupportsImages(t *testing.T) {
	provider := NewOllamaProvider("http://localhost:11434")

	supports := provider.SupportsImages()
	if supports {
		t.Error("SupportsImages() = true, want false (Ollama does not support images)")
	}
}
