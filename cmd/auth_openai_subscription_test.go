package cmd

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	openaisubscription "github.com/susugadx/xelyon-cli/internal/api/providers/openai_subscription"
)

func TestOpenAISubscriptionAuthStatusMissingIsLoginRequired(t *testing.T) {
	authDir := filepath.Join(t.TempDir(), "auth")
	t.Setenv("XELYON_OPENAI_SUBSCRIPTION_AUTH_DIR", authDir)
	out := newRootCommandExecutionTest(t)
	rootCmd.SetArgs([]string{"auth", "openai-subscription", "status"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root Execute() error = %v\noutput:\n%s", err, out.String())
	}
	output := out.String()
	for _, want := range []string{"OpenAI Subscription auth", "Status: login_required", "Run: xelyon auth openai-subscription login"} {
		if !strings.Contains(output, want) {
			t.Fatalf("status output missing %q:\n%s", want, output)
		}
	}
	if strings.Contains(strings.ToLower(output), "api key") {
		t.Fatalf("status output mentioned API key:\n%s", output)
	}
}

func TestOpenAISubscriptionAuthStatusMasksAccountAndRedactsTokens(t *testing.T) {
	authDir := filepath.Join(t.TempDir(), "auth")
	t.Setenv("XELYON_OPENAI_SUBSCRIPTION_AUTH_DIR", authDir)
	config := openaisubscription.DefaultSubscriptionAuthConfig()
	accessToken := fakeCmdSubscriptionJWT(t, map[string]any{
		"chatgpt_account_id": "acct_secretabcd",
		"exp":                time.Now().Add(time.Hour).Unix(),
	})
	if err := openaisubscription.SaveSubscriptionCredential(config, openaisubscription.SubscriptionCredential{
		AccessToken:  accessToken,
		RefreshToken: "refresh-secret-token",
		AccountID:    "acct_secretabcd",
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("SaveSubscriptionCredential() error = %v", err)
	}
	out := newRootCommandExecutionTest(t)
	rootCmd.SetArgs([]string{"auth", "chatgpt", "status"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root Execute() error = %v\noutput:\n%s", err, out.String())
	}
	output := out.String()
	if !strings.Contains(output, "Account: acct_****abcd") {
		t.Fatalf("status output did not mask account:\n%s", output)
	}
	for _, leaked := range []string{accessToken, "refresh-secret-token", "acct_secretabcd"} {
		if strings.Contains(output, leaked) {
			t.Fatalf("status output leaked %q:\n%s", leaked, output)
		}
	}
}

func TestOpenAISubscriptionAuthStatusRedactsConfiguredEndpointSecrets(t *testing.T) {
	authDir := filepath.Join(t.TempDir(), "auth")
	t.Setenv("XELYON_OPENAI_SUBSCRIPTION_AUTH_DIR", authDir)
	t.Setenv("XELYON_OPENAI_SUBSCRIPTION_ENDPOINT", "https://user-secret:pass-secret@proxy.example.test/backend-api/codex/responses?token=query-secret&debug=query-secret#frag-secret")

	out := newRootCommandExecutionTest(t)
	rootCmd.SetArgs([]string{"auth", "openai-subscription", "status"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root Execute() error = %v\noutput:\n%s", err, out.String())
	}
	output := out.String()
	if !strings.Contains(output, "Endpoint: https://redacted@proxy.example.test/backend-api/codex/responses?redacted#redacted") {
		t.Fatalf("status output endpoint was not sanitized as expected:\n%s", output)
	}
	for _, leaked := range []string{"user-secret", "pass-secret", "query-secret", "frag-secret"} {
		if strings.Contains(output, leaked) {
			t.Fatalf("status output leaked %q:\n%s", leaked, output)
		}
	}
}

func TestOpenAISubscriptionAuthLogoutIsIdempotent(t *testing.T) {
	authDir := filepath.Join(t.TempDir(), "auth")
	t.Setenv("XELYON_OPENAI_SUBSCRIPTION_AUTH_DIR", authDir)
	config := openaisubscription.DefaultSubscriptionAuthConfig()
	if err := openaisubscription.SaveSubscriptionCredential(config, openaisubscription.SubscriptionCredential{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		AccountID:    "acct_1234abcd",
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("SaveSubscriptionCredential() error = %v", err)
	}
	for i := 0; i < 2; i++ {
		out := newRootCommandExecutionTest(t)
		rootCmd.SetArgs([]string{"auth", "openai-subscription", "logout"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("logout %d error = %v\noutput:\n%s", i, err, out.String())
		}
	}
	if _, err := os.Stat(config.AuthDir); err != nil {
		t.Fatalf("auth dir should remain after logout: %v", err)
	}
	if _, err := os.Stat(config.AuthDir + "/openai_subscription.json"); !os.IsNotExist(err) {
		t.Fatalf("auth token stat error = %v, want not exist", err)
	}
}

func TestOpenAISubscriptionAuthHelpShowsDeviceLogin(t *testing.T) {
	out := newRootCommandExecutionTest(t)
	rootCmd.SetArgs([]string{"auth", "openai-subscription", "login", "--help"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root Execute() error = %v\noutput:\n%s", err, out.String())
	}
	output := out.String()
	for _, want := range []string{"--device", "Sign in with ChatGPT/Codex OAuth"} {
		if !strings.Contains(output, want) {
			t.Fatalf("login help missing %q:\n%s", want, output)
		}
	}
}

func fakeCmdSubscriptionJWT(t *testing.T, claims map[string]any) string {
	t.Helper()
	header, err := json.Marshal(map[string]any{"alg": "none", "typ": "JWT"})
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(header) + "." +
		base64.RawURLEncoding.EncodeToString(payload) + ".signature"
}
