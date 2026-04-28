package azure

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

const (
	azureSmokeEnabledEnv        = "XELYON_AZURE_SMOKE"
	azureSmokeDeploymentEnv     = "AZURE_OPENAI_DEPLOYMENT"
	azureSmokeCatalogModelEnv   = "AZURE_OPENAI_CATALOG_MODEL"
	azureSmokeProDeploymentEnv  = "AZURE_OPENAI_PRO_DEPLOYMENT"
	azureSmokeProCatalogModel   = "AZURE_OPENAI_PRO_CATALOG_MODEL"
	azureSmokeRequestTimeout    = 120 * time.Second
	azureSmokeMaxOutputTokens   = 64
	azureSmokeDefaultCatalog    = "gpt-5.4"
	azureSmokeDefaultProCatalog = "gpt-5.5-pro"
)

func TestAzureResponsesSmoke(t *testing.T) {
	requireAzureSmokeEnabled(t)
	requireAzureSmokeCredentials(t)

	t.Setenv("AZURE_OPENAI_FUNCTION_CALLING", "0")

	deployment := strings.TrimSpace(os.Getenv(azureSmokeDeploymentEnv))
	if deployment == "" {
		t.Fatalf("%s is required when %s=1", azureSmokeDeploymentEnv, azureSmokeEnabledEnv)
	}

	t.Run("response id chaining", func(t *testing.T) {
		cfg := azureSmokeConfig(deployment, envOrDefault(azureSmokeCatalogModelEnv, azureSmokeDefaultCatalog))
		p := New(os.Getenv(apiKeyEnv))
		ctx, cancel := context.WithTimeout(azureTestContext(cfg), azureSmokeRequestTimeout)
		defer cancel()

		first, err := p.ChatWithTools(ctx, "Reply briefly.", []api.Message{{Role: "user", Content: "Reply with: xelyon azure smoke one"}}, "")
		if err != nil {
			t.Fatalf("first ChatWithTools() error = %v", err)
		}
		if strings.TrimSpace(first) == "" {
			t.Fatal("first ChatWithTools() returned empty content")
		}
		firstResponseID := p.GetResponseID()
		if firstResponseID == "" {
			t.Fatal("first response ID is empty")
		}

		second, err := p.ChatWithTools(ctx, "Reply briefly.", []api.Message{{Role: "user", Content: "Reply with: xelyon azure smoke two"}}, "")
		if err != nil {
			t.Fatalf("second ChatWithTools() error = %v", err)
		}
		if strings.TrimSpace(second) == "" {
			t.Fatal("second ChatWithTools() returned empty content")
		}
		if p.GetResponseID() == "" {
			t.Fatal("second response ID is empty")
		}
	})

	if proDeployment := strings.TrimSpace(os.Getenv(azureSmokeProDeploymentEnv)); proDeployment != "" {
		t.Run("pro non streaming", func(t *testing.T) {
			cfg := azureSmokeConfig(proDeployment, envOrDefault(azureSmokeProCatalogModel, azureSmokeDefaultProCatalog))
			p := New(os.Getenv(apiKeyEnv))
			ctx, cancel := context.WithTimeout(azureTestContext(cfg), azureSmokeRequestTimeout)
			defer cancel()

			content, err := p.ChatWithTools(ctx, "Reply briefly.", []api.Message{{Role: "user", Content: "Reply with: xelyon azure pro smoke"}}, "")
			if err != nil {
				t.Fatalf("ChatWithTools() error = %v", err)
			}
			if strings.TrimSpace(content) == "" {
				t.Fatal("ChatWithTools() returned empty content")
			}
			if p.GetResponseID() == "" {
				t.Fatal("response ID is empty")
			}
		})
	}
}

func requireAzureSmokeEnabled(t *testing.T) {
	t.Helper()
	if os.Getenv(azureSmokeEnabledEnv) != "1" {
		t.Skipf("set %s=1 to run the live Azure OpenAI smoke test", azureSmokeEnabledEnv)
	}
}

func requireAzureSmokeCredentials(t *testing.T) {
	t.Helper()
	if strings.TrimSpace(os.Getenv(baseURLEnv)) == "" {
		t.Fatalf("%s is required when %s=1", baseURLEnv, azureSmokeEnabledEnv)
	}
	if strings.TrimSpace(os.Getenv(apiKeyEnv)) == "" && strings.TrimSpace(os.Getenv(authTokenEnv)) == "" {
		t.Fatalf("%s or %s is required when %s=1", apiKeyEnv, authTokenEnv, azureSmokeEnabledEnv)
	}
}

func azureSmokeConfig(deployment, catalogModel string) *config.Config {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("azure", config.ProviderModelConfig{
		DefaultModel: deployment,
		CatalogModel: catalogModel,
		ModelOverrides: map[string]config.ModelOverride{
			deployment: {
				CatalogModel:    catalogModel,
				MaxOutputTokens: azureSmokeMaxOutputTokens,
			},
		},
	})
	return cfg
}

func envOrDefault(envName, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(envName)); value != "" {
		return value
	}
	return fallback
}
