package config

import "testing"

func TestGeminiServiceTierDefaultsAndNormalization(t *testing.T) {
	if got := (*Config)(nil).GeminiServiceTier(); got != GeminiServiceTierStandard {
		t.Fatalf("nil config GeminiServiceTier() = %q, want standard", got)
	}

	cfg := DefaultConfig()
	if got := cfg.GeminiServiceTier(); got != GeminiServiceTierStandard {
		t.Fatalf("default GeminiServiceTier() = %q, want standard", got)
	}

	cfg.Gemini.ServiceTier = " Priority "
	if got := cfg.GeminiServiceTier(); got != GeminiServiceTierPriority {
		t.Fatalf("normalized GeminiServiceTier() = %q, want priority", got)
	}

	cfg.Gemini.ServiceTier = "turbo"
	if got := cfg.GeminiServiceTier(); got != GeminiServiceTierStandard {
		t.Fatalf("invalid GeminiServiceTier() = %q, want standard", got)
	}
}

func TestValidateGeminiServiceTier(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Gemini.ServiceTier = "turbo"

	result := ValidateConfig(cfg)
	if len(result.Issues) != 1 {
		t.Fatalf("ValidateConfig() issues = %#v, want one Gemini issue", result.Issues)
	}
	issue := result.Issues[0]
	if issue.Field != "gemini.service_tier" || !issue.CanAutoFix || issue.FixedValue != GeminiServiceTierStandard {
		t.Fatalf("Gemini issue = %#v, want autofix to standard", issue)
	}

	if fixed := ApplyAutoFixes(cfg, result); fixed != 1 {
		t.Fatalf("ApplyAutoFixes() = %d, want 1", fixed)
	}
	if cfg.Gemini.ServiceTier != GeminiServiceTierStandard {
		t.Fatalf("Gemini.ServiceTier = %q, want standard", cfg.Gemini.ServiceTier)
	}
}
