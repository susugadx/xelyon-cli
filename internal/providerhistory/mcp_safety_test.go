package providerhistory

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
)

func TestMCPRawOutputArtifactOmitReason(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "bare password in free text JSON value",
			content: `{"text":"password: hunter2"}`,
			want:    string(rawoutputs.ReasonSensitiveArtifactForbidden),
		},
		{
			name:    "secret assignment",
			content: `{"text":"secret=super-secret"}`,
			want:    string(rawoutputs.ReasonSensitiveArtifactForbidden),
		},
		{
			name:    "bare token in free text JSON value",
			content: `{"text":"token: ghp_secret"}`,
			want:    string(rawoutputs.ReasonSensitiveArtifactForbidden),
		},
		{
			name:    "authorization bearer header",
			content: `{"text":"Authorization: Bearer abc.def.ghi"}`,
			want:    string(rawoutputs.ReasonSensitiveArtifactForbidden),
		},
		{
			name:    "quoted authorization bearer JSON field",
			content: `{"Authorization":"Bearer abc.def.ghi"}`,
			want:    string(rawoutputs.ReasonSensitiveArtifactForbidden),
		},
		{
			name:    "api key assignment",
			content: `{"text":"api_key=abc123"}`,
			want:    string(rawoutputs.ReasonSensitiveArtifactForbidden),
		},
		{
			name:    "private marker only",
			content: `{"text":"private customer issue body"}`,
			want:    MCPSensitiveOrPrivateResultKeepReason,
		},
		{
			name:    "safe token metrics",
			content: `{"usage":{"tokens":123,"cached_tokens":45},"text":"total tokens: 168 cached_tokens: 45"}`,
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MCPRawOutputArtifactOmitReason(tt.content); got != tt.want {
				t.Fatalf("MCPRawOutputArtifactOmitReason(%q) = %q, want %q", tt.content, got, tt.want)
			}
		})
	}
}

func TestMCPRawOutputArtifactOmitReasonAllowsRuntimeExcerpt(t *testing.T) {
	tests := []struct {
		name   string
		reason string
		want   bool
	}{
		{name: "secret-like artifact forbidden", reason: string(rawoutputs.ReasonSensitiveArtifactForbidden), want: false},
		{name: "private-looking artifact keep", reason: MCPSensitiveOrPrivateResultKeepReason, want: true},
		{name: "artifact disabled", reason: "raw_output_artifacts_disabled", want: true},
		{name: "empty", reason: "", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MCPRawOutputArtifactOmitReasonAllowsRuntimeExcerpt(tt.reason); got != tt.want {
				t.Fatalf("MCPRawOutputArtifactOmitReasonAllowsRuntimeExcerpt(%q) = %v, want %v", tt.reason, got, tt.want)
			}
		})
	}
}
