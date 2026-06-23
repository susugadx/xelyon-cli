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
			name:    "short bare token in free text JSON value",
			content: `{"text":"token: abc123"}`,
			want:    string(rawoutputs.ReasonSensitiveArtifactForbidden),
		},
		{
			name:    "short bare token JSON field",
			content: `{"token":"abc123"}`,
			want:    string(rawoutputs.ReasonSensitiveArtifactForbidden),
		},
		{
			name:    "double quoted exact token assignment",
			content: `"token"=abc123`,
			want:    string(rawoutputs.ReasonSensitiveArtifactForbidden),
		},
		{
			name:    "single quoted exact token assignment",
			content: `'token'=abc123`,
			want:    string(rawoutputs.ReasonSensitiveArtifactForbidden),
		},
		{
			name:    "embedded double quoted exact token assignment",
			content: `prefix "token"=abc123 suffix`,
			want:    string(rawoutputs.ReasonSensitiveArtifactForbidden),
		},
		{
			name:    "quoted exact token assignment value",
			content: `"token"="abc123"`,
			want:    string(rawoutputs.ReasonSensitiveArtifactForbidden),
		},
		{
			name:    "exact token array assignment value",
			content: `token=["abc123"]`,
			want:    string(rawoutputs.ReasonSensitiveArtifactForbidden),
		},
		{
			name:    "exact token object assignment value",
			content: `token={"value":"abc123"}`,
			want:    string(rawoutputs.ReasonSensitiveArtifactForbidden),
		},
		{
			name:    "dotted token object assignment value",
			content: `github.token={"value":"ghp_dot"}`,
			want:    string(rawoutputs.ReasonSensitiveArtifactForbidden),
		},
		{
			name:    "github dotted token assignment",
			content: `github.token=ghp_dot`,
			want:    string(rawoutputs.ReasonSensitiveArtifactForbidden),
		},
		{
			name:    "auth dotted token assignment",
			content: `auth.token=abc123`,
			want:    string(rawoutputs.ReasonSensitiveArtifactForbidden),
		},
		{
			name:    "csrf camel token assignment",
			content: `csrfToken=csrf-secret`,
			want:    string(rawoutputs.ReasonSensitiveArtifactForbidden),
		},
		{
			name:    "github token suffix JSON field",
			content: `{"github_token":"ghp_secret"}`,
			want:    string(rawoutputs.ReasonSensitiveArtifactForbidden),
		},
		{
			name:    "github token dotted JSON field",
			content: `{"github.token":"ghp_dot"}`,
			want:    string(rawoutputs.ReasonSensitiveArtifactForbidden),
		},
		{
			name:    "csrf token camel case JSON field",
			content: `{"csrfToken":"csrf-secret"}`,
			want:    string(rawoutputs.ReasonSensitiveArtifactForbidden),
		},
		{
			name:    "access token camel case numeric JSON field",
			content: `{"accessToken":12345}`,
			want:    string(rawoutputs.ReasonSensitiveArtifactForbidden),
		},
		{
			name:    "exact numeric token JSON field",
			content: `{"token":123456}`,
			want:    string(rawoutputs.ReasonSensitiveArtifactForbidden),
		},
		{
			name:    "embedded quoted token JSON field",
			content: `prefix "token":"abc123" suffix`,
			want:    string(rawoutputs.ReasonSensitiveArtifactForbidden),
		},
		{
			name:    "embedded dotted token JSON field",
			content: `prefix "github.token":"ghp_dot" suffix`,
			want:    string(rawoutputs.ReasonSensitiveArtifactForbidden),
		},
		{
			name:    "embedded csrf token JSON field",
			content: `prefix "csrfToken":"csrf-secret" suffix`,
			want:    string(rawoutputs.ReasonSensitiveArtifactForbidden),
		},
		{
			name:    "embedded unquoted dotted token field",
			content: `prefix github.token:ghp_dot suffix`,
			want:    string(rawoutputs.ReasonSensitiveArtifactForbidden),
		},
		{
			name:    "quoted spaced token JSON label",
			content: `{"GitHub Token":"ghp_secret"}`,
			want:    string(rawoutputs.ReasonSensitiveArtifactForbidden),
		},
		{
			name:    "quoted spaced token assignment label",
			content: `"API Token"="ghp_secret"`,
			want:    string(rawoutputs.ReasonSensitiveArtifactForbidden),
		},
		{
			name:    "quoted spaced token structured label",
			content: `"GitHub Token"={"value":"ghp_secret"}`,
			want:    string(rawoutputs.ReasonSensitiveArtifactForbidden),
		},
		{
			name:    "bearer token in free text JSON value",
			content: `{"text":"token: Bearer abc.def.ghi"}`,
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
		{
			name:    "safe token metric key variants",
			content: `{"usage":{"tokens":4096,"cached_tokens":1024,"token_count":4,"total_tokens":5120}}`,
			want:    "",
		},
		{
			name:    "safe quoted spaced plural token metrics",
			content: `{"GitHub Tokens":4096,"API Tokens":1024}`,
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
		{name: "private-looking artifact keep", reason: MCPSensitiveOrPrivateResultKeepReason, want: false},
		{name: "artifact dry run", reason: "raw_output_artifacts_dry_run", want: true},
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
