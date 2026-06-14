package config

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	if cfg.DefaultProvider != "deepseek" {
		t.Errorf("DefaultProvider = %v, want deepseek", cfg.DefaultProvider)
	}
	if cfg.DefaultModel != "deepseek-v4-flash" {
		t.Errorf("DefaultModel = %v, want deepseek-v4-flash", cfg.DefaultModel)
	}
	if cfg.ProviderModels == nil {
		t.Fatal("ProviderModels is nil")
	}
	if len(cfg.CommandAliases) != 0 {
		t.Fatalf("CommandAliases default = %#v, want empty; catalog aliases are the supported source of truth", cfg.CommandAliases)
	}

	expectedProviders := llmcatalog.ProviderKeys(false)
	for _, provider := range expectedProviders {
		if _, ok := cfg.ProviderModels[provider]; !ok {
			t.Errorf("ProviderModels missing provider: %s", provider)
		}
	}

	if cfg.Compression.Enabled != true {
		t.Error("Compression.Enabled should default to true")
	}
	if cfg.Compression.ThresholdTokens != 0 {
		t.Errorf("Compression.ThresholdTokens = %d, want 0 (percentage-based)", cfg.Compression.ThresholdTokens)
	}
	if cfg.Compression.TriggerPercent != 80 {
		t.Errorf("Compression.TriggerPercent = %d, want 80", cfg.Compression.TriggerPercent)
	}
	if cfg.Compression.TokenThreshold != 0 {
		t.Errorf("Compression.TokenThreshold = %d, want 0", cfg.Compression.TokenThreshold)
	}
	if cfg.Compression.Model != "" {
		t.Errorf("Compression.Model = %q, want empty string", cfg.Compression.Model)
	}
	if cfg.Compression.KeepRecent != 20 {
		t.Errorf("Compression.KeepRecent = %d, want 20", cfg.Compression.KeepRecent)
	}
	if cfg.ProviderHistoryReduction.Mode != ProviderHistoryReductionModeDryRun {
		t.Errorf("ProviderHistoryReduction.Mode = %q, want dry_run", cfg.ProviderHistoryReduction.Mode)
	}
	if !cfg.ProviderHistoryReduction.RehydrateContext {
		t.Error("ProviderHistoryReduction.RehydrateContext should default to true")
	}
	if got := cfg.ProviderHistoryReduction.RawOutputArtifacts.Mode; got != ProviderHistoryRawOutputArtifactsModeDryRun {
		t.Errorf("RawOutputArtifacts.Mode = %q, want dry_run", got)
	}
	if got := cfg.ProviderHistoryReduction.RawOutputArtifacts.MaxArtifactBytes; got != defaultRawOutputArtifactMaxBytes {
		t.Errorf("RawOutputArtifacts.MaxArtifactBytes = %d, want %d", got, defaultRawOutputArtifactMaxBytes)
	}
	if got := cfg.ProviderHistoryReduction.RawOutputArtifacts.SessionQuotaBytes; got != defaultRawOutputArtifactSessionQuotaBytes {
		t.Errorf("RawOutputArtifacts.SessionQuotaBytes = %d, want %d", got, defaultRawOutputArtifactSessionQuotaBytes)
	}
	if got := cfg.ProviderHistoryReduction.RawOutputArtifacts.ChunkBytes; got != defaultRawOutputArtifactChunkBytes {
		t.Errorf("RawOutputArtifacts.ChunkBytes = %d, want %d", got, defaultRawOutputArtifactChunkBytes)
	}
	if got := cfg.ProviderHistoryReduction.RawOutputArtifacts.ActiveContextBudgetTokens; got != defaultRawOutputArtifactActiveContextBudgetTokens {
		t.Errorf("RawOutputArtifacts.ActiveContextBudgetTokens = %d, want %d", got, defaultRawOutputArtifactActiveContextBudgetTokens)
	}
	if got := cfg.ProviderHistoryReduction.RawOutputArtifacts.ActiveContextBudgetMaxTokens; got != defaultRawOutputArtifactActiveContextBudgetMax {
		t.Errorf("RawOutputArtifacts.ActiveContextBudgetMaxTokens = %d, want %d", got, defaultRawOutputArtifactActiveContextBudgetMax)
	}
	if got := cfg.ProviderHistoryReduction.RawOutputArtifacts.Retention; got != ProviderHistoryRawOutputArtifactsRetentionSession {
		t.Errorf("RawOutputArtifacts.Retention = %q, want session", got)
	}
	if cfg.Compression.PreferCompactAPI != true {
		t.Error("Compression.PreferCompactAPI should default to true")
	}
	if cfg.Compression.ClaudeCompaction != true {
		t.Error("Compression.ClaudeCompaction should default to true")
	}
	if cfg.Compression.CompactionTrigger != 150000 {
		t.Errorf("Compression.CompactionTrigger = %d, want 150000", cfg.Compression.CompactionTrigger)
	}
	if cfg.Compression.ClearToolUses != true {
		t.Error("Compression.ClearToolUses should default to true")
	}
	if cfg.Compression.ClearToolUsesTrigger != 80000 {
		t.Errorf("Compression.ClearToolUsesTrigger = %d, want 80000", cfg.Compression.ClearToolUsesTrigger)
	}
	if cfg.Compression.ClearToolInputs {
		t.Error("Compression.ClearToolInputs should default to false")
	}
	if cfg.LoopDetection.Threshold != 3 {
		t.Errorf("LoopDetection.Threshold = %d, want 3", cfg.LoopDetection.Threshold)
	}
	if cfg.APIRetry.Count != 3 {
		t.Errorf("APIRetry.Count = %d, want 3", cfg.APIRetry.Count)
	}
	if cfg.APIRetry.InitialDelay != 1 {
		t.Errorf("APIRetry.InitialDelay = %d, want 1", cfg.APIRetry.InitialDelay)
	}
	if cfg.APIRetry.MaxDelay != 30 {
		t.Errorf("APIRetry.MaxDelay = %d, want 30", cfg.APIRetry.MaxDelay)
	}
	if cfg.Diff.ContextLines != 10 {
		t.Errorf("Diff.ContextLines = %d, want 10", cfg.Diff.ContextLines)
	}
	if cfg.General.ToolLoopLimit != 0 {
		t.Errorf("General.ToolLoopLimit = %d, want 0", cfg.General.ToolLoopLimit)
	}
	if !cfg.ProjectMap.Enabled {
		t.Error("ProjectMap.Enabled should default to true")
	}
	if cfg.ProjectMap.ContextRatio != ProjectMapContextRatioDefault {
		t.Errorf("ProjectMap.ContextRatio = %f, want %f", cfg.ProjectMap.ContextRatio, ProjectMapContextRatioDefault)
	}
	if !cfg.SubAgent.Enabled {
		t.Error("SubAgent.Enabled should default to true")
	}
	if cfg.SubAgent.DefaultModel != "gpt-5.4-mini" {
		t.Errorf("SubAgent.DefaultModel = %q, want gpt-5.4-mini", cfg.SubAgent.DefaultModel)
	}
	if cfg.SubAgent.DefaultEffort != "" {
		t.Errorf("SubAgent.DefaultEffort = %q, want empty string", cfg.SubAgent.DefaultEffort)
	}
	if cfg.SubAgent.MaxConcurrent != 1 {
		t.Errorf("SubAgent.MaxConcurrent = %d, want 1", cfg.SubAgent.MaxConcurrent)
	}
	if cfg.AgentInstructions.Project.Mode != "always" {
		t.Errorf("AgentInstructions.Project.Mode = %q, want always", cfg.AgentInstructions.Project.Mode)
	}
	if !cfg.AgentInstructions.Global.Enabled {
		t.Error("AgentInstructions.Global.Enabled should default to true")
	}
	if len(cfg.AgentInstructions.Project.Files) == 0 {
		t.Error("AgentInstructions.Project.Files should have defaults")
	}
	if got, want := cfg.AgentInstructions.Project.Files, []string{"AGENTS.md"}; len(got) != len(want) || got[0] != want[0] {
		t.Errorf("AgentInstructions.Project.Files = %#v, want %#v", got, want)
	}
	if len(cfg.AgentInstructions.Global.Files) == 0 {
		t.Error("AgentInstructions.Global.Files should have defaults")
	}
	if got, want := cfg.AgentInstructions.Global.Files, []string{"~/.xelyon/AGENTS.md"}; len(got) != len(want) || got[0] != want[0] {
		t.Errorf("AgentInstructions.Global.Files = %#v, want %#v", got, want)
	}
	if cfg.AgentInstructions.MaxFileBytes != 20000 {
		t.Errorf("AgentInstructions.MaxFileBytes = %d, want 20000", cfg.AgentInstructions.MaxFileBytes)
	}
	if cfg.AgentInstructions.MaxTotalBytes != 60000 {
		t.Errorf("AgentInstructions.MaxTotalBytes = %d, want 60000", cfg.AgentInstructions.MaxTotalBytes)
	}
	if !cfg.Skills.Router.Enabled {
		t.Error("Skills.Router.Enabled should default to true")
	}
	if cfg.Skills.Router.Activation != SkillsRouterActivationHint {
		t.Errorf("Skills.Router.Activation = %q, want hint", cfg.Skills.Router.Activation)
	}
	if !cfg.Skills.Router.UsageLedger {
		t.Error("Skills.Router.UsageLedger should default to true")
	}
	if cfg.Skills.Router.UsageRetentionDays != 30 {
		t.Errorf("Skills.Router.UsageRetentionDays = %d, want 30", cfg.Skills.Router.UsageRetentionDays)
	}
}
