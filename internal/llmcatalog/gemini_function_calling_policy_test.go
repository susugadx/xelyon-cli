package llmcatalog

import "testing"

func TestGeminiFunctionCallingPolicy(t *testing.T) {
	tests := []struct {
		name               string
		request            string
		catalog            string
		wantRequest        string
		wantCatalog        string
		wantPolicy         string
		wantKnown          bool
		wantEnabled        bool
		wantRequestSupport ModelCapabilitySupport
		wantCatalogSupport ModelCapabilitySupport
	}{
		{
			name:               "supported catalog alias",
			request:            "corp-flash",
			catalog:            "models/gemini-3.5-flash",
			wantRequest:        "corp-flash",
			wantCatalog:        "gemini-3.5-flash",
			wantPolicy:         "gemini-3.5-flash",
			wantKnown:          true,
			wantEnabled:        true,
			wantRequestSupport: ModelCapabilitySupport{},
			wantCatalogSupport: ModelCapabilitySupport{Known: true, Supported: true},
		},
		{
			name:               "unsupported catalog alias",
			request:            "corp-lite",
			catalog:            "models/gemini-2.0-flash-lite",
			wantRequest:        "corp-lite",
			wantCatalog:        "gemini-2.0-flash-lite",
			wantPolicy:         "gemini-2.0-flash-lite",
			wantKnown:          true,
			wantEnabled:        false,
			wantRequestSupport: ModelCapabilitySupport{},
			wantCatalogSupport: ModelCapabilitySupport{
				Known:       true,
				Supported:   false,
				Reason:      "Gemini 2.0 Flash-Lite is not in the Gemini function calling supported-model list",
				Replacement: "gemini-3.1-flash-lite",
			},
		},
		{
			name:        "unsupported request model cannot be bypassed by catalog model",
			request:     "models/gemini-2.0-flash-lite",
			catalog:     "gemini-3.5-flash",
			wantRequest: "models/gemini-2.0-flash-lite",
			wantCatalog: "gemini-3.5-flash",
			wantPolicy:  "gemini-2.0-flash-lite",
			wantKnown:   true,
			wantEnabled: false,
			wantRequestSupport: ModelCapabilitySupport{
				Known:       true,
				Supported:   false,
				Reason:      "Gemini 2.0 Flash-Lite is not in the Gemini function calling supported-model list",
				Replacement: "gemini-3.1-flash-lite",
			},
			wantCatalogSupport: ModelCapabilitySupport{Known: true, Supported: true},
		},
		{
			name:               "unknown alias remains optimistic",
			request:            "corp-gemini",
			wantRequest:        "corp-gemini",
			wantCatalog:        "corp-gemini",
			wantPolicy:         "corp-gemini",
			wantKnown:          false,
			wantEnabled:        true,
			wantRequestSupport: ModelCapabilitySupport{},
			wantCatalogSupport: ModelCapabilitySupport{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewGeminiFunctionCallingPolicy(tt.request, tt.catalog)
			if got.RequestModel() != tt.wantRequest {
				t.Fatalf("RequestModel() = %q, want %q", got.RequestModel(), tt.wantRequest)
			}
			if got.CatalogModel() != tt.wantCatalog {
				t.Fatalf("CatalogModel() = %q, want %q", got.CatalogModel(), tt.wantCatalog)
			}
			if got.PolicyModel() != tt.wantPolicy {
				t.Fatalf("PolicyModel() = %q, want %q", got.PolicyModel(), tt.wantPolicy)
			}
			if got.Support().Known != tt.wantKnown {
				t.Fatalf("Support().Known = %t, want %t", got.Support().Known, tt.wantKnown)
			}
			if got.Enabled() != tt.wantEnabled {
				t.Fatalf("Enabled() = %t, want %t", got.Enabled(), tt.wantEnabled)
			}
			if got.RequestSupport() != tt.wantRequestSupport {
				t.Fatalf("RequestSupport() = %#v, want %#v", got.RequestSupport(), tt.wantRequestSupport)
			}
			if got.CatalogSupport() != tt.wantCatalogSupport {
				t.Fatalf("CatalogSupport() = %#v, want %#v", got.CatalogSupport(), tt.wantCatalogSupport)
			}
		})
	}
}
