package api

import (
	"net/http"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestNewBaseProvider_ConfiguresResponseHeaderTimeout(t *testing.T) {
	provider := NewBaseProvider("test", "key", "https://example.com", "")

	if provider.HTTPClient == nil {
		t.Fatal("HTTPClient should not be nil")
	}

	if provider.HTTPClient.Timeout != config.DefaultHTTPTimeout {
		t.Fatalf("HTTPClient.Timeout = %v, want %v", provider.HTTPClient.Timeout, config.DefaultHTTPTimeout)
	}

	transport, ok := provider.HTTPClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("HTTPClient.Transport type = %T, want *http.Transport", provider.HTTPClient.Transport)
	}

	if transport.ResponseHeaderTimeout != defaultResponseHeaderTimeout {
		t.Fatalf("ResponseHeaderTimeout = %v, want %v", transport.ResponseHeaderTimeout, defaultResponseHeaderTimeout)
	}
}
