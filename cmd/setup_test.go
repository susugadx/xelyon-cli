package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestSetupCommand_PreservesInvalidProviderFromEnvironment(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XELYON_PROVIDER", " TypoCloud ")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := newSetupCommand()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, `unknown provider "typocloud"`) {
		t.Fatalf("setup output missing invalid provider detail:\n%s", output)
	}
	if strings.Contains(output, "deepseek /") {
		t.Fatalf("setup output fell back to deepseek for invalid provider:\n%s", output)
	}
}

func TestSetupCommand_AzurePlaceholderDefaultModelRequiresDeployment(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XELYON_PROVIDER", "azure")
	t.Setenv("AZURE_OPENAI_API_KEY", "azure-key")
	t.Setenv("AZURE_OPENAI_BASE_URL", "https://example.openai.azure.com/openai/v1")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := newSetupCommand()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "[todo] Default provider/model: azure / azure-gpt-5.4") {
		t.Fatalf("setup output missing Azure default model todo:\n%s", output)
	}
	if !strings.Contains(output, "deployment is not configured") {
		t.Fatalf("setup output missing Azure deployment guidance:\n%s", output)
	}
	if strings.Contains(output, "[ok] Default provider/model: azure / azure-gpt-5.4") {
		t.Fatalf("setup output incorrectly marks Azure placeholder ready:\n%s", output)
	}
}
