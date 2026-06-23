package rawoutputs

import (
	"context"
	"strings"
	"testing"
)

func TestStoreCreateRejectsTokenCredentialJSONFields(t *testing.T) {
	store := newTestStore(t, StoreOptions{})

	tests := []struct {
		name string
		body string
	}{
		{name: "snake suffix", body: `{"github_token":"ghp_secret"}`},
		{name: "camel suffix", body: `{"csrfToken":"csrf-secret"}`},
		{name: "numeric suffix", body: `{"accessToken":12345}`},
		{name: "dotted suffix", body: `{"github.token":"ghp_dot"}`},
		{name: "dotted numeric suffix", body: `{"auth.token":123456}`},
		{name: "exact numeric", body: `{"token":123456}`},
		{name: "exact quoted numeric", body: `{"token":"123456"}`},
		{name: "double quoted exact assignment", body: `"token"=abc123`},
		{name: "single quoted exact assignment", body: `'token'=abc123`},
		{name: "quoted exact assignment value", body: `"token"="abc123"`},
		{name: "exact array assignment value", body: `token=["abc123"]`},
		{name: "exact object assignment value", body: `token={"value":"abc123"}`},
		{name: "dotted object assignment value", body: `github.token={"value":"ghp_dot"}`},
		{name: "embedded double quoted exact assignment", body: `prefix "token"=abc123 suffix`},
		{name: "dotted assignment", body: `github.token=ghp_dot`},
		{name: "auth dotted assignment", body: `auth.token=abc123`},
		{name: "camel assignment", body: `csrfToken=csrf-secret`},
		{name: "embedded dotted assignment", body: `prefix github.token=ghp_dot suffix`},
		{name: "quoted dotted assignment", body: `"github.token"="ghp_dot"`},
		{name: "quoted spaced JSON label", body: `{"GitHub Token":"ghp_secret"}`},
		{name: "quoted spaced assignment label", body: `"API Token"="ghp_secret"`},
		{name: "single quoted spaced assignment label", body: `'API Token'='ghp_secret'`},
		{name: "quoted spaced structured label", body: `"GitHub Token"={"value":"ghp_secret"}`},
		{name: "embedded quoted exact", body: `prefix "token":"abc123" suffix`},
		{name: "embedded dotted suffix", body: `prefix "github.token":"ghp_dot" suffix`},
		{name: "embedded camel suffix", body: `prefix "csrfToken":"csrf-secret" suffix`},
		{name: "embedded unquoted dotted suffix", body: `prefix github.token:ghp_dot suffix`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := testCreateRequest("session-token-safety-"+strings.ReplaceAll(tt.name, " ", "-"), "call-token", tt.body)
			if _, err := store.Create(context.Background(), req); ReasonOf(err) != ReasonSensitiveArtifactForbidden {
				t.Fatalf("Create(%s) error = %v, want %s", tt.name, err, ReasonSensitiveArtifactForbidden)
			}
		})
	}
}

func TestStoreCreateAllowsTokenMetricKeyVariants(t *testing.T) {
	store := newTestStore(t, StoreOptions{})

	tests := []struct {
		name string
		body string
	}{
		{
			name: "metric key variants",
			body: `{"usage":{"tokens":8192,"cached_tokens":1024,"token_count":4,"total_tokens":9216}}`,
		},
		{
			name: "quoted spaced plural labels",
			body: `{"GitHub Tokens":4096,"API Tokens":1024}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionID := "session-token-metric-" + strings.ReplaceAll(tt.name, " ", "-")
			result, err := store.Create(context.Background(), testCreateRequest(sessionID, "call-token-metric", tt.body))
			if err != nil {
				t.Fatalf("Create(%s) error = %v", tt.name, err)
			}

			resolved, err := store.Resolve(context.Background(), result.Ref)
			if err != nil {
				t.Fatalf("Resolve(%s) error = %v", tt.name, err)
			}
			got, err := readResolved(resolved)
			if err != nil {
				t.Fatalf("read %s: %v", tt.name, err)
			}
			if got != tt.body {
				t.Fatalf("resolved body = %q, want %q", got, tt.body)
			}
		})
	}
}

func TestStoreRejectsSensitiveBodyAcrossChunkBoundary(t *testing.T) {
	store := newTestStore(t, StoreOptions{ChunkBytes: 8, MaxArtifactBytes: 1024})
	body := "safe\nAuthorization: Bearer secret-value\n"

	if _, err := store.Create(context.Background(), testCreateRequest("session-chunk-secret", "call-secret", body)); ReasonOf(err) != ReasonSensitiveArtifactForbidden {
		t.Fatalf("Create(chunk boundary sensitive body) error = %v, want %s", err, ReasonSensitiveArtifactForbidden)
	}

	body = `safe prefix "token":"abc123" suffix`
	if _, err := store.Create(context.Background(), testCreateRequest("session-chunk-token-field", "call-token-field", body)); ReasonOf(err) != ReasonSensitiveArtifactForbidden {
		t.Fatalf("Create(chunk boundary embedded token field) error = %v, want %s", err, ReasonSensitiveArtifactForbidden)
	}

	body = `safe prefix "token"=abc123 suffix`
	if _, err := store.Create(context.Background(), testCreateRequest("session-chunk-token-assignment", "call-token-assignment", body)); ReasonOf(err) != ReasonSensitiveArtifactForbidden {
		t.Fatalf("Create(chunk boundary embedded quoted token assignment) error = %v, want %s", err, ReasonSensitiveArtifactForbidden)
	}

	body = `safe prefix github.token=ghp_dot suffix`
	if _, err := store.Create(context.Background(), testCreateRequest("session-chunk-token-dotted-assignment", "call-token-dotted-assignment", body)); ReasonOf(err) != ReasonSensitiveArtifactForbidden {
		t.Fatalf("Create(chunk boundary embedded dotted token assignment) error = %v, want %s", err, ReasonSensitiveArtifactForbidden)
	}

	body = `safe prefix "GitHub Token":"ghp_secret" suffix`
	if _, err := store.Create(context.Background(), testCreateRequest("session-chunk-spaced-token-label", "call-spaced-token-label", body)); ReasonOf(err) != ReasonSensitiveArtifactForbidden {
		t.Fatalf("Create(chunk boundary embedded spaced token label) error = %v, want %s", err, ReasonSensitiveArtifactForbidden)
	}
}
