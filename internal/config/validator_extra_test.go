package config

import (
	"testing"
)

func TestApplyAutoFixes_ProviderFix(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultProvider = "invalid"

	result := ValidateConfig(cfg)
	fixCount := ApplyAutoFixes(cfg, result)

	if fixCount == 0 {
		t.Fatal("ApplyAutoFixes should fix at least 1 issue")
	}
	if cfg.DefaultProvider != "deepseek" {
		t.Errorf("DefaultProvider should be fixed to %q, got %q", "deepseek", cfg.DefaultProvider)
	}
}

func TestApplyAutoFixes_EmptyProvider(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultProvider = ""

	result := ValidateConfig(cfg)
	fixCount := ApplyAutoFixes(cfg, result)

	if fixCount == 0 {
		t.Fatal("ApplyAutoFixes should fix empty provider")
	}
	if cfg.DefaultProvider != "deepseek" {
		t.Errorf("DefaultProvider should be fixed to %q, got %q", "deepseek", cfg.DefaultProvider)
	}
}

func TestApplyAutoFixes_NumericRange(t *testing.T) {
	cfg := DefaultConfig()
	cfg.LoopDetection.Threshold = 100 // 範囲外 (1-10)
	cfg.APIRetry.Count = 100          // 範囲外 (1-10)
	cfg.APIRetry.InitialDelay = 100   // 範囲外 (1-60)
	cfg.APIRetry.MaxDelay = 1000      // 範囲外 (1-300)
	cfg.APIRetry.Timeout = 99999      // 範囲外 (30-7200)
	cfg.Diff.ContextLines = 999       // 範囲外 (0-100)

	result := ValidateConfig(cfg)
	fixCount := ApplyAutoFixes(cfg, result)

	if fixCount < 6 {
		t.Errorf("ApplyAutoFixes should fix at least 6 numeric range issues, got %d", fixCount)
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
	if cfg.APIRetry.Timeout != 3600 {
		t.Errorf("APIRetry.Timeout = %d, want 3600", cfg.APIRetry.Timeout)
	}
	if cfg.Diff.ContextLines != 10 {
		t.Errorf("Diff.ContextLines = %d, want 10", cfg.Diff.ContextLines)
	}
}

func TestApplyAutoFixes_BashSafetyLevel(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Bash.SafetyLevel = "invalid_level"

	result := ValidateConfig(cfg)
	fixCount := ApplyAutoFixes(cfg, result)

	if fixCount == 0 {
		t.Fatal("ApplyAutoFixes should fix invalid bash safety level")
	}
	if cfg.Bash.SafetyLevel != "moderate" {
		t.Errorf("Bash.SafetyLevel = %q, want %q", cfg.Bash.SafetyLevel, "moderate")
	}
}

func TestApplyAutoFixes_ToolLoopLimit(t *testing.T) {
	cfg := DefaultConfig()
	cfg.General.ToolLoopLimit = -5

	result := ValidateConfig(cfg)
	fixCount := ApplyAutoFixes(cfg, result)

	if fixCount == 0 {
		t.Fatal("ApplyAutoFixes should fix negative tool_loop_limit")
	}
	if cfg.General.ToolLoopLimit != 0 {
		t.Errorf("General.ToolLoopLimit = %d, want 0", cfg.General.ToolLoopLimit)
	}
}

func TestApplyAutoFixes_ProjectMapContextRatio(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ProjectMap.ContextRatio = 0.5 // 範囲外 (0.01-0.20)

	result := ValidateConfig(cfg)
	fixCount := ApplyAutoFixes(cfg, result)

	if fixCount == 0 {
		t.Fatal("ApplyAutoFixes should fix out-of-range context_ratio")
	}
	if cfg.ProjectMap.ContextRatio != ProjectMapContextRatioDefault {
		t.Errorf("ProjectMap.ContextRatio = %v, want %v", cfg.ProjectMap.ContextRatio, ProjectMapContextRatioDefault)
	}
}

func TestApplyAutoFixes_NoIssues(t *testing.T) {
	cfg := DefaultConfig()

	result := ValidateConfig(cfg)
	fixCount := ApplyAutoFixes(cfg, result)

	if fixCount != 0 {
		t.Errorf("ApplyAutoFixes should return 0 for valid config, got %d", fixCount)
	}
}

func TestApplyAutoFixes_NonAutoFixableIssue(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ProviderModels["invalid_provider"] = ProviderModelConfig{
		DefaultModel: "some-model",
	}

	result := ValidateConfig(cfg)

	// invalid provider_models のワーニングは CanAutoFix=false
	hasNonAutoFixable := false
	for _, issue := range result.Issues {
		if issue.Field == "provider_models.invalid_provider" && !issue.CanAutoFix {
			hasNonAutoFixable = true
		}
	}
	if !hasNonAutoFixable {
		t.Fatal("Expected non-auto-fixable issue for invalid provider in ProviderModels")
	}

	fixCount := ApplyAutoFixes(cfg, result)

	// CanAutoFix=false の問題は修正されない
	if _, ok := cfg.ProviderModels["invalid_provider"]; !ok {
		t.Error("Non-auto-fixable issue should not be modified")
	}
	if fixCount != 0 {
		t.Errorf("ApplyAutoFixes should return 0 for non-auto-fixable issues only, got %d", fixCount)
	}
}

func TestApplyAutoFixes_CompressionTriggers(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Compression.CompactionTrigger = 1        // 下限未満
	cfg.Compression.ClearToolUsesTrigger = 10000 // 下限未満

	result := ValidateConfig(cfg)
	fixCount := ApplyAutoFixes(cfg, result)

	if fixCount < 2 {
		t.Errorf("ApplyAutoFixes should fix at least 2 compression trigger issues, got %d", fixCount)
	}
	if cfg.Compression.CompactionTrigger != 150000 {
		t.Errorf("Compression.CompactionTrigger = %d, want 150000", cfg.Compression.CompactionTrigger)
	}
	if cfg.Compression.ClearToolUsesTrigger != 80000 {
		t.Errorf("Compression.ClearToolUsesTrigger = %d, want 80000", cfg.Compression.ClearToolUsesTrigger)
	}
}
