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
}

func TestReviewConfigYAMLRoundTrip(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Review = ReviewConfig{
		Provider: "claude",
		Model:    "claude-sonnet-4-6",
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

func validationIssuesContain(issues []ValidationIssue, field, messageFragment string) bool {
	for _, issue := range issues {
		if issue.Field == field && strings.Contains(issue.Message, messageFragment) {
			return true
		}
	}
	return false
}
