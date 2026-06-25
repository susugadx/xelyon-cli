package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

type headlessToolErrorUsageProvider struct {
	responses     []string
	callCount     int
	usageCallback api.UsageCallback
}

func (p *headlessToolErrorUsageProvider) Name() string { return "openai" }

func (p *headlessToolErrorUsageProvider) SupportsImages() bool { return false }

func (p *headlessToolErrorUsageProvider) IsFunctionCallingEnabled() bool { return true }

func (p *headlessToolErrorUsageProvider) SetUsageCallback(callback api.UsageCallback) {
	p.usageCallback = callback
}

func (p *headlessToolErrorUsageProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	if p.callCount == 0 && p.usageCallback != nil {
		p.usageCallback(api.Usage{
			InputTokens:       100,
			CachedInputTokens: 20,
			OutputTokens:      30,
			ThinkingTokens:    5,
		})
	}
	if p.callCount >= len(p.responses) {
		return p.responses[len(p.responses)-1], nil
	}
	response := p.responses[p.callCount]
	p.callCount++
	return response, nil
}

func (p *headlessToolErrorUsageProvider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	return p.ChatWithTools(ctx, systemPrompt, history, model)
}

func newHeadlessFailingToolProvider(t *testing.T, finalResponse string) *headlessToolErrorUsageProvider {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XELYON_EDIT_TOOL", "str_replace")

	dir := testSubDir(t)
	targetPath := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(targetPath, []byte("actual content\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	return &headlessToolErrorUsageProvider{
		responses: []string{
			fmt.Sprintf(`{"tool":"str_replace","args":{"path":%q,"old_str":"missing content","new_str":"updated content"}}`, "target.txt"),
			finalResponse,
		},
	}
}

func TestRunHeadlessWithConfig_DefaultKeepsFailedToolCallSuccess(t *testing.T) {
	provider := newHeadlessFailingToolProvider(t, "final response after tool failure")

	result := RunHeadlessWithConfig(context.Background(), "edit missing content", "gpt-5.4-nano", provider, newProjectMapDisabledConfig())

	if result.Status != HeadlessStatusSuccess {
		t.Fatalf("Status = %q, want success: %+v", result.Status, result.Error)
	}
	if result.FailureReason != "" {
		t.Fatalf("FailureReason = %q, want empty", result.FailureReason)
	}
	if result.Response != "final response after tool failure" {
		t.Fatalf("Response = %q, want final response", result.Response)
	}
	if len(result.ToolCalls) != 1 {
		t.Fatalf("ToolCalls length = %d, want 1", len(result.ToolCalls))
	}
	if result.ToolCalls[0].Success {
		t.Fatalf("ToolCalls[0].Success = true, want false; output=%q", result.ToolCalls[0].Output)
	}
}

func TestRunHeadlessWithConfigOptions_FailOnToolErrorPromotesSuccessResult(t *testing.T) {
	provider := newHeadlessFailingToolProvider(t, "final response after tool failure")

	result := RunHeadlessWithConfigOptions(context.Background(), "edit missing content", "gpt-5.4-nano", provider, newProjectMapDisabledConfig(), HeadlessRunOptions{
		FailOnToolError: true,
	})

	if result.Status != HeadlessStatusError {
		t.Fatalf("Status = %q, want error", result.Status)
	}
	if result.Error == nil || result.Error.Type != HeadlessErrorTypeToolError {
		t.Fatalf("Error = %+v, want %s", result.Error, HeadlessErrorTypeToolError)
	}
	if result.FailureReason != HeadlessFailureReasonToolError {
		t.Fatalf("FailureReason = %q, want %q", result.FailureReason, HeadlessFailureReasonToolError)
	}
	if result.RecommendedExitCode != 1 {
		t.Fatalf("RecommendedExitCode = %d, want legacy error code 1", result.RecommendedExitCode)
	}
	if result.Response != "final response after tool failure" {
		t.Fatalf("Response = %q, want preserved final response", result.Response)
	}
	if len(result.ToolCalls) != 1 || result.ToolCalls[0].Success {
		t.Fatalf("ToolCalls = %+v, want one failed tool call", result.ToolCalls)
	}
	if result.Tokens == nil {
		t.Fatal("Tokens = nil, want preserved token usage")
	}
	if result.Tokens.Input != 100 || result.Tokens.Cached != 20 || result.Tokens.Output != 30 || result.Tokens.Thinking != 5 || result.Tokens.Total != 135 {
		t.Fatalf("Tokens = %+v, want input=100 cached=20 output=30 thinking=5 total=135", result.Tokens)
	}
	if result.Cost <= 0 {
		t.Fatalf("Cost = %f, want preserved positive cost", result.Cost)
	}
}

func TestRunHeadlessWithConfigOptions_FailOnToolErrorDoesNotOverrideToolLoopLimit(t *testing.T) {
	provider := newHeadlessFailingToolProvider(t, "unused final response")
	cfg := newProjectMapDisabledConfig()
	cfg.General.ToolLoopLimit = 1

	result := RunHeadlessWithConfigOptions(context.Background(), "edit missing content", "gpt-5.4-nano", provider, cfg, HeadlessRunOptions{
		FailOnToolError: true,
	})

	if result.Status != HeadlessStatusError {
		t.Fatalf("Status = %q, want error", result.Status)
	}
	if result.Error == nil || result.Error.Type != HeadlessErrorTypeToolLoopLimit {
		t.Fatalf("Error = %+v, want %s", result.Error, HeadlessErrorTypeToolLoopLimit)
	}
	if result.FailureReason != HeadlessFailureReasonToolLoopLimit {
		t.Fatalf("FailureReason = %q, want %q", result.FailureReason, HeadlessFailureReasonToolLoopLimit)
	}
	if len(result.ToolCalls) != 1 || result.ToolCalls[0].Success {
		t.Fatalf("ToolCalls = %+v, want one failed tool call before loop limit", result.ToolCalls)
	}
	if !strings.Contains(result.Error.Message, "tool loop limit") {
		t.Fatalf("Error.Message = %q, want tool loop limit message", result.Error.Message)
	}
}
