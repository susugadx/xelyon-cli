package agent

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/cost"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func TestRunHeadless_ClearsInheritedActiveContextWhenRuntimeGateOff(t *testing.T) {
	provider := &headlessActiveContextProbeProvider{}
	ctx := api.WithActiveContextBlocks(context.Background(), []api.ActiveContextBlock{{
		Name:    currentTaskStateActiveContextName,
		Content: "parent task state",
	}})

	result := RunHeadlessWithConfig(ctx, "hello", "gpt-5.4", provider, config.DefaultConfig())

	if result.Status != "success" {
		t.Fatalf("RunHeadlessWithConfig() status = %q, want success: %v", result.Status, result.Error)
	}
	if provider.activeContextBlocks != 0 {
		t.Fatalf("active context blocks sent to child request = %d, want 0", provider.activeContextBlocks)
	}
}

func TestRunHeadlessWithConfig_CollectsTokenUsageAndCost(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	provider := &headlessUsageProvider{}
	result := RunHeadlessWithConfig(context.Background(), "probe", "gpt-5.4-nano", provider, newProjectMapDisabledConfig())
	if result.Status != "success" {
		t.Fatalf("result.Status = %q, want success", result.Status)
	}
	if result.Tokens == nil {
		t.Fatal("result.Tokens = nil, want usage summary")
	}
	if result.Tokens.Input != 1000 {
		t.Fatalf("result.Tokens.Input = %d, want 1000", result.Tokens.Input)
	}
	if result.Tokens.Cached != 200 {
		t.Fatalf("result.Tokens.Cached = %d, want 200", result.Tokens.Cached)
	}
	if result.Tokens.Output != 300 {
		t.Fatalf("result.Tokens.Output = %d, want 300", result.Tokens.Output)
	}
	if result.Tokens.Thinking != 50 {
		t.Fatalf("result.Tokens.Thinking = %d, want 50", result.Tokens.Thinking)
	}
	if result.Tokens.Total != 1350 {
		t.Fatalf("result.Tokens.Total = %d, want 1350", result.Tokens.Total)
	}

	expectedCost := cost.CalculateRequestCostWithCache("openai", "gpt-5.4-nano", api.Usage{
		InputTokens:       1000,
		CachedInputTokens: 200,
		OutputTokens:      300,
		ThinkingTokens:    50,
	})
	if result.Cost != expectedCost {
		t.Fatalf("result.Cost = %f, want %f", result.Cost, expectedCost)
	}
	if result.PricingUnavailable {
		t.Fatal("result.PricingUnavailable = true, want false for known pricing")
	}
}

func TestRunHeadlessWithConfig_AttachesArgsInputMetadata(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	query := "日本語 prompt"
	provider := &headlessToolSetProbeProvider{}
	result := RunHeadlessWithConfig(context.Background(), query, "gpt-5.4", provider, newProjectMapDisabledConfig())
	if result.Status != HeadlessStatusSuccess {
		t.Fatalf("result.Status = %q, want success", result.Status)
	}
	if result.Input == nil {
		t.Fatal("result.Input = nil, want args metadata")
	}
	if result.Input.Source != HeadlessInputSourceArgs {
		t.Fatalf("result.Input.Source = %q, want %q", result.Input.Source, HeadlessInputSourceArgs)
	}
	if result.Input.Bytes != len([]byte(query)) {
		t.Fatalf("result.Input.Bytes = %d, want %d", result.Input.Bytes, len([]byte(query)))
	}
	if result.Input.PromptFile != "" {
		t.Fatalf("result.Input.PromptFile = %q, want empty", result.Input.PromptFile)
	}
}

func TestRunHeadlessWithConfig_CollectsWebSearchObservation(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	provider := &headlessWebSearchUsageProvider{}
	result := RunHeadlessWithConfig(context.Background(), "probe", "kimi-k2.6", provider, newProjectMapDisabledConfig())
	if result.Status != "success" {
		t.Fatalf("result.Status = %q, want success", result.Status)
	}
	if result.WebSearch == nil {
		t.Fatal("result.WebSearch = nil, want web search observation")
	}
	if result.WebSearch.Calls != 1 {
		t.Fatalf("result.WebSearch.Calls = %d, want 1", result.WebSearch.Calls)
	}
	if result.WebSearch.FeeEstimate != 0.005 {
		t.Fatalf("result.WebSearch.FeeEstimate = %f, want 0.005", result.WebSearch.FeeEstimate)
	}
	if result.WebSearch.ResultTokens != 222 {
		t.Fatalf("result.WebSearch.ResultTokens = %d, want 222", result.WebSearch.ResultTokens)
	}
	if result.Tokens == nil || result.Tokens.Total != 0 {
		t.Fatalf("result.Tokens = %+v, want no token double count", result.Tokens)
	}
	if result.Cost != 0.005 {
		t.Fatalf("result.Cost = %f, want web search fee 0.005", result.Cost)
	}
}

func TestRunHeadlessWithConfig_GeminiWebSearchUsageIsTokenUsage(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg := newProjectMapDisabledConfig()
	provider := &headlessGeminiWebSearchUsageProvider{}
	model := "gemini-3.1-pro-preview-customtools"
	result := RunHeadlessWithConfig(context.Background(), "probe", model, provider, cfg)
	if result.Status != "success" {
		t.Fatalf("result.Status = %q, want success", result.Status)
	}
	if result.Tokens == nil {
		t.Fatal("result.Tokens = nil, want Gemini usage summary")
	}
	if result.Tokens.Input != 17 || result.Tokens.Output != 5 || result.Tokens.Thinking != 3 || result.Tokens.Cached != 4 || result.Tokens.Total != 25 {
		t.Fatalf("result.Tokens = %+v, want input=17 cached=4 output=5 thinking=3 total=25", result.Tokens)
	}
	if result.WebSearch != nil {
		t.Fatalf("result.WebSearch = %+v, want nil because Gemini native web search reports token usage, not call-fee usage", result.WebSearch)
	}
	expectedCost := cost.CalculateRequestCostWithCacheForConfig(cfg, "gemini", model, api.Usage{
		InputTokens:       17,
		OutputTokens:      5,
		ThinkingTokens:    3,
		CachedInputTokens: 4,
	})
	if result.Cost != expectedCost {
		t.Fatalf("result.Cost = %f, want Gemini token cost %f", result.Cost, expectedCost)
	}
	if result.PricingUnavailable {
		t.Fatal("result.PricingUnavailable = true, want false for known Gemini pricing")
	}
}

func TestRunHeadlessWithConfig_UnknownPricing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	provider := &headlessUsageProvider{}
	result := RunHeadlessWithConfig(context.Background(), "probe", "unknown-model", provider, newProjectMapDisabledConfig())
	if result.Status != "success" {
		t.Fatalf("result.Status = %q, want success", result.Status)
	}
	if !result.PricingUnavailable {
		t.Fatalf("result.PricingUnavailable = false, want true")
	}
	if result.Cost != 0 {
		t.Fatalf("result.Cost = %f, want 0 for unknown pricing", result.Cost)
	}
}

func TestRunHeadlessWithConfig_ProjectMapAddsQueryFocusOverlay(t *testing.T) {
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("ripgrep (rg) not available")
	}

	t.Setenv("HOME", t.TempDir())

	root := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })
	markProjectMapTestRoot(t, root)

	nested := filepath.Join(root, "internal", "agent", "compress.go")
	if err := os.MkdirAll(filepath.Dir(nested), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(nested, []byte("package agent\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	provider := &headlessToolSetProbeProvider{}

	result := RunHeadlessWithConfig(context.Background(), "internal/agent/compress.go を見て", "gpt-5.4", provider, cfg)
	if result.Status != "success" {
		t.Fatalf("result.Status = %q, want success", result.Status)
	}
	if !strings.Contains(provider.systemPrompt, "## Project Map") {
		t.Fatalf("expected stable project map section in headless prompt:\n%s", provider.systemPrompt)
	}
	if !strings.Contains(provider.systemPrompt, "<project_map_data>") {
		t.Fatalf("expected project map data wrapper in headless prompt:\n%s", provider.systemPrompt)
	}
	if !strings.Contains(provider.systemPrompt, "Focus files for current task:") {
		t.Fatalf("expected focus overlay in headless prompt:\n%s", provider.systemPrompt)
	}
	if !strings.Contains(provider.systemPrompt, "internal/agent/compress.go") {
		t.Fatalf("expected headless system prompt to include query focus file:\n%s", provider.systemPrompt)
	}
}
