package cmd

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/agent"
	"github.com/susugadx/xelyon-cli/internal/api"
	openaisubscription "github.com/susugadx/xelyon-cli/internal/api/providers/openai_subscription"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestRootCommand_InteractiveMissingProviderCredentialStartsTUIWithPlaceholder(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OPENAI_API_KEY", "")

	var gotProvider api.Provider
	runTUI = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
		gotProvider = provider
	}
	runOnce = func(query string, model string, provider api.Provider, cfg *config.Config, autoApprove bool, quiet bool) error {
		t.Fatal("one-shot path must not run for interactive provider setup")
		return nil
	}

	rootCmd.SetArgs([]string{"--interactive", "--provider", "openai", "--no-update-check"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if gotProvider == nil {
		t.Fatal("expected TUI provider")
	}
	if !api.IsProviderSetupRequired(gotProvider) {
		t.Fatalf("provider = %T, want setup placeholder", gotProvider)
	}
	msg, ok := api.ProviderSetupRequiredMessage(gotProvider)
	if !ok {
		t.Fatal("expected provider setup message")
	}
	for _, fragment := range []string{"provider setup required", "OPENAI_API_KEY", "xelyon setup"} {
		if !strings.Contains(msg, fragment) {
			t.Fatalf("setup message missing %q:\n%s", fragment, msg)
		}
	}
}

func TestRootCommand_OneShotMissingProviderCredentialReturnsSetupGuidance(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OPENAI_API_KEY", "")

	onceCalled := false
	runOnce = func(query string, model string, provider api.Provider, cfg *config.Config, autoApprove bool, quiet bool) error {
		onceCalled = true
		return nil
	}

	rootCmd.SetArgs([]string{"--provider", "openai", "--no-update-check", "hello"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected setup guidance error")
	}
	if onceCalled {
		t.Fatal("one-shot runner must not be called without provider credential")
	}
	for _, fragment := range []string{"provider setup required", "OPENAI_API_KEY", "xelyon setup"} {
		if !strings.Contains(err.Error(), fragment) {
			t.Fatalf("error missing %q:\n%s", fragment, err.Error())
		}
	}
}

func TestRootCommand_OneShotOpenAISubscriptionExpiredTokenReachesRunner(t *testing.T) {
	withRootCommandTest(t)
	saveExpiredOpenAISubscriptionCredentialForRootTest(t)

	onceCalled := false
	runOnce = func(query string, model string, provider api.Provider, cfg *config.Config, autoApprove bool, quiet bool) error {
		onceCalled = true
		if api.IsProviderSetupRequired(provider) {
			t.Fatalf("provider = %T, want executable subscription provider", provider)
		}
		if query != "hello" {
			t.Fatalf("query = %q, want hello", query)
		}
		return nil
	}

	rootCmd.SetArgs([]string{"--provider", "openai_subscription", "--model", "gpt-5.4-mini", "--no-update-check", "hello"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !onceCalled {
		t.Fatal("one-shot runner was not called")
	}
}

func TestRootCommand_HeadlessOpenAISubscriptionExpiredTokenReachesRunner(t *testing.T) {
	withRootCommandTest(t)
	saveExpiredOpenAISubscriptionCredentialForRootTest(t)

	headlessCalled := false
	runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config, options agent.HeadlessRunOptions) *agent.HeadlessResult {
		headlessCalled = true
		if api.IsProviderSetupRequired(provider) {
			t.Fatalf("provider = %T, want executable subscription provider", provider)
		}
		if query != "hello" {
			t.Fatalf("query = %q, want hello", query)
		}
		return agent.NewSuccessResult(provider.Name(), model, "ok", nil, 0)
	}

	rootCmd.SetArgs([]string{"--headless", "--provider", "openai_subscription", "--model", "gpt-5.4-mini", "--no-update-check", "hello"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !headlessCalled {
		t.Fatal("headless runner was not called")
	}
}

func saveExpiredOpenAISubscriptionCredentialForRootTest(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XELYON_OPENAI_SUBSCRIPTION_AUTH_DIR", filepath.Join(t.TempDir(), "auth"))
	if err := openaisubscription.SaveSubscriptionCredential(openaisubscription.DefaultSubscriptionAuthConfig(), openaisubscription.SubscriptionCredential{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		AccountID:    "acct_1234abcd",
		ExpiresAt:    time.Now().Add(-time.Minute),
	}); err != nil {
		t.Fatalf("SaveSubscriptionCredential() error = %v", err)
	}
}
