package agent

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestCustomTokenThreshold_TriggersBeforePercentage(t *testing.T) {
	provider := &compressionTestProvider{name: "openai", summary: "compressed summary"}
	cfg := config.DefaultConfig()
	cfg.Compression.TokenThreshold = 100000
	cfg.Compression.TriggerPercent = 99
	cfg.Compression.KeepRecent = 1
	cfg.Compression.PreferCompactAPI = false

	agent, out := newCompressionTestAgent(t, provider, "gpt-5.4", cfg)
	agent.History = oversizedCompressionHistory()

	beforePercentage := agent.GetTokenUsagePercentage()
	if beforePercentage >= 99 {
		t.Fatalf("GetTokenUsagePercentage() = %f, want below percentage threshold", beforePercentage)
	}

	if !agent.maybeAutoCompress() {
		t.Fatal("maybeAutoCompress() = false, want true when custom threshold is exceeded")
	}
	if provider.chatCalls != 1 {
		t.Fatalf("ChatWithTools call count = %d, want 1", provider.chatCalls)
	}
	if !strings.Contains(out.String(), "custom threshold") {
		t.Fatalf("output %q does not mention custom threshold", out.String())
	}
}

func TestMaybeAutoCompress_UsesProviderFacingHistoryReductionBudget(t *testing.T) {
	provider := &compressionTestProvider{name: "openai", summary: "compressed summary"}
	cfg := config.DefaultConfig()
	cfg.Compression.TokenThreshold = 20000
	cfg.Compression.TriggerPercent = 99
	cfg.Compression.KeepRecent = 1
	cfg.Compression.PreferCompactAPI = false

	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", cfg)
	callID := "call_budget_old"
	agent.Runtime.Options.EnableProviderHistoryReduction = true
	agent.Runtime.TaskLedger = providerHistoryTaskLedgerWithEvidence(t,
		providerHistoryEvidenceItem{ToolName: "read_file", ToolCallID: callID, Path: providerHistoryRequestEvidencePath, StartLine: 1, EndLine: 3},
	)
	agent.History = providerHistoryReductionRequestHistory(callID, strings.Repeat("あ", 30000))

	rawTotal := agent.EstimateSystemPromptTokens() + estimateTokens(agent.CurrentModel, agent.History)
	if rawTotal < cfg.Compression.TokenThreshold {
		t.Fatalf("raw token estimate = %d, want above token threshold %d", rawTotal, cfg.Compression.TokenThreshold)
	}
	if got := agent.EstimateTokens(); got >= cfg.Compression.TokenThreshold {
		t.Fatalf("EstimateTokens() = %d, want provider-facing estimate below token threshold %d", got, cfg.Compression.TokenThreshold)
	}

	if agent.maybeAutoCompress() {
		t.Fatal("maybeAutoCompress() = true, want false when provider-facing history is below threshold")
	}
	if provider.chatCalls != 0 {
		t.Fatalf("ChatWithTools call count = %d, want 0 without unnecessary compression", provider.chatCalls)
	}
}

func TestMaybeAutoCompress_CountsProviderHistoryRehydratedEvidenceBudget(t *testing.T) {
	provider := &compressionTestProvider{name: "openai", summary: "compressed summary"}
	cfg := config.DefaultConfig()
	cfg.Compression.Enabled = true
	cfg.Compression.TriggerPercent = 99
	cfg.Compression.KeepRecent = 1
	cfg.Compression.PreferCompactAPI = false

	agent, _, _ := newProviderHistoryRehydrateContextFixture(t, activeContextOpenAIResponses)
	agent.CurrentProvider = provider
	agent.Stats = NewSessionStats("openai", agent.CurrentModel)

	baseTokens := agent.EstimateSystemPromptTokens() + agent.EstimateHistoryTokens()
	withActiveContext := agent.EstimateTokens()
	if withActiveContext <= baseTokens {
		t.Fatalf("EstimateTokens() = %d, want above base provider-facing tokens %d due to rehydrated evidence", withActiveContext, baseTokens)
	}

	cfg.Compression.TokenThreshold = baseTokens + 1
	agent.Runtime.SetConfig(cfg)

	if !agent.maybeAutoCompress() {
		t.Fatal("maybeAutoCompress() = false, want true when rehydrated evidence crosses token threshold")
	}
	if provider.chatCalls != 1 {
		t.Fatalf("ChatWithTools call count = %d, want 1 compression summary call", provider.chatCalls)
	}
}

func TestCustomTokenThreshold_BelowThresholdContinuesToStandardThreshold(t *testing.T) {
	provider := &compressionTestProvider{name: "openai", summary: "compressed summary"}
	cfg := config.DefaultConfig()
	cfg.Compression.TokenThreshold = 900000
	cfg.Compression.TriggerPercent = 80
	cfg.Compression.KeepRecent = 1
	cfg.Compression.PreferCompactAPI = false

	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", cfg)
	agent.History = []api.Message{
		{Role: "user", Content: strings.Repeat("あ", 850000)},
		{Role: "assistant", Content: "latest message"},
	}

	standardThreshold, ok := localAutoCompressionTokenThresholdForConfig(cfg, "openai", "gpt-5.4")
	if !ok {
		t.Fatal("localAutoCompressionTokenThresholdForConfig() ok = false, want true")
	}
	currentTokens := agent.EstimateTokens()
	if currentTokens < standardThreshold || currentTokens >= cfg.Compression.TokenThreshold {
		t.Fatalf("EstimateTokens() = %d, want between standard threshold %d and token_threshold %d",
			currentTokens,
			standardThreshold,
			cfg.Compression.TokenThreshold,
		)
	}

	if !agent.maybeAutoCompress() {
		t.Fatal("maybeAutoCompress() = false, want true when standard threshold is exceeded below token_threshold")
	}
	if provider.chatCalls != 1 {
		t.Fatalf("ChatWithTools call count = %d, want 1", provider.chatCalls)
	}
}

func TestAutoCompress_OpenRouterDelegatedClaudeUsesKnownContextLimit(t *testing.T) {
	provider := &compressionTestProvider{name: "openrouter", summary: "compressed summary"}
	cfg := config.DefaultConfig()
	cfg.Compression.TriggerPercent = 80
	cfg.Compression.KeepRecent = 1
	cfg.Compression.PreferCompactAPI = false

	model := "anthropic/claude-sonnet-4.6"
	agent, _ := newCompressionTestAgent(t, provider, model, cfg)
	agent.History = []api.Message{
		{Role: "user", Content: strings.Repeat("あ", 150000)},
		{Role: "assistant", Content: "latest message"},
	}

	standardThreshold, ok := localAutoCompressionTokenThresholdForConfig(cfg, "openrouter", model)
	if !ok {
		t.Fatal("localAutoCompressionTokenThresholdForConfig() ok = false, want true")
	}
	if currentTokens := agent.EstimateTokens(); currentTokens < standardThreshold {
		t.Fatalf("EstimateTokens() = %d, want >= standard threshold %d", currentTokens, standardThreshold)
	}

	if !agent.maybeAutoCompress() {
		t.Fatal("maybeAutoCompress() = false, want true for delegated OpenRouter Claude model")
	}
	if provider.chatCalls != 1 {
		t.Fatalf("ChatWithTools call count = %d, want 1", provider.chatCalls)
	}
}

func TestDefaultTokenThresholdZero_DoesNotFallbackTo100K(t *testing.T) {
	provider := &compressionTestProvider{name: "openai", summary: "compressed summary"}
	cfg := config.DefaultConfig()
	cfg.Compression.TokenThreshold = 0
	cfg.Compression.TriggerPercent = 99
	cfg.Compression.KeepRecent = 1
	cfg.Compression.PreferCompactAPI = false

	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", cfg)
	agent.History = oversizedCompressionHistory()

	if agent.maybeAutoCompress() {
		t.Fatal("maybeAutoCompress() = true, want false when only removed 100K fallback would have triggered")
	}
	if provider.chatCalls != 0 {
		t.Fatalf("ChatWithTools call count = %d, want 0", provider.chatCalls)
	}
}

func TestProviderThresholdOverrideSuppressesStandardThreshold(t *testing.T) {
	provider := &compressionTestProvider{name: "deepseek", summary: "compressed summary"}
	cfg := config.DefaultConfig()
	cfg.Compression.KeepRecent = 1
	cfg.Compression.PreferCompactAPI = false
	cfg.Compression.ProviderThresholds["deepseek"] = 900000

	agent, _ := newCompressionTestAgent(t, provider, "deepseek-v4-flash", cfg)
	agent.History = []api.Message{
		{Role: "user", Content: strings.Repeat("あ", 700000)},
		{Role: "assistant", Content: "latest message"},
	}

	currentTokens := agent.EstimateTokens()
	if currentTokens < 616000 || currentTokens >= 900000 {
		t.Fatalf("EstimateTokens() = %d, want between standard threshold and provider override", currentTokens)
	}

	if agent.maybeAutoCompress() {
		t.Fatal("maybeAutoCompress() = true, want false when explicit provider threshold is not reached")
	}
	if provider.chatCalls != 0 {
		t.Fatalf("ChatWithTools call count = %d, want 0", provider.chatCalls)
	}
}

func TestProviderThresholdOverridePrecedesThresholdTokens(t *testing.T) {
	provider := &compressionTestProvider{name: "deepseek", summary: "compressed summary"}
	cfg := config.DefaultConfig()
	cfg.Compression.KeepRecent = 1
	cfg.Compression.PreferCompactAPI = false
	cfg.Compression.ThresholdTokens = 900000
	cfg.Compression.ProviderThresholds["deepseek"] = 600000

	agent, out := newCompressionTestAgent(t, provider, "deepseek-v4-flash", cfg)
	agent.History = []api.Message{
		{Role: "user", Content: strings.Repeat("あ", 700000)},
		{Role: "assistant", Content: "latest message"},
	}

	currentTokens := agent.EstimateTokens()
	if currentTokens < 600000 || currentTokens >= cfg.Compression.ThresholdTokens {
		t.Fatalf("EstimateTokens() = %d, want between provider threshold and threshold_tokens", currentTokens)
	}

	if !agent.maybeAutoCompress() {
		t.Fatal("maybeAutoCompress() = false, want true when provider threshold is exceeded below threshold_tokens")
	}
	if provider.chatCalls != 1 {
		t.Fatalf("ChatWithTools call count = %d, want 1", provider.chatCalls)
	}
	if !strings.Contains(out.String(), "custom provider threshold") {
		t.Fatalf("output %q does not mention provider threshold", out.String())
	}
}

func TestProviderThresholdOverridePrecedesClaudeCompactionSkip(t *testing.T) {
	provider := &compressionTestProvider{name: "claude", summary: "compressed summary", supportsClaudeCompaction: true}
	cfg := config.DefaultConfig()
	cfg.Compression.KeepRecent = 1
	cfg.Compression.PreferCompactAPI = false
	cfg.Compression.ProviderThresholds["claude"] = 1

	agent, _ := newCompressionTestAgent(t, provider, "claude-sonnet-4-6", cfg)
	agent.History = oversizedCompressionHistory()

	if !agent.maybeAutoCompress() {
		t.Fatal("maybeAutoCompress() = false, want true when explicit provider threshold is exceeded")
	}
	if provider.chatCalls != 1 {
		t.Fatalf("ChatWithTools call count = %d, want 1", provider.chatCalls)
	}
}

func TestThresholdTokens_MessageIncludesTokenThresholdAndUsage(t *testing.T) {
	provider := &compressionTestProvider{name: "openai", summary: "compressed summary"}
	cfg := config.DefaultConfig()
	cfg.Compression.ThresholdTokens = 10000
	cfg.Compression.KeepRecent = 1
	cfg.Compression.PreferCompactAPI = false

	agent, out := newCompressionTestAgent(t, provider, "gpt-5.4", cfg)
	agent.History = oversizedCompressionHistory()

	if !agent.maybeAutoCompress() {
		t.Fatal("maybeAutoCompress() = false, want true when threshold_tokens is exceeded")
	}
	output := out.String()
	if !strings.Contains(output, "10K threshold") {
		t.Fatalf("output %q does not mention token threshold", output)
	}
	if !strings.Contains(output, "context used") {
		t.Fatalf("output %q does not mention context usage", output)
	}
}

func TestClaudeCompactionSkipsGenericCustomThresholds(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*config.Config)
	}{
		{
			name: "token_threshold",
			configure: func(cfg *config.Config) {
				cfg.Compression.TokenThreshold = 1
			},
		},
		{
			name: "threshold_tokens",
			configure: func(cfg *config.Config) {
				cfg.Compression.ThresholdTokens = 1
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &compressionTestProvider{name: "claude", summary: "compressed summary", supportsClaudeCompaction: true}
			cfg := config.DefaultConfig()
			cfg.Compression.KeepRecent = 1
			cfg.Compression.PreferCompactAPI = false
			tt.configure(cfg)

			agent, _ := newCompressionTestAgent(t, provider, "claude-sonnet-4-6", cfg)
			agent.History = oversizedCompressionHistory()

			if agent.maybeAutoCompress() {
				t.Fatal("maybeAutoCompress() = true, want false when Claude Compaction can handle context")
			}
			if provider.chatCalls != 0 {
				t.Fatalf("ChatWithTools call count = %d, want 0", provider.chatCalls)
			}
		})
	}
}

func TestCustomTokenThreshold_ClaudeCompactionDisabledAllowsLocalCompression(t *testing.T) {
	provider := &compressionTestProvider{name: "claude", summary: "compressed summary", supportsClaudeCompaction: true}
	cfg := config.DefaultConfig()
	cfg.Compression.ClaudeCompaction = false
	cfg.Compression.TokenThreshold = 1
	cfg.Compression.KeepRecent = 1
	cfg.Compression.PreferCompactAPI = false

	agent, _ := newCompressionTestAgent(t, provider, "claude-sonnet-4-6", cfg)
	agent.History = oversizedCompressionHistory()

	if !agent.maybeAutoCompress() {
		t.Fatal("maybeAutoCompress() = false, want true when Claude Compaction is disabled")
	}
	if provider.chatCalls != 1 {
		t.Fatalf("ChatWithTools call count = %d, want 1", provider.chatCalls)
	}
}

func TestMaybeAutoCompress_UnknownContextSkipsDefaultFallback(t *testing.T) {
	provider := &compressionTestProvider{name: "openai", summary: "compressed summary"}
	cfg := config.DefaultConfig()
	cfg.Compression.TokenThreshold = 0
	cfg.Compression.TriggerPercent = 80
	cfg.Compression.KeepRecent = 1
	cfg.Compression.PreferCompactAPI = false

	agent, out := newCompressionTestAgent(t, provider, "corp-gpt-deployment", cfg)
	agent.History = hugeCompressionHistory()

	if agent.maybeAutoCompress() {
		t.Fatal("maybeAutoCompress() = true, want false when model context window is unknown")
	}
	if provider.chatCalls != 0 {
		t.Fatalf("ChatWithTools call count = %d, want 0", provider.chatCalls)
	}
	if !strings.Contains(out.String(), "context window is unknown") {
		t.Fatalf("output %q does not mention unknown context window", out.String())
	}
}

func TestMaybeAutoCompress_UsesSessionAnthropicThresholdOverride(t *testing.T) {
	provider := &compressionTestProvider{name: "claude", summary: "compressed summary"}
	cfg := config.DefaultConfig()
	cfg.Compression.KeepRecent = 1
	cfg.Compression.PreferCompactAPI = false
	cfg.Compression.ProviderThresholds["anthropic"] = 1
	cfg.Compression.ProviderThresholds["claude"] = 100000000

	agent, _ := newCompressionTestAgent(t, provider, "claude-haiku-4-5", cfg)
	agent.ProviderConfigKey = "anthropic"
	agent.History = oversizedCompressionHistory()

	if !agent.maybeAutoCompress() {
		t.Fatal("maybeAutoCompress() = false, want true when anthropic alias threshold is exceeded")
	}
	if provider.chatCalls != 1 {
		t.Fatalf("ChatWithTools call count = %d, want 1", provider.chatCalls)
	}
}

func TestCustomTokenThreshold_RespectsResponsesServerCompactionSkip(t *testing.T) {
	provider := &compressionTestProvider{
		name:                      "openai",
		summary:                   "compressed summary",
		cachedResponseID:          true,
		responseID:                "resp_cached",
		serverCompactionLocalSkip: true,
	}
	cfg := config.DefaultConfig()
	cfg.Compression.TokenThreshold = 100000
	cfg.Compression.TriggerPercent = 99
	cfg.Compression.KeepRecent = 1
	cfg.Compression.PreferCompactAPI = false

	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", cfg)
	agent.History = hugeCompressionHistory()

	if agent.maybeAutoCompress() {
		t.Fatal("maybeAutoCompress() = true, want false when server compaction skips local compression")
	}
	if provider.chatCalls != 0 {
		t.Fatalf("ChatWithTools call count = %d, want 0", provider.chatCalls)
	}
}

func TestCustomTokenThreshold_AzureResponsesServerCompactionSkipsLocalCompression(t *testing.T) {
	provider := &compressionTestProvider{
		name:                      "azure",
		summary:                   "compressed summary",
		cachedResponseID:          true,
		responseID:                "resp_cached",
		serverCompactionLocalSkip: true,
	}
	cfg := config.DefaultConfig()
	cfg.Compression.TokenThreshold = 1
	cfg.Compression.KeepRecent = 1
	cfg.Compression.PreferCompactAPI = false

	agent, _ := newCompressionTestAgent(t, provider, "azure-gpt-5.4", cfg)
	agent.History = oversizedCompressionHistory()

	if agent.maybeAutoCompress() {
		t.Fatal("maybeAutoCompress() = true, want false when Azure request includes server compaction")
	}
	if provider.chatCalls != 0 {
		t.Fatalf("ChatWithTools call count = %d, want 0", provider.chatCalls)
	}
}

func TestCustomTokenThreshold_ServerCompactionDisabledAllowsLocalCompression(t *testing.T) {
	provider := &compressionTestProvider{
		name:                      "openai",
		summary:                   "compressed summary",
		cachedResponseID:          true,
		responseID:                "resp_cached",
		serverCompactionLocalSkip: true,
	}
	cfg := config.DefaultConfig()
	cfg.Responses.ServerCompaction.Enabled = false
	cfg.Compression.TokenThreshold = 1
	cfg.Compression.KeepRecent = 1
	cfg.Compression.PreferCompactAPI = false

	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", cfg)
	agent.History = oversizedCompressionHistory()

	if !agent.maybeAutoCompress() {
		t.Fatal("maybeAutoCompress() = false, want true when server compaction skip is disabled")
	}
	if provider.chatCalls != 1 {
		t.Fatalf("ChatWithTools call count = %d, want 1", provider.chatCalls)
	}
}

func TestCustomTokenThreshold_ServerCompactionFallbackKeepsLocalDecisionOnUnknownContext(t *testing.T) {
	provider := &compressionTestProvider{
		name:                      "openai",
		summary:                   "compressed summary",
		cachedResponseID:          true,
		responseID:                "resp_cached",
		serverCompactionLocalSkip: false,
	}
	cfg := config.DefaultConfig()
	cfg.Compression.TokenThreshold = 0
	cfg.Compression.TriggerPercent = 80
	cfg.Compression.KeepRecent = 1
	cfg.Compression.PreferCompactAPI = false
	cfg.Responses.ServerCompaction.LocalFallback = true

	agent, out := newCompressionTestAgent(t, provider, "corp-gpt-deployment", cfg)
	agent.History = hugeCompressionHistory()

	if agent.maybeAutoCompress() {
		t.Fatal("maybeAutoCompress() = true, want false when fallback keeps unknown-context local decision")
	}
	if provider.chatCalls != 0 {
		t.Fatalf("ChatWithTools call count = %d, want 0", provider.chatCalls)
	}
	if !strings.Contains(out.String(), "context window is unknown") {
		t.Fatalf("output %q does not mention unknown context window", out.String())
	}
}
