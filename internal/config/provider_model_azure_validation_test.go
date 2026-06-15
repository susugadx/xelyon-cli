package config

import (
	"strings"
	"testing"
)

func TestValidateAzureDeploymentSelection_RejectsImplicitPlaceholder(t *testing.T) {
	cfg := DefaultConfig()

	err := ValidateAzureDeploymentSelection(cfg, "azure", "azure-gpt-5.4", false)

	if err == nil {
		t.Fatal("ValidateAzureDeploymentSelection() error = nil, want missing deployment error")
	}
	if !strings.Contains(err.Error(), "deployment is not configured") {
		t.Fatalf("ValidateAzureDeploymentSelection() error = %v, want deployment guidance", err)
	}
}

func TestValidateAzureDeploymentSelection_AllowsExplicitPlaceholder(t *testing.T) {
	cfg := DefaultConfig()

	if err := ValidateAzureDeploymentSelection(cfg, "azure", "azure-gpt-5.4", true); err != nil {
		t.Fatalf("ValidateAzureDeploymentSelection() error = %v, want nil for explicit deployment", err)
	}
}

func TestValidateAzureDeploymentSelection_AllowsConfiguredPlaceholder(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SetProviderModelConfig("azure", ProviderModelConfig{DefaultModel: "azure-gpt-5.4"})

	if err := ValidateAzureDeploymentSelection(cfg, "azure", "azure-gpt-5.4", false); err != nil {
		t.Fatalf("ValidateAzureDeploymentSelection() error = %v, want nil for configured deployment", err)
	}
}
