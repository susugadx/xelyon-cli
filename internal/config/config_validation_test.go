package config

import "testing"

func TestValidateModel(t *testing.T) {
	tests := []string{"any-model", "gpt-4", "deepseek-coder", ""}
	for _, model := range tests {
		if !ValidateModel(model) {
			t.Errorf("ValidateModel(%q) should always return true", model)
		}
	}
}

func TestValidateProviderHistoryReductionMode(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ProviderHistoryReduction.Mode = ProviderHistoryReductionMode("auto")

	result := ValidateConfig(cfg)
	if len(result.Issues) != 1 {
		t.Fatalf("ValidateConfig() issues = %#v, want one provider history reduction issue", result.Issues)
	}
	issue := result.Issues[0]
	if issue.Field != "provider_history_reduction.mode" || issue.Severity != ValidationSeverityError || !issue.CanAutoFix || issue.FixedValue != ProviderHistoryReductionModeDryRun {
		t.Fatalf("provider history reduction issue = %#v, want error autofix to dry_run", issue)
	}

	if fixed := ApplyAutoFixes(cfg, result); fixed != 1 {
		t.Fatalf("ApplyAutoFixes() = %d, want 1", fixed)
	}
	if cfg.ProviderHistoryReduction.Mode != ProviderHistoryReductionModeDryRun {
		t.Fatalf("ProviderHistoryReduction.Mode = %q, want dry_run", cfg.ProviderHistoryReduction.Mode)
	}
}

func TestValidateProviderHistoryRawOutputArtifacts(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ProviderHistoryReduction.RawOutputArtifacts.Mode = ProviderHistoryRawOutputArtifactsMode("auto")
	cfg.ProviderHistoryReduction.RawOutputArtifacts.SessionQuotaBytes = 1
	cfg.ProviderHistoryReduction.RawOutputArtifacts.ActiveContextBudgetMaxTokens = 1
	cfg.ProviderHistoryReduction.RawOutputArtifacts.Retention = ProviderHistoryRawOutputArtifactsRetention("forever")

	result := ValidateConfig(cfg)
	if len(result.Issues) != 4 {
		t.Fatalf("ValidateConfig() issues = %#v, want four raw_output_artifacts issues", result.Issues)
	}
	for _, want := range []string{
		"provider_history_reduction.raw_output_artifacts.mode",
		"provider_history_reduction.raw_output_artifacts.session_quota_bytes",
		"provider_history_reduction.raw_output_artifacts.active_context_budget_max_tokens",
		"provider_history_reduction.raw_output_artifacts.retention",
	} {
		found := false
		for _, issue := range result.Issues {
			if issue.Field == want && issue.Severity == ValidationSeverityError && issue.CanAutoFix {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("issues = %#v, want autofixable error for %s", result.Issues, want)
		}
	}

	if fixed := ApplyAutoFixes(cfg, result); fixed != 4 {
		t.Fatalf("ApplyAutoFixes() = %d, want 4", fixed)
	}
	if cfg.ProviderHistoryReduction.RawOutputArtifacts.Mode != ProviderHistoryRawOutputArtifactsModeDryRun ||
		cfg.ProviderHistoryReduction.RawOutputArtifacts.SessionQuotaBytes != defaultRawOutputArtifactSessionQuotaBytes ||
		cfg.ProviderHistoryReduction.RawOutputArtifacts.ActiveContextBudgetMaxTokens != defaultRawOutputArtifactActiveContextBudgetMax ||
		cfg.ProviderHistoryReduction.RawOutputArtifacts.Retention != ProviderHistoryRawOutputArtifactsRetentionSession {
		t.Fatalf("RawOutputArtifacts after autofix = %#v", cfg.ProviderHistoryReduction.RawOutputArtifacts)
	}
}

func TestValidateProviderHistoryRawOutputArtifactsRoot(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ProviderHistoryReduction.RawOutputArtifacts.Root = "relative/rawoutputs"

	result := ValidateConfig(cfg)
	if len(result.Issues) != 1 {
		t.Fatalf("ValidateConfig() issues = %#v, want one root issue", result.Issues)
	}
	issue := result.Issues[0]
	if issue.Field != "provider_history_reduction.raw_output_artifacts.root" ||
		issue.Severity != ValidationSeverityError ||
		issue.CanAutoFix {
		t.Fatalf("root issue = %#v, want non-autofixable error", issue)
	}
}

func TestValidateSkillsRouterConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Skills.Router.Activation = SkillsRouterActivation("auto")
	cfg.Skills.Router.UsageRetentionDays = 0

	result := ValidateConfig(cfg)
	if len(result.Issues) != 2 {
		t.Fatalf("ValidateConfig() issues = %#v, want two skills issues", result.Issues)
	}
	for _, want := range []string{"skills.router.activation", "skills.router.usage_retention_days"} {
		found := false
		for _, issue := range result.Issues {
			if issue.Field == want && issue.Severity == ValidationSeverityError && issue.CanAutoFix {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("issues = %#v, want autofixable error for %s", result.Issues, want)
		}
	}

	if fixed := ApplyAutoFixes(cfg, result); fixed != 2 {
		t.Fatalf("ApplyAutoFixes() = %d, want 2", fixed)
	}
	if cfg.Skills.Router.Activation != SkillsRouterActivationHint {
		t.Fatalf("Skills.Router.Activation = %q, want hint", cfg.Skills.Router.Activation)
	}
	if cfg.Skills.Router.UsageRetentionDays != defaultSkillsRouterUsageRetentionDays {
		t.Fatalf("Skills.Router.UsageRetentionDays = %d, want %d", cfg.Skills.Router.UsageRetentionDays, defaultSkillsRouterUsageRetentionDays)
	}
}

func TestValidateMCPSurfaceBudgetRejectsNegativeValues(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MCP.SurfaceBudget.MaxTools = -1
	cfg.MCP.SurfaceBudget.EstimatedTokens = -2
	cfg.MCP.SurfaceBudget.MaxSchemaBytesPerTool = -3

	result := ValidateConfig(cfg)
	if len(result.Issues) != 3 {
		t.Fatalf("ValidateConfig() issues = %#v, want three MCP surface budget issues", result.Issues)
	}
	for _, want := range []string{
		"mcp.surface_budget.max_tools",
		"mcp.surface_budget.estimated_tokens",
		"mcp.surface_budget.max_schema_bytes_per_tool",
	} {
		found := false
		for _, issue := range result.Issues {
			if issue.Field == want && issue.Severity == ValidationSeverityError && issue.CanAutoFix {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("issues = %#v, want autofixable error for %s", result.Issues, want)
		}
	}

	if fixed := ApplyAutoFixes(cfg, result); fixed != 3 {
		t.Fatalf("ApplyAutoFixes() = %d, want 3", fixed)
	}
	defaults := defaultMCPSurfaceBudgetConfig()
	if cfg.MCP.SurfaceBudget != defaults {
		t.Fatalf("MCP.SurfaceBudget after autofix = %#v, want %#v", cfg.MCP.SurfaceBudget, defaults)
	}
}
