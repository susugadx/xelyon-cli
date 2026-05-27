package llmcatalog

import "testing"

func TestResolveProviderRoute_DescriptorDefaults(t *testing.T) {
	tests := []struct {
		provider         string
		wantRuntime      string
		wantPrompt       string
		wantEdit         string
		wantCapability   string
		wantModelCatalog string
		wantPricing      string
		wantDoctorPolicy string
	}{
		{
			provider:         "openai",
			wantRuntime:      "openai",
			wantPrompt:       "openai",
			wantEdit:         "apply_patch",
			wantCapability:   "openai",
			wantModelCatalog: "openai",
			wantPricing:      "openai",
			wantDoctorPolicy: "openai",
		},
		{
			provider:         "azure",
			wantRuntime:      "openai",
			wantPrompt:       "openai",
			wantEdit:         "apply_patch",
			wantCapability:   "azure",
			wantModelCatalog: "openai",
			wantPricing:      "openai",
			wantDoctorPolicy: "azure",
		},
		{
			provider:         "groq",
			wantRuntime:      "openai_compatible",
			wantPrompt:       "groq",
			wantEdit:         "legacy",
			wantCapability:   "groq",
			wantModelCatalog: "groq",
			wantPricing:      "groq",
			wantDoctorPolicy: "simple",
		},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			got := ResolveProviderRoute(tt.provider, "", "")
			if got.RuntimeFamily != tt.wantRuntime ||
				got.PromptFamily != tt.wantPrompt ||
				got.EditToolFamily != tt.wantEdit ||
				got.CapabilityFamily != tt.wantCapability ||
				got.ModelCatalogFamily != tt.wantModelCatalog ||
				got.PricingFamily != tt.wantPricing ||
				got.DoctorPolicyFamily != tt.wantDoctorPolicy {
				t.Fatalf("ResolveProviderRoute(%q) = %#v", tt.provider, got)
			}
		})
	}
}

func TestResolveProviderRoute_RoutedProviderFamilies(t *testing.T) {
	tests := []struct {
		name        string
		provider    string
		model       string
		catalog     string
		wantRuntime string
		wantPrompt  string
		wantEdit    string
	}{
		{
			name:        "bedrock claude catalog alias",
			provider:    "bedrock",
			model:       "corp-bedrock-sonnet",
			catalog:     "global.anthropic.claude-sonnet-4-6",
			wantRuntime: "bedrock_claude",
			wantPrompt:  "claude",
			wantEdit:    "legacy",
		},
		{
			name:        "bedrock converse default",
			provider:    "bedrock",
			model:       "amazon.nova-pro-v1:0",
			wantRuntime: "bedrock_converse",
			wantPrompt:  "",
			wantEdit:    "legacy",
		},
		{
			name:        "openrouter openai edit family",
			provider:    "openrouter",
			model:       "openai/gpt-5.4",
			wantRuntime: "openrouter",
			wantPrompt:  "",
			wantEdit:    "apply_patch",
		},
		{
			name:        "openrouter google edit family",
			provider:    "openrouter",
			model:       "google/gemini-3.1-pro-preview",
			wantRuntime: "openrouter",
			wantPrompt:  "",
			wantEdit:    "apply_patch",
		},
		{
			name:        "openrouter anthropic stays legacy",
			provider:    "openrouter",
			model:       "anthropic/claude-sonnet-4.6",
			wantRuntime: "openrouter",
			wantPrompt:  "",
			wantEdit:    "legacy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveProviderRoute(tt.provider, tt.model, tt.catalog)
			if got.RuntimeFamily != tt.wantRuntime || got.PromptFamily != tt.wantPrompt || got.EditToolFamily != tt.wantEdit {
				t.Fatalf("ResolveProviderRoute(%q, %q, %q) = %#v", tt.provider, tt.model, tt.catalog, got)
			}
		})
	}
}

func TestDefaultModelForProvider(t *testing.T) {
	if got := DefaultModelForProvider("openai"); got != "gpt-5.4" {
		t.Fatalf("DefaultModelForProvider(openai) = %q, want gpt-5.4", got)
	}
	if got := DefaultModelForProvider("moonshot"); got != "kimi-k2.6" {
		t.Fatalf("DefaultModelForProvider(moonshot) = %q, want kimi-k2.6", got)
	}
	if got := DefaultModelForProvider("unknown"); got != "" {
		t.Fatalf("DefaultModelForProvider(unknown) = %q, want empty", got)
	}
}
