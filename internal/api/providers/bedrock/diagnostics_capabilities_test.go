package bedrock

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

func TestDiagnoseBedrock_LocalCapabilitiesDoNotLoadAWSConfigOrCredentials(t *testing.T) {
	setBedrockDiagnosticTestEnv(t)
	missingAWSPath := filepath.Join(t.TempDir(), "missing")
	t.Setenv("AWS_PROFILE", "xelyon-missing-profile")
	t.Setenv("AWS_CONFIG_FILE", missingAWSPath)
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", missingAWSPath)
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")
	t.Setenv("AWS_SESSION_TOKEN", "")
	thinkingCfg := bedrockDiagnosticTestConfig(defaultModel, defaultModel)
	thinkingCfg.Thinking.Enabled = true

	tests := []struct {
		name    string
		options DiagnosticOptions
	}{
		{
			name: "capabilities",
			options: DiagnosticOptions{
				Config:       bedrockDiagnosticTestConfig(defaultModel, defaultModel),
				Model:        defaultModel,
				CatalogModel: defaultModel,
				Capabilities: true,
			},
		},
		{
			name: "required capabilities",
			options: DiagnosticOptions{
				Config:               thinkingCfg,
				Model:                defaultModel,
				CatalogModel:         defaultModel,
				RequiredCapabilities: []string{"image_input", "thinking"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := Diagnose(context.Background(), tt.options)

			if report.HasFailures() {
				t.Fatalf("Diagnose() HasFailures = true; checks=%#v", report.Checks)
			}
			for _, checkName := range []string{"aws_config", "region", "auth"} {
				if hasBedrockDiagnosticCheckName(report, checkName) {
					t.Fatalf("%s check should not be added for capability-only diagnostics: %#v", checkName, report.Checks)
				}
			}
			if tt.options.Capabilities && report.Capabilities == nil {
				t.Fatal("Capabilities = nil, want resolved capability DTO")
			}
			if providerdiag.HasRequiredCapabilityRequest(tt.options.RequiredCapabilities) &&
				!hasBedrockDiagnosticCheck(report, providerdiag.RequiredCapabilityCheckName, DiagnosticStatusOK) {
				t.Fatalf("required_capability check missing or not ok: %#v", report.Checks)
			}
		})
	}
}

func TestDiagnoseBedrock_RequiredThinkingFollowsRequestConfig(t *testing.T) {
	setBedrockDiagnosticTestEnv(t)

	disabled := Diagnose(context.Background(), DiagnosticOptions{
		Config:               bedrockDiagnosticTestConfig(defaultModel, defaultModel),
		Model:                defaultModel,
		CatalogModel:         defaultModel,
		RequiredCapabilities: []string{providerdiag.RequiredCapabilityThinking},
	})
	if !disabled.HasFailures() {
		t.Fatalf("HasFailures() = false, want thinking required capability failure when request config disables thinking: %#v", disabled.Checks)
	}
	check, ok := bedrockDiagnosticCheck(disabled, providerdiag.RequiredCapabilityCheckName)
	if !ok || check.Status != DiagnosticStatusFail || !strings.Contains(check.Detail, "thinking=missing") {
		t.Fatalf("required_capability = %#v ok=%t, want missing thinking failure", check, ok)
	}

	cfg := bedrockDiagnosticTestConfig(defaultModel, defaultModel)
	cfg.Thinking.Enabled = true
	enabled := Diagnose(context.Background(), DiagnosticOptions{
		Config:               cfg,
		Model:                defaultModel,
		CatalogModel:         defaultModel,
		RequiredCapabilities: []string{providerdiag.RequiredCapabilityThinking},
	})
	if enabled.HasFailures() {
		t.Fatalf("HasFailures() = true, want thinking required capability to pass when request config enables thinking: %#v", enabled.Checks)
	}
	check, ok = bedrockDiagnosticCheck(enabled, providerdiag.RequiredCapabilityCheckName)
	if !ok || check.Status != DiagnosticStatusOK || !strings.Contains(check.Detail, "thinking=ok") {
		t.Fatalf("required_capability = %#v ok=%t, want ok thinking", check, ok)
	}
}

func TestDiagnoseBedrock_CapabilitiesPreserveMaxOutputSource(t *testing.T) {
	setBedrockDiagnosticTestEnv(t)

	tests := []struct {
		name       string
		cfg        *config.Config
		model      string
		catalog    string
		wantTokens int
		wantSource string
		wantRoute  string
	}{
		{
			name:       "converse catalog",
			cfg:        bedrockDiagnosticPolicyMaxConfig("amazon.nova-pro-v1:0", "amazon.nova-pro-v1:0", 9999, config.ModelOverride{}),
			model:      "amazon.nova-pro-v1:0",
			catalog:    "amazon.nova-pro-v1:0",
			wantTokens: 5000,
			wantSource: providerdiag.MaxOutputSourceCatalog,
			wantRoute:  string(bedrockRouteConverseStream),
		},
		{
			name:       "converse model override",
			cfg:        bedrockDiagnosticPolicyMaxConfig("amazon.nova-pro-v1:0", "amazon.nova-pro-v1:0", 9999, config.ModelOverride{CatalogModel: "amazon.nova-pro-v1:0", MaxOutputTokens: 2048}),
			model:      "amazon.nova-pro-v1:0",
			catalog:    "amazon.nova-pro-v1:0",
			wantTokens: 2048,
			wantSource: providerdiag.MaxOutputSourceModelOverrides,
			wantRoute:  string(bedrockRouteConverseStream),
		},
		{
			name:       "claude provider default",
			cfg:        bedrockDiagnosticPolicyMaxConfig("anthropic.claude-custom-v1:0", "anthropic.claude-custom-v1:0", 9999, config.ModelOverride{}),
			model:      "anthropic.claude-custom-v1:0",
			catalog:    "anthropic.claude-custom-v1:0",
			wantTokens: 9999,
			wantSource: providerdiag.MaxOutputSourceProviderDefault,
			wantRoute:  string(bedrockRouteClaudeMessages),
		},
		{
			name:       "claude model override",
			cfg:        bedrockDiagnosticPolicyMaxConfig(defaultModel, defaultModel, 9999, config.ModelOverride{CatalogModel: defaultModel, MaxOutputTokens: 2048}),
			model:      defaultModel,
			catalog:    defaultModel,
			wantTokens: 2048,
			wantSource: providerdiag.MaxOutputSourceModelOverrides,
			wantRoute:  string(bedrockRouteClaudeMessages),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := Diagnose(context.Background(), DiagnosticOptions{
				Config:       tt.cfg,
				Model:        tt.model,
				CatalogModel: tt.catalog,
				Capabilities: true,
			})

			if report.Route != tt.wantRoute {
				t.Fatalf("Route = %q, want %q", report.Route, tt.wantRoute)
			}
			if report.Capabilities == nil {
				t.Fatal("Capabilities = nil, want resolved capability DTO")
			}
			if !report.Capabilities.MaxOutputTokensKnown ||
				report.Capabilities.MaxOutputTokens != tt.wantTokens ||
				report.Capabilities.MaxOutputTokensSource != tt.wantSource {
				t.Fatalf(
					"capability max output = %d known=%t source=%q, want %d known=true source=%q",
					report.Capabilities.MaxOutputTokens,
					report.Capabilities.MaxOutputTokensKnown,
					report.Capabilities.MaxOutputTokensSource,
					tt.wantTokens,
					tt.wantSource,
				)
			}
		})
	}
}

func TestDiagnoseBedrock_CapabilitiesRejectCrossProviderCatalogMetadata(t *testing.T) {
	setBedrockDiagnosticTestEnv(t)

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "amazon.nova-pro-v1:0",
		CatalogModel: "gpt-5.4",
		Capabilities: true,
	})

	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want invalid catalog metadata to warn without failing: %#v", report.Checks)
	}
	if report.Capabilities == nil {
		t.Fatal("Capabilities = nil, want resolved capability DTO")
	}
	if report.Capabilities.ContextWindowKnown || report.Capabilities.ContextWindowTokens != 0 {
		t.Fatalf("context window = %d known=%t, want unknown for non-Bedrock catalog", report.Capabilities.ContextWindowTokens, report.Capabilities.ContextWindowKnown)
	}
	if !report.Capabilities.MaxOutputTokensKnown || report.Capabilities.MaxOutputTokens != 5000 || report.Capabilities.MaxOutputTokensSource != providerdiag.MaxOutputSourceCatalog {
		t.Fatalf("max output = %d known=%t source=%q, want Bedrock request-model catalog fallback", report.Capabilities.MaxOutputTokens, report.Capabilities.MaxOutputTokensKnown, report.Capabilities.MaxOutputTokensSource)
	}
	if report.Capabilities.Pricing.Available {
		t.Fatalf("pricing = %+v, want unavailable for non-Bedrock catalog", report.Capabilities.Pricing)
	}

	capabilities, ok := bedrockDiagnosticCheck(report, "capabilities")
	if !ok || capabilities.Status != DiagnosticStatusOK {
		t.Fatalf("capabilities check = %#v, %v; want ok", capabilities, ok)
	}
	for _, unwanted := range []string{"context_window=1000000", "max_output_tokens=64000", "pricing=input $2.50/M"} {
		if strings.Contains(capabilities.Detail, unwanted) {
			t.Fatalf("capabilities detail = %q, should not contain cross-provider metadata %q", capabilities.Detail, unwanted)
		}
	}
	catalogPolicy, ok := bedrockDiagnosticCheck(report, "catalog_policy")
	if !ok || catalogPolicy.Status != DiagnosticStatusWarn {
		t.Fatalf("catalog_policy check = %#v, %v; want warn", catalogPolicy, ok)
	}
	if !strings.Contains(catalogPolicy.Detail, "catalog_model=gpt-5.4, context_window=unknown") {
		t.Fatalf("catalog_policy detail = %q, want unknown context for non-Bedrock catalog", catalogPolicy.Detail)
	}
}

func TestDiagnoseBedrock_ConverseCrossProviderCatalogIgnoresProviderDefaultMaxOutput(t *testing.T) {
	setBedrockDiagnosticTestEnv(t)

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       bedrockDiagnosticPolicyMaxConfig("amazon.nova-pro-v1:0", "gpt-5.4", 9999, config.ModelOverride{}),
		Model:        "amazon.nova-pro-v1:0",
		CatalogModel: "gpt-5.4",
		Capabilities: true,
	})

	if report.Route != string(bedrockRouteConverseStream) {
		t.Fatalf("Route = %q, want %q", report.Route, bedrockRouteConverseStream)
	}
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want invalid catalog metadata to warn without failing: %#v", report.Checks)
	}
	if report.Capabilities == nil {
		t.Fatal("Capabilities = nil, want resolved capability DTO")
	}
	if !report.Capabilities.MaxOutputTokensKnown ||
		report.Capabilities.MaxOutputTokens != 5000 ||
		report.Capabilities.MaxOutputTokensSource != providerdiag.MaxOutputSourceCatalog {
		t.Fatalf(
			"max output = %d known=%t source=%q, want request-model catalog max output",
			report.Capabilities.MaxOutputTokens,
			report.Capabilities.MaxOutputTokensKnown,
			report.Capabilities.MaxOutputTokensSource,
		)
	}
	catalogPolicy, ok := bedrockDiagnosticCheck(report, "catalog_policy")
	if !ok || catalogPolicy.Status != DiagnosticStatusWarn {
		t.Fatalf("catalog_policy check = %#v, %v; want warn", catalogPolicy, ok)
	}
	if !strings.Contains(catalogPolicy.Detail, "max_output_tokens=5000") {
		t.Fatalf("catalog_policy detail = %q, want request-model catalog max output", catalogPolicy.Detail)
	}
	if strings.Contains(catalogPolicy.Detail, "max_output_tokens=9999") {
		t.Fatalf("catalog_policy detail = %q, should not use provider default max output for Converse", catalogPolicy.Detail)
	}
}

func TestDiagnoseBedrock_ConverseUnknownModelCrossProviderCatalogOmitsProviderDefaultMaxOutput(t *testing.T) {
	setBedrockDiagnosticTestEnv(t)

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       bedrockDiagnosticPolicyMaxConfig("amazon.unknown-future-v1:0", "gpt-5.4", 9999, config.ModelOverride{}),
		Model:        "amazon.unknown-future-v1:0",
		CatalogModel: "gpt-5.4",
		Capabilities: true,
	})

	if report.Route != string(bedrockRouteConverseStream) {
		t.Fatalf("Route = %q, want %q", report.Route, bedrockRouteConverseStream)
	}
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want missing metadata to warn without failing: %#v", report.Checks)
	}
	if report.Capabilities == nil {
		t.Fatal("Capabilities = nil, want resolved capability DTO")
	}
	if report.Capabilities.MaxOutputTokensKnown || report.Capabilities.MaxOutputTokens != 0 {
		t.Fatalf(
			"max output = %d known=%t source=%q, want unknown for Converse unknown request model",
			report.Capabilities.MaxOutputTokens,
			report.Capabilities.MaxOutputTokensKnown,
			report.Capabilities.MaxOutputTokensSource,
		)
	}
	catalogPolicy, ok := bedrockDiagnosticCheck(report, "catalog_policy")
	if !ok || catalogPolicy.Status != DiagnosticStatusWarn {
		t.Fatalf("catalog_policy check = %#v, %v; want warn", catalogPolicy, ok)
	}
	if !strings.Contains(catalogPolicy.Detail, "max_output_tokens=unknown") {
		t.Fatalf("catalog_policy detail = %q, want unknown max output", catalogPolicy.Detail)
	}
	if strings.Contains(catalogPolicy.Detail, "max_output_tokens=9999") {
		t.Fatalf("catalog_policy detail = %q, should not use provider default max output for Converse", catalogPolicy.Detail)
	}
}

func TestDiagnoseBedrock_ClaudeMessagesCrossProviderCatalogUsesBedrockMaxOutput(t *testing.T) {
	setBedrockDiagnosticTestEnv(t)

	model := "global.anthropic.claude-sonnet-4-6"
	cfg := bedrockDiagnosticPolicyMaxConfig(model, "gpt-5.5", 9999, config.ModelOverride{})
	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       cfg,
		Model:        model,
		CatalogModel: "gpt-5.5",
		Capabilities: true,
	})

	if report.Route != string(bedrockRouteClaudeMessages) {
		t.Fatalf("Route = %q, want %q", report.Route, bedrockRouteClaudeMessages)
	}
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want cross-provider catalog metadata warning without failure: %#v", report.Checks)
	}
	if report.Capabilities == nil {
		t.Fatal("Capabilities = nil, want resolved capability DTO")
	}
	if !report.Capabilities.MaxOutputTokensKnown ||
		report.Capabilities.MaxOutputTokens != 64000 ||
		report.Capabilities.MaxOutputTokensSource != providerdiag.MaxOutputSourceCatalog {
		t.Fatalf(
			"max output = %d known=%t source=%q, want Bedrock request-model catalog max output 64000",
			report.Capabilities.MaxOutputTokens,
			report.Capabilities.MaxOutputTokensKnown,
			report.Capabilities.MaxOutputTokensSource,
		)
	}
	catalogPolicy, ok := bedrockDiagnosticCheck(report, "catalog_policy")
	if !ok || catalogPolicy.Status != DiagnosticStatusWarn {
		t.Fatalf("catalog_policy check = %#v, %v; want warn", catalogPolicy, ok)
	}
	if !strings.Contains(catalogPolicy.Detail, "max_output_tokens=64000") {
		t.Fatalf("catalog_policy detail = %q, want Bedrock request-model catalog max output", catalogPolicy.Detail)
	}
	for _, unwanted := range []string{"max_output_tokens=128000", "max_output_tokens=9999"} {
		if strings.Contains(catalogPolicy.Detail, unwanted) {
			t.Fatalf("catalog_policy detail = %q, should not contain %q", catalogPolicy.Detail, unwanted)
		}
	}
}

func TestDiagnoseBedrock_CapabilitiesPreserveMaxOutputOverrideForCrossProviderCatalog(t *testing.T) {
	setBedrockDiagnosticTestEnv(t)

	model := "amazon.nova-pro-v1:0"
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("bedrock", config.ProviderModelConfig{
		ModelOverrides: map[string]config.ModelOverride{
			model: {
				CatalogModel:    "gpt-5.4",
				MaxOutputTokens: 2048,
			},
		},
	})
	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       cfg,
		Model:        model,
		CatalogModel: "gpt-5.4",
		Capabilities: true,
	})

	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want override-backed capabilities without failure: %#v", report.Checks)
	}
	if report.Capabilities == nil {
		t.Fatal("Capabilities = nil, want resolved capability DTO")
	}
	if report.Capabilities.ContextWindowKnown || report.Capabilities.ContextWindowTokens != 0 {
		t.Fatalf("context window = %d known=%t, want unknown for non-Bedrock catalog", report.Capabilities.ContextWindowTokens, report.Capabilities.ContextWindowKnown)
	}
	if !report.Capabilities.MaxOutputTokensKnown ||
		report.Capabilities.MaxOutputTokens != 2048 ||
		report.Capabilities.MaxOutputTokensSource != providerdiag.MaxOutputSourceModelOverrides {
		t.Fatalf(
			"max output = %d known=%t source=%q, want 2048 known=true source=%q",
			report.Capabilities.MaxOutputTokens,
			report.Capabilities.MaxOutputTokensKnown,
			report.Capabilities.MaxOutputTokensSource,
			providerdiag.MaxOutputSourceModelOverrides,
		)
	}
	if report.Capabilities.Pricing.Available {
		t.Fatalf("pricing = %+v, want unavailable for non-Bedrock catalog", report.Capabilities.Pricing)
	}
}

func TestDiagnoseBedrock_RequiredFunctionCallingFollowsRouteToolUseGate(t *testing.T) {
	setBedrockDiagnosticTestEnv(t)

	tests := []struct {
		name        string
		model       string
		catalog     string
		wantStatus  DiagnosticStatus
		wantFailure bool
		wantDetail  string
	}{
		{
			name:       "claude messages route",
			model:      defaultModel,
			catalog:    defaultModel,
			wantStatus: DiagnosticStatusOK,
			wantDetail: "function_calling=ok",
		},
		{
			name:       "converse route with verified streaming tool use",
			model:      "amazon.nova-pro-v1:0",
			catalog:    "amazon.nova-pro-v1:0",
			wantStatus: DiagnosticStatusOK,
			wantDetail: "function_calling=ok",
		},
		{
			name:        "converse route without verified streaming tool use",
			model:       "us.deepseek.r1-v1:0",
			catalog:     "us.deepseek.r1-v1:0",
			wantStatus:  DiagnosticStatusFail,
			wantFailure: true,
			wantDetail:  "function_calling=missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := Diagnose(context.Background(), DiagnosticOptions{
				Config:               bedrockDiagnosticTestConfig(tt.model, tt.catalog),
				Model:                tt.model,
				CatalogModel:         tt.catalog,
				RequiredCapabilities: []string{providerdiag.RequiredCapabilityFunctionCalling},
			})

			if got := report.HasFailures(); got != tt.wantFailure {
				t.Fatalf("HasFailures() = %t, want %t; checks=%#v", got, tt.wantFailure, report.Checks)
			}
			check, ok := bedrockDiagnosticCheck(report, providerdiag.RequiredCapabilityCheckName)
			if !ok {
				t.Fatalf("missing required_capability check: %#v", report.Checks)
			}
			if check.Status != tt.wantStatus {
				t.Fatalf("required_capability status = %s, want %s; check=%#v", check.Status, tt.wantStatus, check)
			}
			if !strings.Contains(check.Detail, tt.wantDetail) {
				t.Fatalf("required_capability detail = %q, want %q", check.Detail, tt.wantDetail)
			}
		})
	}
}

func TestDiagnoseBedrock_CapabilitySnapshotMarksUnsupportedConverseToolUseMissing(t *testing.T) {
	setBedrockDiagnosticTestEnv(t)

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       bedrockDiagnosticTestConfig("us.deepseek.r1-v1:0", "us.deepseek.r1-v1:0"),
		Model:        "us.deepseek.r1-v1:0",
		CatalogModel: "us.deepseek.r1-v1:0",
		Capabilities: true,
	})

	if report.Capabilities == nil {
		t.Fatal("Capabilities = nil, want resolved capability DTO")
	}
	if report.Capabilities.FunctionCalling {
		t.Fatalf("Capabilities.FunctionCalling = true, want false for unsupported Converse tool use: %+v", report.Capabilities)
	}
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want capability snapshot without required gate to stay non-failing; checks=%#v", report.Checks)
	}
}
