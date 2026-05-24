package llmcatalog

import "testing"

func TestGeminiFunctionCallingPolicy(t *testing.T) {
	tests := []struct {
		name        string
		request     string
		catalog     string
		wantRequest string
		wantPolicy  string
		wantKnown   bool
		wantEnabled bool
	}{
		{
			name:        "supported catalog alias",
			request:     "corp-flash",
			catalog:     "models/gemini-3.5-flash",
			wantRequest: "corp-flash",
			wantPolicy:  "gemini-3.5-flash",
			wantKnown:   true,
			wantEnabled: true,
		},
		{
			name:        "unsupported catalog alias",
			request:     "corp-lite",
			catalog:     "models/gemini-2.0-flash-lite",
			wantRequest: "corp-lite",
			wantPolicy:  "gemini-2.0-flash-lite",
			wantKnown:   true,
			wantEnabled: false,
		},
		{
			name:        "unknown alias remains optimistic",
			request:     "corp-gemini",
			wantRequest: "corp-gemini",
			wantPolicy:  "corp-gemini",
			wantKnown:   false,
			wantEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewGeminiFunctionCallingPolicy(tt.request, tt.catalog)
			if got.RequestModel() != tt.wantRequest {
				t.Fatalf("RequestModel() = %q, want %q", got.RequestModel(), tt.wantRequest)
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
		})
	}
}
