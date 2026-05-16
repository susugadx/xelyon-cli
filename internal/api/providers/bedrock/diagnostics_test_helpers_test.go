package bedrock

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func bedrockDiagnosticTestConfig(model, catalogModel string) *config.Config {
	cfg := config.DefaultConfig()
	cfg.ProviderModels["bedrock"] = config.ProviderModelConfig{
		DefaultModel:    model,
		CatalogModel:    catalogModel,
		MaxOutputTokens: 64,
	}
	return cfg
}

func bedrockDiagnosticPolicyMaxConfig(model, catalogModel string, providerMax int, override config.ModelOverride) *config.Config {
	cfg := config.DefaultConfig()
	pm := config.ProviderModelConfig{
		DefaultModel:    model,
		CatalogModel:    catalogModel,
		MaxOutputTokens: providerMax,
	}
	if override.CatalogModel != "" || override.MaxOutputTokens > 0 {
		pm.ModelOverrides = map[string]config.ModelOverride{
			model: override,
		}
	}
	cfg.ProviderModels["bedrock"] = pm
	return cfg
}

func bedrockDiagnosticRequest(requests []DiagnosticSmokeRequestResult, name string) DiagnosticSmokeRequestResult {
	for _, request := range requests {
		if request.Name == name {
			return request
		}
	}
	return DiagnosticSmokeRequestResult{}
}

func bedrockDiagnosticCheck(report DiagnosticReport, name string) (DiagnosticCheck, bool) {
	for _, check := range report.Checks {
		if check.Name == name {
			return check, true
		}
	}
	return DiagnosticCheck{}, false
}

func hasBedrockDiagnosticCheck(report DiagnosticReport, name string, status DiagnosticStatus) bool {
	for _, check := range report.Checks {
		if check.Name == name && check.Status == status {
			return true
		}
	}
	return false
}

func hasBedrockDiagnosticCheckName(report DiagnosticReport, name string) bool {
	for _, check := range report.Checks {
		if check.Name == name {
			return true
		}
	}
	return false
}

func setBedrockDiagnosticTestEnv(t *testing.T) {
	t.Helper()
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("AWS_DEFAULT_REGION", "")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("BEDROCK_FUNCTION_CALLING", "")
	t.Setenv("XELYON_MODEL", "")
}
