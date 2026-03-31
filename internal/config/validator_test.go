package config

import (
	"bytes"
	"io"
	"math"
	"os"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/stdio"
)

func TestIsValidProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		want     bool
	}{
		{"valid deepseek", "deepseek", true},
		{"valid openai", "openai", true},
		{"valid gemini", "gemini", true},
		{"valid claude", "claude", true},
		{"valid anthropic", "anthropic", true},
		{"valid ollama", "ollama", true},
		{"valid groq", "groq", true},
		{"valid uppercase", "DeepSeek", true},
		{"valid mixed case", "OpenAI", true},
		{"invalid provider", "gpt4", false},
		{"invalid typo", "deepseek2", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidProvider(tt.provider)
			if got != tt.want {
				t.Errorf("isValidProvider(%q) = %v, want %v", tt.provider, got, tt.want)
			}
		})
	}
}

func TestSuggestProvider(t *testing.T) {
	tests := []struct {
		name    string
		invalid string
		want    string
	}{
		{"typo deepseek2", "deepseek2", "deepseek"},
		{"typo deepseak", "deepseak", "deepseek"},
		{"gpt", "gpt", "openai"},
		{"chatgpt", "chatgpt", "openai"},
		{"google", "google", "gemini"},
		{"sonnet", "sonnet", "claude"},
		{"llama", "llama", "ollama"},
		{"unknown", "unknown", "deepseek"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := suggestProvider(tt.invalid)
			if got != tt.want {
				t.Errorf("suggestProvider(%q) = %q, want %q", tt.invalid, got, tt.want)
			}
		})
	}
}

func TestValidateConfig_ValidConfig(t *testing.T) {
	cfg := DefaultConfig()
	result := ValidateConfig(cfg)

	if !result.Valid {
		t.Errorf("ValidateConfig(DefaultConfig()) should be valid, got issues: %+v", result.Issues)
	}
	if len(result.Issues) != 0 {
		t.Errorf("ValidateConfig(DefaultConfig()) should have no issues, got %d", len(result.Issues))
	}
}

func TestValidateConfig_InvalidProvider(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultProvider = "invalid_provider"

	result := ValidateConfig(cfg)

	if result.Valid {
		t.Error("ValidateConfig should be invalid for invalid provider")
	}
	if len(result.Issues) == 0 {
		t.Error("ValidateConfig should have issues for invalid provider")
	}

	// Check that the issue is about provider
	found := false
	for _, issue := range result.Issues {
		if issue.Field == "default_provider" {
			found = true
			if issue.Severity != "error" {
				t.Errorf("Invalid provider should be error, got %s", issue.Severity)
			}
		}
	}
	if !found {
		t.Error("Should have issue for default_provider field")
	}
}

func TestValidateConfig_EmptyProvider(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultProvider = ""

	result := ValidateConfig(cfg)

	if result.Valid {
		t.Error("ValidateConfig should be invalid for empty provider")
	}

	found := false
	for _, issue := range result.Issues {
		if issue.Field == "default_provider" && issue.Severity == "error" {
			found = true
		}
	}
	if !found {
		t.Error("Should have error issue for empty default_provider")
	}
}

func TestValidateConfig_RangeValidation(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Compression.TriggerPercent = 200 // Out of range (1-100)

	result := ValidateConfig(cfg)

	found := false
	for _, issue := range result.Issues {
		if issue.Field == "compression.trigger_percent" {
			found = true
			if issue.Severity != "warning" {
				t.Errorf("Out of range should be warning, got %s", issue.Severity)
			}
		}
	}
	if !found {
		t.Error("Should have warning for out of range compression.trigger_percent")
	}
}

func TestValidateConfig_DoesNotMutateCfg(t *testing.T) {
	cfg := DefaultConfig()
	cfg.General.ToolLoopLimit = -1
	cfg.Compression.TriggerPercent = 200
	cfg.Bash.SafetyLevel = "invalid"

	// ValidateConfig 前の値を控える
	origLimit := cfg.General.ToolLoopLimit
	origPercent := cfg.Compression.TriggerPercent
	origLevel := cfg.Bash.SafetyLevel

	_ = ValidateConfig(cfg)

	// ValidateConfig は cfg を変更してはならない
	if cfg.General.ToolLoopLimit != origLimit {
		t.Errorf("ValidateConfig mutated ToolLoopLimit: %d → %d", origLimit, cfg.General.ToolLoopLimit)
	}
	if cfg.Compression.TriggerPercent != origPercent {
		t.Errorf("ValidateConfig mutated TriggerPercent: %d → %d", origPercent, cfg.Compression.TriggerPercent)
	}
	if cfg.Bash.SafetyLevel != origLevel {
		t.Errorf("ValidateConfig mutated SafetyLevel: %q → %q", origLevel, cfg.Bash.SafetyLevel)
	}
}

func TestValidateConfig_InvalidBashSafetyLevel(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Bash.SafetyLevel = "invalid"

	result := ValidateConfig(cfg)

	found := false
	for _, issue := range result.Issues {
		if issue.Field == "bash.safety_level" {
			found = true
		}
	}
	if !found {
		t.Error("Should have issue for invalid bash.safety_level")
	}
}

func TestValidateConfig_ProjectMapContextRatio(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		want  string
	}{
		{name: "too small", value: 0.005, want: "0.005"},
		{name: "too large", value: 0.5, want: "0.5"},
		{name: "nan", value: math.NaN(), want: "NaN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.ProjectMap.ContextRatio = tt.value

			result := ValidateConfig(cfg)

			found := false
			for _, issue := range result.Issues {
				if issue.Field != "project_map.context_ratio" {
					continue
				}
				found = true
				if issue.Severity != "warning" {
					t.Errorf("Severity = %s, want warning", issue.Severity)
				}
				if issue.Value != tt.want {
					t.Errorf("Value = %q, want %q", issue.Value, tt.want)
				}
				if issue.FixedValue != ProjectMapContextRatioDefault {
					t.Errorf("FixedValue = %v, want %v", issue.FixedValue, ProjectMapContextRatioDefault)
				}
			}
			if !found {
				t.Fatal("Should have issue for project_map.context_ratio")
			}
		})
	}
}

func TestApplyAutoFixes(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultProvider = "invalid"
	cfg.Compression.TriggerPercent = 200
	cfg.ProjectMap.ContextRatio = 0.5

	result := ValidateConfig(cfg)
	fixCount := ApplyAutoFixes(cfg, result)

	if fixCount < 3 {
		t.Errorf("ApplyAutoFixes should fix at least 3 issues, got %d", fixCount)
	}

	// Verify fixes were applied
	if cfg.DefaultProvider != "deepseek" {
		t.Errorf("DefaultProvider should be fixed to 'deepseek', got %q", cfg.DefaultProvider)
	}
	if cfg.Compression.TriggerPercent != 80 {
		t.Errorf("Compression.TriggerPercent should be fixed to 80, got %d", cfg.Compression.TriggerPercent)
	}
	if cfg.ProjectMap.ContextRatio != ProjectMapContextRatioDefault {
		t.Errorf("ProjectMap.ContextRatio should be fixed to %v, got %v", ProjectMapContextRatioDefault, cfg.ProjectMap.ContextRatio)
	}
}

func TestValidationResult_HasErrors(t *testing.T) {
	result := ValidationResult{
		Valid: false,
		Issues: []ValidationIssue{
			{Severity: "warning"},
			{Severity: "error"},
		},
	}

	if !result.HasErrors() {
		t.Error("HasErrors should return true when there are error issues")
	}

	result.Issues = []ValidationIssue{
		{Severity: "warning"},
	}
	if result.HasErrors() {
		t.Error("HasErrors should return false when there are only warnings")
	}
}

func TestValidationResult_HasWarnings(t *testing.T) {
	result := ValidationResult{
		Issues: []ValidationIssue{
			{Severity: "warning"},
		},
	}

	if !result.HasWarnings() {
		t.Error("HasWarnings should return true when there are warning issues")
	}

	result.Issues = []ValidationIssue{
		{Severity: "error"},
	}
	if result.HasWarnings() {
		t.Error("HasWarnings should return false when there are only errors")
	}
}

func TestValidateConfig_ProviderModelsValidation(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ProviderModels["invalid_provider"] = ProviderModelConfig{
		DefaultModel: "some-model",
	}

	result := ValidateConfig(cfg)

	found := false
	for _, issue := range result.Issues {
		if issue.Field == "provider_models.invalid_provider" {
			found = true
			if issue.Severity != "warning" {
				t.Errorf("Invalid provider in ProviderModels should be warning, got %s", issue.Severity)
			}
		}
	}
	if !found {
		t.Error("Should have warning for invalid provider in ProviderModels")
	}
}

func TestPrintValidationWarnings_UsesConfiguredDefaultOutput(t *testing.T) {
	t.Cleanup(func() {
		stdio.SetDefaults(nil, nil, nil)
	})

	var out bytes.Buffer
	stdio.SetDefaults(nil, &out, io.Discard)

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = oldStdout
	})

	PrintValidationWarnings(ValidationResult{
		Issues: []ValidationIssue{{
			Field:      "default_provider",
			Message:    "required",
			Severity:   "error",
			Suggestion: "deepseek",
		}},
	})
	_ = w.Close()

	var leaked bytes.Buffer
	_, _ = io.Copy(&leaked, r)

	if !bytes.Contains(out.Bytes(), []byte("default_provider")) {
		t.Fatalf("expected configured default output to receive warnings, got %q", out.String())
	}
	if leaked.Len() != 0 {
		t.Fatalf("expected no process stdout leak, got %q", leaked.String())
	}
}

func TestContains(t *testing.T) {
	slice := []string{"a", "b", "c"}

	if !contains(slice, "a") {
		t.Error("contains should return true for 'a'")
	}
	if !contains(slice, "b") {
		t.Error("contains should return true for 'b'")
	}
	if contains(slice, "d") {
		t.Error("contains should return false for 'd'")
	}
	if contains(nil, "a") {
		t.Error("contains should return false for nil slice")
	}
}
