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
	cfg.Compression.TriggerPercent = 200 // 範囲外 (1-100)
	cfg.Compression.KeepRecent = 999     // 範囲外 (1-100)

	result := ValidateConfig(cfg)
	fixCount := ApplyAutoFixes(cfg, result)

	if fixCount < 2 {
		t.Errorf("ApplyAutoFixes should fix at least 2 numeric range issues, got %d", fixCount)
	}
	if cfg.Compression.TriggerPercent != 80 {
		t.Errorf("Compression.TriggerPercent = %d, want 80", cfg.Compression.TriggerPercent)
	}
	if cfg.Compression.KeepRecent != 10 {
		t.Errorf("Compression.KeepRecent = %d, want 10", cfg.Compression.KeepRecent)
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

// TestApplyAutoFixes_CompressionTriggers は内部化されたため、
// trigger_percent/keep_recent の範囲チェックのみ検証。
// CompactionTrigger/ClearToolUsesTrigger は内部既定値として validation 対象外。
