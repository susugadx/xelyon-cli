package config

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDefaultConfig_ReviewConfigUsesCurrentProviderByDefault(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Review.Provider != "" {
		t.Fatalf("Review.Provider = %q, want empty", cfg.Review.Provider)
	}
	if cfg.Review.Model != "" {
		t.Fatalf("Review.Model = %q, want empty", cfg.Review.Model)
	}
	if cfg.Review.Thinking.Mode != ReviewThinkingModeInherit {
		t.Fatalf("Review.Thinking.Mode = %q, want inherit", cfg.Review.Thinking.Mode)
	}
	if cfg.Review.Thinking.Level != "" {
		t.Fatalf("Review.Thinking.Level = %q, want empty", cfg.Review.Thinking.Level)
	}
	if cfg.Review.WebSearchEvidence.Enabled {
		t.Fatal("Review.WebSearchEvidence.Enabled = true, want false")
	}
	if cfg.Review.WebSearchEvidence.MaxQueries != 3 {
		t.Fatalf("Review.WebSearchEvidence.MaxQueries = %d, want 3", cfg.Review.WebSearchEvidence.MaxQueries)
	}
	if cfg.Review.WebSearchEvidence.MaxResultsPerQuery != 3 {
		t.Fatalf("Review.WebSearchEvidence.MaxResultsPerQuery = %d, want 3", cfg.Review.WebSearchEvidence.MaxResultsPerQuery)
	}
}

func TestReviewConfigYAMLRoundTrip(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Review = ReviewConfig{
		Provider: "claude",
		Model:    "claude-sonnet-4-6",
		Thinking: ReviewThinkingConfig{
			Mode:  ReviewThinkingModeOn,
			Level: "high",
		},
		WebSearchEvidence: ReviewWebSearchEvidenceConfig{
			Enabled:            true,
			MaxQueries:         2,
			MaxResultsPerQuery: 4,
		},
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("yaml.Marshal() error = %v", err)
	}

	var got Config
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	if got.Review.Provider != "claude" {
		t.Fatalf("Review.Provider = %q, want claude", got.Review.Provider)
	}
	if got.Review.Model != "claude-sonnet-4-6" {
		t.Fatalf("Review.Model = %q, want claude-sonnet-4-6", got.Review.Model)
	}
	if got.Review.Thinking.Mode != ReviewThinkingModeOn {
		t.Fatalf("Review.Thinking.Mode = %q, want on", got.Review.Thinking.Mode)
	}
	if got.Review.Thinking.Level != "high" {
		t.Fatalf("Review.Thinking.Level = %q, want high", got.Review.Thinking.Level)
	}
	if !got.Review.WebSearchEvidence.Enabled {
		t.Fatal("Review.WebSearchEvidence.Enabled = false, want true")
	}
	if got.Review.WebSearchEvidence.MaxQueries != 2 {
		t.Fatalf("Review.WebSearchEvidence.MaxQueries = %d, want 2", got.Review.WebSearchEvidence.MaxQueries)
	}
	if got.Review.WebSearchEvidence.MaxResultsPerQuery != 4 {
		t.Fatalf("Review.WebSearchEvidence.MaxResultsPerQuery = %d, want 4", got.Review.WebSearchEvidence.MaxResultsPerQuery)
	}
}

func TestApplyDefaults_ReviewThinkingModeDefaultsToInherit(t *testing.T) {
	tests := []struct {
		name string
		mode ReviewThinkingMode
	}{
		{name: "unset", mode: ""},
		{name: "blank", mode: "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{}
			cfg.Review.Thinking.Mode = tt.mode

			applyDefaults(cfg)

			if cfg.Review.Thinking.Mode != ReviewThinkingModeInherit {
				t.Fatalf("Review.Thinking.Mode = %q, want inherit", cfg.Review.Thinking.Mode)
			}
		})
	}
}

func TestValidateConfig_ReviewModelRequiresProvider(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Review.Model = "claude-sonnet-4-6"

	result := ValidateConfig(cfg)
	if result.Valid {
		t.Fatal("ValidateConfig() valid = true, want false")
	}
	if !validationIssuesContain(result.Issues, "review.model", "review.provider") {
		t.Fatalf("issues = %#v, want review.model error requiring review.provider", result.Issues)
	}
}

func TestValidateConfig_ReviewProviderAllowsAlias(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Review.Provider = "anthropic"
	cfg.Review.Model = "claude-sonnet-4-6"

	result := ValidateConfig(cfg)
	if !result.Valid {
		t.Fatalf("ValidateConfig() valid = false, issues = %#v", result.Issues)
	}
}

func TestValidateConfig_ReviewProviderRejectsUnknown(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Review.Provider = "unknown-provider"

	result := ValidateConfig(cfg)
	if result.Valid {
		t.Fatal("ValidateConfig() valid = true, want false")
	}
	if !validationIssuesContain(result.Issues, "review.provider", "無効") {
		t.Fatalf("issues = %#v, want review.provider invalid error", result.Issues)
	}
}

func TestValidateConfig_ReviewThinkingRejectsInvalidMode(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Review.Thinking.Mode = "auto"

	result := ValidateConfig(cfg)
	if result.Valid {
		t.Fatal("ValidateConfig() valid = true, want false")
	}
	if !validationIssuesContain(result.Issues, "review.thinking.mode", "inherit/off/on") {
		t.Fatalf("issues = %#v, want review.thinking.mode invalid error", result.Issues)
	}
}

func TestValidateConfig_ReviewThinkingRejectsInvalidLevel(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Review.Thinking.Level = "max"

	result := ValidateConfig(cfg)
	if result.Valid {
		t.Fatal("ValidateConfig() valid = true, want false")
	}
	if !validationIssuesContain(result.Issues, "review.thinking.level", "low/medium/high/xhigh") {
		t.Fatalf("issues = %#v, want review.thinking.level invalid error", result.Issues)
	}
}

func TestValidateConfig_ReviewThinkingAllowsEmptyLevel(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Review.Thinking = ReviewThinkingConfig{
		Mode:  ReviewThinkingModeOn,
		Level: "",
	}

	result := ValidateConfig(cfg)
	if !result.Valid {
		t.Fatalf("ValidateConfig() valid = false, issues = %#v", result.Issues)
	}
}

func TestResolveReviewThinkingConfigAppliesLevelOverrideForInherit(t *testing.T) {
	base := ThinkingConfig{Enabled: true, Level: "low"}
	review := ReviewThinkingConfig{
		Mode:  ReviewThinkingModeInherit,
		Level: "HIGH",
	}

	got := ResolveReviewThinkingConfig(base, review)

	if !got.Enabled {
		t.Fatal("Enabled = false, want inherited true")
	}
	if got.Level != "high" {
		t.Fatalf("Level = %q, want high", got.Level)
	}
	if base.Level != "low" {
		t.Fatalf("base Level mutated to %q, want low", base.Level)
	}
}

func TestResolveReviewThinkingConfigEmptyLevelFallsBackToRuntimeLevel(t *testing.T) {
	base := ThinkingConfig{Enabled: false, Level: "xhigh"}
	review := ReviewThinkingConfig{
		Mode:  ReviewThinkingModeOn,
		Level: "",
	}

	got := ResolveReviewThinkingConfig(base, review)

	if !got.Enabled {
		t.Fatal("Enabled = false, want on override")
	}
	if got.Level != "xhigh" {
		t.Fatalf("Level = %q, want runtime xhigh", got.Level)
	}
}

func TestResolveReviewThinkingConfigOffIgnoresLevelOverride(t *testing.T) {
	base := ThinkingConfig{Enabled: true, Level: "xhigh"}
	review := ReviewThinkingConfig{
		Mode:  ReviewThinkingModeOff,
		Level: "high",
	}

	got := ResolveReviewThinkingConfig(base, review)

	if got.Enabled {
		t.Fatal("Enabled = true, want off override")
	}
	if got.Level != "xhigh" {
		t.Fatalf("Level = %q, want runtime xhigh", got.Level)
	}
}

func TestValidateConfig_ReviewWebSearchEvidenceBudgetRequiresPositiveValuesWhenEnabled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Review.WebSearchEvidence = ReviewWebSearchEvidenceConfig{
		Enabled:            true,
		MaxQueries:         0,
		MaxResultsPerQuery: -1,
	}

	result := ValidateConfig(cfg)
	if result.Valid {
		t.Fatal("ValidateConfig() valid = true, want false")
	}
	if !validationIssuesContain(result.Issues, "review.web_search_evidence.max_queries", "正の整数") {
		t.Fatalf("issues = %#v, want max_queries error", result.Issues)
	}
	if !validationIssuesContain(result.Issues, "review.web_search_evidence.max_results_per_query", "正の整数") {
		t.Fatalf("issues = %#v, want max_results_per_query error", result.Issues)
	}
}

func TestValidateConfig_ReviewWebSearchEvidenceBudgetIgnoredWhenDisabled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Review.WebSearchEvidence = ReviewWebSearchEvidenceConfig{
		Enabled:            false,
		MaxQueries:         0,
		MaxResultsPerQuery: 0,
	}

	result := ValidateConfig(cfg)
	if !result.Valid {
		t.Fatalf("ValidateConfig() valid = false, issues = %#v", result.Issues)
	}
}

func TestApplyEnvironmentOverrides_ReviewWebSearchEvidence(t *testing.T) {
	t.Setenv(reviewWebSearchEvidenceEnv, "1")

	cfg := DefaultConfig()
	cfg.ApplyEnvironmentOverrides()

	if !cfg.Review.WebSearchEvidence.Enabled {
		t.Fatal("Review.WebSearchEvidence.Enabled = false, want true from env override")
	}
}

func TestApplyAutoFixes_ReviewProviderTypo(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Review.Provider = "deepseak"

	result := ValidateConfig(cfg)
	if result.Valid {
		t.Fatal("ValidateConfig() valid = true, want false")
	}

	if fixed := ApplyAutoFixes(cfg, result); fixed != 1 {
		t.Fatalf("ApplyAutoFixes() = %d, want 1", fixed)
	}
	if cfg.Review.Provider != "deepseek" {
		t.Fatalf("Review.Provider = %q, want deepseek", cfg.Review.Provider)
	}
}

func TestApplyAutoFixes_ReviewThinkingMode(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Review.Thinking.Mode = "auto"

	result := ValidateConfig(cfg)
	if result.Valid {
		t.Fatal("ValidateConfig() valid = true, want false")
	}

	if fixed := ApplyAutoFixes(cfg, result); fixed != 1 {
		t.Fatalf("ApplyAutoFixes() = %d, want 1", fixed)
	}
	if cfg.Review.Thinking.Mode != ReviewThinkingModeInherit {
		t.Fatalf("Review.Thinking.Mode = %q, want inherit", cfg.Review.Thinking.Mode)
	}
}

func validationIssuesContain(issues []ValidationIssue, field, messageFragment string) bool {
	for _, issue := range issues {
		if issue.Field == field && strings.Contains(issue.Message, messageFragment) {
			return true
		}
	}
	return false
}
