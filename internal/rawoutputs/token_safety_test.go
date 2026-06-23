package rawoutputs

import (
	"strings"
	"testing"
)

func TestSensitiveTokenKey(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		{key: "token", want: true},
		{key: "TOKEN", want: true},
		{key: "github.token", want: true},
		{key: "auth.token", want: true},
		{key: "github_token", want: true},
		{key: "auth-token", want: true},
		{key: "GitHub Token", want: true},
		{key: "API Token", want: true},
		{key: "csrfToken", want: true},
		{key: "accessToken", want: true},
		{key: "tokens", want: false},
		{key: "GitHub Tokens", want: false},
		{key: "API Tokens", want: false},
		{key: "cached_tokens", want: false},
		{key: "token_count", want: false},
		{key: "total_tokens", want: false},
		{key: "promptTokens", want: false},
		{key: "completion_tokens", want: false},
		{key: "tokenizer", want: false},
		{key: "token_value", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if got := sensitiveTokenKey(tt.key); got != tt.want {
				t.Fatalf("sensitiveTokenKey(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestTokenCredentialFieldDetection(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "exact assignment", value: "token=secret-value", want: true},
		{name: "bearer exact assignment", value: "token=Bearer abcdef", want: true},
		{name: "numeric exact assignment", value: "token=123456", want: true},
		{name: "double quoted exact assignment", value: `"token"=abc123`, want: true},
		{name: "single quoted exact assignment", value: `'token'=abc123`, want: true},
		{name: "quoted exact assignment value", value: `"token"="abc123"`, want: true},
		{name: "single quoted exact assignment value", value: `'token'='abc123'`, want: true},
		{name: "exact array assignment value", value: `token=["abc123"]`, want: true},
		{name: "exact object assignment value", value: `token={"value":"abc123"}`, want: true},
		{name: "dotted object assignment value", value: `github.token={"value":"ghp_dot"}`, want: true},
		{name: "nested object assignment value", value: `token={"nested":{"value":"abc123"}}`, want: true},
		{name: "array object assignment value", value: `token=[{"value":"abc123"}]`, want: true},
		{name: "structured assignment value with quoted delimiter", value: `token={"value":"abc]123"}`, want: true},
		{name: "malformed object assignment value", value: `token={"value":"abc123" suffix`, want: true},
		{name: "embedded exact assignment", value: `prefix "token"=abc123 suffix`, want: true},
		{name: "dotted assignment", value: `github.token=ghp_dot`, want: true},
		{name: "auth dotted assignment", value: `auth.token=abc123`, want: true},
		{name: "camel assignment", value: `csrfToken=csrf-secret`, want: true},
		{name: "snake assignment", value: `github_token=ghp_secret`, want: true},
		{name: "hyphen assignment", value: `auth-token=secret`, want: true},
		{name: "hyphen token header", value: `Access-Token: abc123`, want: true},
		{name: "embedded dotted assignment", value: `prefix github.token=ghp_dot suffix`, want: true},
		{name: "quoted dotted assignment", value: `"github.token"="ghp_dot"`, want: true},
		{name: "quoted spaced json label", value: `{"GitHub Token":"ghp_secret"}`, want: true},
		{name: "quoted spaced assignment label", value: `"API Token"="ghp_secret"`, want: true},
		{name: "single quoted spaced assignment label", value: `'API Token'='ghp_secret'`, want: true},
		{name: "quoted spaced structured label", value: `"GitHub Token"={"value":"ghp_secret"}`, want: true},
		{name: "json exact field", value: `{"token":"abc123"}`, want: true},
		{name: "json numeric exact field", value: `{"token":4096}`, want: true},
		{name: "json snake suffix field", value: `{"github_token":"ghp_secret"}`, want: true},
		{name: "json dotted suffix field", value: `{"github.token":"ghp_dot"}`, want: true},
		{name: "json dotted numeric suffix field", value: `{"auth.token":123456}`, want: true},
		{name: "json camel suffix field", value: `{"csrfToken":"csrf-secret"}`, want: true},
		{name: "json camel numeric suffix field", value: `{"accessToken":12345}`, want: true},
		{name: "embedded quoted token field", value: `prefix "token":"abc123" suffix`, want: true},
		{name: "embedded unquoted dotted field", value: `prefix github.token:ghp_dot suffix`, want: true},
		{name: "token assignment inside JSON string", value: `{"text":"token=123456"}`, want: true},
		{name: "token colon inside JSON string", value: `{"text":"token: Bearer abc.def"}`, want: true},
		{name: "plural metric field", value: `{"tokens":4096}`, want: false},
		{name: "cached metric field", value: `{"cached_tokens":4096}`, want: false},
		{name: "token count metric field", value: `{"token_count":4096}`, want: false},
		{name: "total metric field", value: `{"total_tokens":4096}`, want: false},
		{name: "metric words", value: "usage tokens: 123 cached_tokens: 4 total_tokens: 127", want: false},
		{name: "quoted spaced plural metric label", value: `{"GitHub Tokens":4096}`, want: false},
		{name: "quoted spaced plural metric assignment", value: `"API Tokens"=4096`, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LooksSensitiveContent(tt.value); got != tt.want {
				t.Fatalf("LooksSensitiveContent(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestTokenCredentialFieldDetectionShortCircuitsFirstMatch(t *testing.T) {
	input := strings.Repeat("github_token=ghp_secret ", 2000)

	var got bool
	allocs := testing.AllocsPerRun(10, func() {
		got = LooksSensitiveContent(input)
	})

	if !got {
		t.Fatal("LooksSensitiveContent() = false, want true")
	}
	if allocs > 100 {
		t.Fatalf("LooksSensitiveContent() allocations = %.0f, want first-match bounded allocation", allocs)
	}
}

func TestTokenCredentialFieldRedaction(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		rejects []string
	}{
		{name: "exact assignment", input: "token=123456", want: "token=[redacted]", rejects: []string{"123456"}},
		{name: "quoted exact assignment", input: `"token"="quoted-value"`, want: "token=[redacted]", rejects: []string{"quoted-value"}},
		{name: "single quoted exact assignment", input: `'token'='abc123'`, want: "token=[redacted]", rejects: []string{"abc123"}},
		{name: "exact array assignment", input: `token=["abc123"]`, want: "token=[redacted]", rejects: []string{"abc123"}},
		{name: "exact object assignment", input: `token={"value":"abc123"}`, want: "token=[redacted]", rejects: []string{"value", "abc123"}},
		{name: "dotted object assignment", input: `prefix github.token={"value":"ghp_dot"} suffix`, want: "prefix github.token=[redacted] suffix", rejects: []string{"value", "ghp_dot"}},
		{name: "nested object assignment", input: `token={"nested":{"value":"abc123"}}`, want: "token=[redacted]", rejects: []string{"nested", "value", "abc123"}},
		{name: "array object assignment", input: `token=[{"value":"abc123"}]`, want: "token=[redacted]", rejects: []string{"value", "abc123"}},
		{name: "structured assignment with quoted delimiter", input: `token={"value":"abc]123"}`, want: "token=[redacted]", rejects: []string{"value", "abc]123"}},
		{name: "malformed object assignment", input: `prefix token={"value":"abc123" suffix`, want: "prefix token=[redacted]", rejects: []string{"value", "abc123"}},
		{name: "bearer dotted assignment", input: `auth.token=Bearer abc.def`, want: "auth.token=[redacted]", rejects: []string{"abc.def"}},
		{name: "snake assignment", input: `github_token=ghp_secret`, want: "github_token=[redacted]", rejects: []string{"ghp_secret"}},
		{name: "hyphen assignment", input: `auth-token=secret`, want: "auth-token=[redacted]", rejects: []string{"secret"}},
		{name: "hyphen token header", input: `Access-Token: abc123`, want: "Access-Token: [redacted]", rejects: []string{"abc123"}},
		{name: "dotted assignment", input: `prefix github.token=ghp_dot suffix`, want: "prefix github.token=[redacted] suffix", rejects: []string{"ghp_dot"}},
		{name: "quoted spaced json label", input: `{"GitHub Token":"ghp_secret"}`, want: `{GitHub Token: [redacted]}`, rejects: []string{"ghp_secret"}},
		{name: "quoted spaced assignment label", input: `"API Token"="ghp_secret"`, want: "API Token=[redacted]", rejects: []string{"ghp_secret"}},
		{name: "single quoted spaced assignment label", input: `'API Token'='ghp_secret'`, want: "API Token=[redacted]", rejects: []string{"ghp_secret"}},
		{name: "quoted spaced structured label", input: `"GitHub Token"={"value":"ghp_secret"}`, want: "GitHub Token=[redacted]", rejects: []string{"value", "ghp_secret"}},
		{name: "camel assignment", input: `csrfToken=csrf-secret`, want: "csrfToken=[redacted]", rejects: []string{"csrf-secret"}},
		{name: "colon field", input: `prefix "github.token":"ghp_dot" suffix`, want: "prefix github.token: [redacted] suffix", rejects: []string{"ghp_dot"}},
		{name: "camel colon field", input: `prefix "csrfToken":"csrf-secret" suffix`, want: "prefix csrfToken: [redacted] suffix", rejects: []string{"csrf-secret"}},
		{name: "metric fields", input: `{"usage":{"tokens":4096,"cached_tokens":1024,"token_count":4,"total_tokens":5120}}`, want: `{"usage":{"tokens":4096,"cached_tokens":1024,"token_count":4,"total_tokens":5120}}`},
		{name: "quoted spaced plural metric labels", input: `{"GitHub Tokens":4096,"API Tokens":1024}`, want: `{"GitHub Tokens":4096,"API Tokens":1024}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RedactDisplaySecrets(tt.input)
			if got != tt.want {
				t.Fatalf("RedactDisplaySecrets(%q) = %q, want %q", tt.input, got, tt.want)
			}
			for _, reject := range tt.rejects {
				if strings.Contains(got, reject) {
					t.Fatalf("RedactDisplaySecrets(%q) leaked %q: %q", tt.input, reject, got)
				}
			}
		})
	}
}
