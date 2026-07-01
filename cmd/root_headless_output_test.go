package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/agent"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

type headlessSequenceProvider struct {
	name      string
	responses []string
	index     int
}

func (p *headlessSequenceProvider) Name() string {
	return p.name
}

func (p *headlessSequenceProvider) SupportsImages() bool {
	return false
}

func (p *headlessSequenceProvider) IsFunctionCallingEnabled() bool {
	return true
}

func (p *headlessSequenceProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	if p.index >= len(p.responses) {
		return "done", nil
	}
	resp := p.responses[p.index]
	p.index++
	return resp, nil
}

func (p *headlessSequenceProvider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	return "", fmt.Errorf("not supported")
}

func TestRootCommand_PositionalQueryUsesHeadlessInJSONMode(t *testing.T) {
	withRootCommandTest(t)

	headlessCalled := false
	onceCalled := false
	interactiveCalled := false
	runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config, options agent.HeadlessRunOptions) *agent.HeadlessResult {
		headlessCalled = true
		if query != "hello" {
			t.Fatalf("query = %q, want hello", query)
		}
		return agent.NewSuccessResult(provider.Name(), model, "ok", nil, 0)
	}
	runOnce = func(query string, model string, provider api.Provider, cfg *config.Config, autoApprove bool, quiet bool) error {
		onceCalled = true
		return nil
	}
	runLegacyInteractive = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
		interactiveCalled = true
	}

	rootCmd.SetArgs([]string{"--output-format", "json", "--provider", "ollama", "--no-update-check", "hello"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !headlessCalled {
		t.Fatal("expected JSON mode to use headless path")
	}
	if onceCalled {
		t.Fatal("one-shot path must not be executed in JSON mode")
	}
	if interactiveCalled {
		t.Fatal("interactive path must not be executed in JSON mode")
	}
}

func TestRootCommand_OutputFormatIsCaseInsensitive(t *testing.T) {
	withRootCommandTest(t)

	headlessCalled := false
	runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config, options agent.HeadlessRunOptions) *agent.HeadlessResult {
		headlessCalled = true
		if query != "hello" {
			t.Fatalf("query = %q, want hello", query)
		}
		return agent.NewSuccessResult(provider.Name(), model, "ok", nil, 0)
	}

	rootCmd.SetArgs([]string{"--output-format", "JSON", "--provider", "ollama", "--no-update-check", "hello"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !headlessCalled {
		t.Fatal("expected normalized JSON mode to use headless path")
	}
}

func TestRootCommand_InvalidOutputFormatReturnsError(t *testing.T) {
	withRootCommandTest(t)
	rootCmd.SetArgs([]string{"--output-format", "yaml", "--no-update-check", "hello"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid output format")
	}
	if !strings.Contains(err.Error(), "invalid --output-format") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestRootCommand_HeadlessJSONStdoutIsPureJSON(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())

	tempDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	targetPath := filepath.Join(tempDir, "input.txt")
	if err := os.WriteFile(targetPath, []byte("hello from headless\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	provider := &headlessSequenceProvider{
		name: "headless-seq",
		responses: []string{
			fmt.Sprintf(`{"tool":"gather_context","args":{"query":%q}}`, targetPath),
			"done",
		},
	}

	runHeadless = func(ctx context.Context, query string, model string, providerArg api.Provider, cfg *config.Config, options agent.HeadlessRunOptions) *agent.HeadlessResult {
		if cfg == nil {
			cfg = config.DefaultConfig()
		} else {
			cloned := *cfg
			cfg = &cloned
		}
		cfg.ProjectMap.Enabled = false
		return agent.RunHeadlessWithConfig(ctx, query, model, provider, cfg)
	}

	oldStdout := os.Stdout
	oldColorOutput := color.Output
	r, w, _ := os.Pipe()
	os.Stdout = w
	color.Output = w

	rootCmd.SetArgs([]string{"--output-format", "json", "--provider", "ollama", "--no-update-check", "hello"})
	execErr := rootCmd.Execute()

	_ = w.Close()
	os.Stdout = oldStdout
	color.Output = oldColorOutput

	if execErr != nil {
		t.Fatalf("Execute() error = %v", execErr)
	}

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := strings.TrimSpace(buf.String())

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("stdout is not pure JSON: %v\noutput=%q", err, output)
	}
	if parsed["status"] != "success" {
		t.Fatalf("status = %v, want success", parsed["status"])
	}
	if parsed["response"] != "done" {
		t.Fatalf("response = %v, want done", parsed["response"])
	}
}

func TestRootCommand_HeadlessErrorReturnsErrorAfterPrintingJSON(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())

	runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config, options agent.HeadlessRunOptions) *agent.HeadlessResult {
		return agent.NewToolLoopLimitResult(provider.Name(), model, 10, nil, 0)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
	})

	rootCmd.SetArgs([]string{"--headless", "--provider", "ollama", "--no-update-check", "hello"})
	execErr := rootCmd.Execute()

	if execErr == nil {
		t.Fatal("expected headless error status to return command error")
	}
	if !strings.Contains(execErr.Error(), "headless execution failed") {
		t.Fatalf("unexpected error: %v", execErr)
	}
	if !rootCmd.SilenceUsage {
		t.Fatal("rootCmd.SilenceUsage = false, want true after printing headless error JSON")
	}
	if !rootCmd.SilenceErrors {
		t.Fatal("rootCmd.SilenceErrors = false, want true after printing headless error JSON")
	}
	if strings.Contains(stderr.String(), "Usage:") {
		t.Fatalf("stderr contains Cobra usage after headless error JSON:\n%s", stderr.String())
	}
	if strings.Contains(stderr.String(), "Error:") {
		t.Fatalf("stderr contains Cobra error after headless error JSON:\n%s", stderr.String())
	}

	output := strings.TrimSpace(stdout.String())

	var parsed agent.HeadlessResult
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("stdout is not headless JSON: %v\noutput=%q", err, output)
	}
	if parsed.Status != agent.HeadlessStatusError {
		t.Fatalf("status = %q, want %q", parsed.Status, agent.HeadlessStatusError)
	}
	if parsed.Error == nil || parsed.Error.Type != agent.HeadlessErrorTypeToolLoopLimit {
		t.Fatalf("error = %+v, want %s", parsed.Error, agent.HeadlessErrorTypeToolLoopLimit)
	}
	if parsed.FailureReason != agent.HeadlessFailureReasonToolLoopLimit {
		t.Fatalf("failure_reason = %q, want %q", parsed.FailureReason, agent.HeadlessFailureReasonToolLoopLimit)
	}
	if parsed.ExitPolicy != agent.HeadlessExitPolicyLegacy {
		t.Fatalf("exit_policy = %q, want %q", parsed.ExitPolicy, agent.HeadlessExitPolicyLegacy)
	}
	if parsed.RecommendedExitCode != 1 {
		t.Fatalf("recommended_exit_code = %d, want 1", parsed.RecommendedExitCode)
	}
}
