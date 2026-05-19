package azure

import "testing"

func TestNormalizeBaseURL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "resource endpoint",
			raw:  "https://example.openai.azure.com",
			want: "https://example.openai.azure.com/openai/v1",
		},
		{
			name: "resource endpoint with slash",
			raw:  "https://example.openai.azure.com/",
			want: "https://example.openai.azure.com/openai/v1",
		},
		{
			name: "openai path",
			raw:  "https://example.openai.azure.com/openai",
			want: "https://example.openai.azure.com/openai/v1",
		},
		{
			name: "v1 base url",
			raw:  "https://example.openai.azure.com/openai/v1/",
			want: "https://example.openai.azure.com/openai/v1",
		},
		{
			name: "proxy path",
			raw:  "https://example.openai.azure.com/proxy/azure/?api-version=2025-04-01-preview",
			want: "https://example.openai.azure.com/proxy/azure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeBaseURL(tt.raw); got != tt.want {
				t.Fatalf("normalizeBaseURL(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}
