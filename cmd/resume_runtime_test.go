package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/history"
)

func writeInvalidDefaultProviderConfig(t *testing.T) {
	t.Helper()

	configDir := filepath.Join(os.Getenv("HOME"), ".xelyon")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatalf("MkdirAll(configDir) error = %v", err)
	}
	data := []byte("default_provider: invalid-provider\ndefault_model: invalid-model\n")
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), data, 0o600); err != nil {
		t.Fatalf("WriteFile(config.yaml) error = %v", err)
	}
}

func seedOllamaResumeSession(t *testing.T, model string) string {
	t.Helper()

	storage, err := history.NewStorage()
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}
	session := history.NewSession(model)
	session.ProviderName = "ollama"
	session.ProviderConfigKey = "ollama"
	session.AddMessage("user", "saved question", model)
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save(session) error = %v", err)
	}
	return session.ID
}

func seedAzureResumeSession(t *testing.T, model string) string {
	t.Helper()

	storage, err := history.NewStorage()
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}
	session := history.NewSession(model)
	session.ProviderName = "Azure OpenAI"
	session.ProviderConfigKey = "azure"
	session.AddMessage("user", "saved azure question", model)
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save(session) error = %v", err)
	}
	return session.ID
}

func seedMetadataLessResumeSession(t *testing.T, model string) string {
	t.Helper()

	storage, err := history.NewStorage()
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}
	session := history.NewSession(model)
	session.AddMessage("user", "legacy saved question", model)
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save(session) error = %v", err)
	}
	return session.ID
}

func assertOllamaResumeRuntime(t *testing.T, model string, provider api.Provider) {
	t.Helper()
	if model != "qwen2.5-coder:14b" {
		t.Fatalf("model = %q, want saved session model", model)
	}
	if provider == nil || !strings.EqualFold(provider.Name(), "ollama") {
		t.Fatalf("provider = %T %q, want ollama", provider, providerNameForTest(provider))
	}
}

func assertExplicitOllamaRuntime(t *testing.T, model string, provider api.Provider) {
	t.Helper()
	if model != "qwen2.5-coder:14b" {
		t.Fatalf("model = %q, want explicit runtime model", model)
	}
	if provider == nil || !strings.EqualFold(provider.Name(), "ollama") {
		t.Fatalf("provider = %T %q, want ollama", provider, providerNameForTest(provider))
	}
}

func configureAzureProviderEnv(t *testing.T) {
	t.Helper()

	t.Setenv("AZURE_OPENAI_API_KEY", "azure-key")
	t.Setenv("AZURE_OPENAI_AUTH_TOKEN", "")
	t.Setenv("AZURE_OPENAI_AUTH_TOKEN_COMMAND", "")
	t.Setenv("AZURE_OPENAI_BASE_URL", "https://example.openai.azure.com/openai/v1")
}

func assertAzureResumeRuntime(t *testing.T, model string, provider api.Provider) {
	t.Helper()
	if model != "azure-gpt-5.4" {
		t.Fatalf("model = %q, want saved Azure deployment", model)
	}
	if provider == nil || config.CanonicalProviderName(provider.Name()) != "azure" {
		t.Fatalf("provider = %T %q, want Azure OpenAI", provider, providerNameForTest(provider))
	}
}

func providerNameForTest(provider api.Provider) string {
	if provider == nil {
		return ""
	}
	return provider.Name()
}

func TestResumeCommandDirectUsesSavedRuntimeWhenDefaultConfigInvalid(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())
	writeInvalidDefaultProviderConfig(t)
	sessionID := seedOllamaResumeSession(t, "qwen2.5-coder:14b")

	var directCalled bool
	runTUIWithResumeDirect = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool, gotSessionID string) error {
		directCalled = true
		assertOllamaResumeRuntime(t, model, provider)
		if gotSessionID != sessionID {
			t.Fatalf("sessionID = %q, want %q", gotSessionID, sessionID)
		}
		return nil
	}

	rootCmd.SetArgs([]string{"resume", sessionID, "--no-update-check"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !directCalled {
		t.Fatal("expected direct resume path")
	}
}

func TestResumeCommandDirectTreatsSavedAzureDeploymentAsExplicit(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())
	configureAzureProviderEnv(t)
	sessionID := seedAzureResumeSession(t, "azure-gpt-5.4")

	var directCalled bool
	runTUIWithResumeDirect = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool, gotSessionID string) error {
		directCalled = true
		assertAzureResumeRuntime(t, model, provider)
		if gotSessionID != sessionID {
			t.Fatalf("sessionID = %q, want %q", gotSessionID, sessionID)
		}
		return nil
	}

	rootCmd.SetArgs([]string{"resume", sessionID, "--no-update-check"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !directCalled {
		t.Fatal("expected direct resume path")
	}
}

func TestResumeCommandLastTreatsSavedAzureDeploymentAsExplicit(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())
	configureAzureProviderEnv(t)
	seedAzureResumeSession(t, "azure-gpt-5.4")

	var resumeCalled bool
	runTUIWithResume = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) error {
		resumeCalled = true
		assertAzureResumeRuntime(t, model, provider)
		return nil
	}

	rootCmd.SetArgs([]string{"resume", "--last", "--no-update-check"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !resumeCalled {
		t.Fatal("expected last resume path")
	}
}

func TestRootResumeTreatsSavedAzureDeploymentAsExplicit(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())
	configureAzureProviderEnv(t)
	seedAzureResumeSession(t, "azure-gpt-5.4")

	var resumeCalled bool
	runTUIWithResume = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) error {
		resumeCalled = true
		assertAzureResumeRuntime(t, model, provider)
		return nil
	}

	rootCmd.SetArgs([]string{"--resume", "--no-update-check"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !resumeCalled {
		t.Fatal("expected root --resume path")
	}
}

func TestResumeCommandLastUsesSavedRuntimeWhenDefaultConfigInvalid(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())
	writeInvalidDefaultProviderConfig(t)
	seedOllamaResumeSession(t, "qwen2.5-coder:14b")

	var resumeCalled bool
	runTUIWithResume = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) error {
		resumeCalled = true
		assertOllamaResumeRuntime(t, model, provider)
		return nil
	}

	rootCmd.SetArgs([]string{"resume", "--last", "--no-update-check"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !resumeCalled {
		t.Fatal("expected last resume path")
	}
}

func TestRootResumeUsesSavedRuntimeWhenDefaultConfigInvalid(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())
	writeInvalidDefaultProviderConfig(t)
	seedOllamaResumeSession(t, "qwen2.5-coder:14b")

	var resumeCalled bool
	runTUIWithResume = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) error {
		resumeCalled = true
		assertOllamaResumeRuntime(t, model, provider)
		return nil
	}

	rootCmd.SetArgs([]string{"--resume", "--no-update-check"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !resumeCalled {
		t.Fatal("expected root --resume path")
	}
}

func TestResumeCommandPickerUsesBootstrapRuntimeWhenDefaultConfigInvalid(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())
	writeInvalidDefaultProviderConfig(t)

	var pickerCalled bool
	runTUIWithResumePicker = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool, all bool) {
		pickerCalled = true
		if model != "resume-bootstrap" {
			t.Fatalf("model = %q, want resume-bootstrap", model)
		}
		if provider == nil || provider.Name() != "resume_bootstrap" {
			t.Fatalf("provider = %T %q, want resume_bootstrap", provider, providerNameForTest(provider))
		}
		_, err := provider.ChatWithTools(context.Background(), "", nil, model)
		if err == nil || !strings.Contains(err.Error(), "resume a session or switch provider") {
			t.Fatalf("bootstrap ChatWithTools error = %v, want resume guidance", err)
		}
	}

	rootCmd.SetArgs([]string{"resume", "--no-update-check"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !pickerCalled {
		t.Fatal("expected resume picker path")
	}
}

func TestRootResumeWithNoSessionsStillStartsResumeTUI(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())

	var resumeCalled bool
	runTUIWithResume = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) error {
		resumeCalled = true
		assertExplicitOllamaRuntime(t, model, provider)
		return nil
	}

	rootCmd.SetArgs([]string{"--resume", "--provider", "ollama", "--model", "qwen2.5-coder:14b", "--no-update-check"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !resumeCalled {
		t.Fatal("expected root --resume to reach TUI resume path")
	}
}

func TestResumeCommandLastWithNoSessionsStillStartsResumeTUI(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())

	var resumeCalled bool
	runTUIWithResume = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) error {
		resumeCalled = true
		assertExplicitOllamaRuntime(t, model, provider)
		return nil
	}

	rootCmd.SetArgs([]string{"resume", "--last", "--provider", "ollama", "--model", "qwen2.5-coder:14b", "--no-update-check"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !resumeCalled {
		t.Fatal("expected resume --last to reach TUI resume path")
	}
}

func TestResumeCommandDirectUsesDefaultRuntimeForMetadataLessSession(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())
	sessionID := seedMetadataLessResumeSession(t, "legacy-model")

	var directCalled bool
	runTUIWithResumeDirect = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool, gotSessionID string) error {
		directCalled = true
		assertExplicitOllamaRuntime(t, model, provider)
		if gotSessionID != sessionID {
			t.Fatalf("sessionID = %q, want %q", gotSessionID, sessionID)
		}
		return nil
	}

	rootCmd.SetArgs([]string{"resume", sessionID, "--provider", "ollama", "--model", "qwen2.5-coder:14b", "--no-update-check"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !directCalled {
		t.Fatal("expected direct resume path")
	}
}

func TestResumeCommandLastUsesDefaultRuntimeForMetadataLessSession(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())
	seedMetadataLessResumeSession(t, "legacy-model")

	var resumeCalled bool
	runTUIWithResume = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) error {
		resumeCalled = true
		assertExplicitOllamaRuntime(t, model, provider)
		return nil
	}

	rootCmd.SetArgs([]string{"resume", "--last", "--provider", "ollama", "--model", "qwen2.5-coder:14b", "--no-update-check"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !resumeCalled {
		t.Fatal("expected last resume path")
	}
}
