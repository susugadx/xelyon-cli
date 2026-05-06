package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestRunAzureDoctorInvocation_JSONReportsConfiguredDeployment(t *testing.T) {
	resetRootFlagsForTest()
	t.Cleanup(resetRootFlagsForTest)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("AZURE_OPENAI_BASE_URL", "https://example.openai.azure.com/openai/v1")
	t.Setenv("AZURE_OPENAI_API_KEY", "azure-key")
	t.Setenv("AZURE_OPENAI_AUTH_TOKEN", "")
	t.Setenv("AZURE_OPENAI_AUTH_TOKEN_COMMAND", "")

	var out bytes.Buffer
	cmd := newAzureDoctorCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	doctorDeploymentFlag = "corp-gpt55-deployment"
	doctorCatalogModelFlag = "gpt-5.5"
	doctorJSONFlag = true

	if err := runAzureDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runAzureDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	var report struct {
		Provider          string `json:"provider"`
		Deployment        string `json:"deployment"`
		CatalogModel      string `json:"catalog_model"`
		NormalizedBaseURL string `json:"normalized_base_url"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, out.String())
	}
	if report.Provider != "azure" {
		t.Fatalf("provider = %q, want azure", report.Provider)
	}
	if report.Deployment != "corp-gpt55-deployment" {
		t.Fatalf("deployment = %q, want CLI deployment", report.Deployment)
	}
	if report.CatalogModel != "gpt-5.5" {
		t.Fatalf("catalog_model = %q, want CLI catalog model", report.CatalogModel)
	}
	if report.NormalizedBaseURL != "https://example.openai.azure.com/openai/v1" {
		t.Fatalf("normalized_base_url = %q, want v1 URL", report.NormalizedBaseURL)
	}
}

func TestRunAzureDoctorInvocation_FailsForMissingAzureSetup(t *testing.T) {
	resetRootFlagsForTest()
	t.Cleanup(resetRootFlagsForTest)
	t.Setenv("HOME", t.TempDir())
	_ = os.Unsetenv("AZURE_OPENAI_BASE_URL")
	_ = os.Unsetenv("AZURE_OPENAI_API_KEY")
	_ = os.Unsetenv("AZURE_OPENAI_AUTH_TOKEN")
	_ = os.Unsetenv("AZURE_OPENAI_AUTH_TOKEN_COMMAND")

	var out bytes.Buffer
	cmd := newAzureDoctorCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	err := runAzureDoctorInvocation(cmd, nil)
	if err == nil {
		t.Fatalf("runAzureDoctorInvocation() error = nil, want diagnostics failure\noutput:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "base_url") {
		t.Fatalf("output = %q, want base_url failure", out.String())
	}
	if !strings.Contains(out.String(), "deployment") {
		t.Fatalf("output = %q, want deployment failure", out.String())
	}
}

func TestRunAzureDoctorInvocation_PrintConfigDoesNotRequireAzureEnv(t *testing.T) {
	resetRootFlagsForTest()
	t.Cleanup(resetRootFlagsForTest)
	t.Setenv("HOME", t.TempDir())
	_ = os.Unsetenv("AZURE_OPENAI_BASE_URL")
	_ = os.Unsetenv("AZURE_OPENAI_API_KEY")
	_ = os.Unsetenv("AZURE_OPENAI_AUTH_TOKEN")
	_ = os.Unsetenv("AZURE_OPENAI_AUTH_TOKEN_COMMAND")

	var out bytes.Buffer
	cmd := newAzureDoctorCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	doctorDeploymentFlag = "corp-gpt55-deployment"
	doctorCatalogModelFlag = "gpt-5.5"
	doctorPrintConfigFlag = true

	if err := runAzureDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runAzureDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	output := out.String()
	for _, want := range []string{
		"default_provider: azure",
		"provider_models:",
		"default_model: corp-gpt55-deployment",
		"catalog_model: gpt-5.5",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output = %q, want substring %q", output, want)
		}
	}
	if strings.Contains(output, "Azure OpenAI doctor") {
		t.Fatalf("output = %q, should only print YAML snippet", output)
	}
}

func TestRunAzureDoctorInvocation_PrintConfigRequiresDeploymentAndCatalogModel(t *testing.T) {
	resetRootFlagsForTest()
	t.Cleanup(resetRootFlagsForTest)

	var out bytes.Buffer
	cmd := newAzureDoctorCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	doctorPrintConfigFlag = true
	doctorDeploymentFlag = "corp-gpt55-deployment"

	err := runAzureDoctorInvocation(cmd, nil)
	if err == nil {
		t.Fatal("runAzureDoctorInvocation() error = nil, want missing catalog-model error")
	}
	if !strings.Contains(err.Error(), "--catalog-model") {
		t.Fatalf("error = %v, want --catalog-model guidance", err)
	}
	if !cmd.SilenceUsage {
		t.Fatal("cmd.SilenceUsage = false, want true for print-config validation error")
	}
}

func TestRootCommand_AzureDoctorCommandParsesFlags(t *testing.T) {
	resetRootFlagsForTest()
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		resetRootFlagsForTest()
	})
	t.Setenv("HOME", t.TempDir())
	t.Setenv("AZURE_OPENAI_BASE_URL", "https://example.openai.azure.com/openai/v1")
	t.Setenv("AZURE_OPENAI_API_KEY", "azure-key")
	t.Setenv("AZURE_OPENAI_AUTH_TOKEN", "")
	t.Setenv("AZURE_OPENAI_AUTH_TOKEN_COMMAND", "")

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"doctor", "azure", "--deployment", "corp-gpt55-deployment", "--catalog-model", "gpt-5.5", "--json"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root Execute() error = %v\noutput:\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), `"deployment": "corp-gpt55-deployment"`) {
		t.Fatalf("output = %q, want parsed deployment", out.String())
	}
}

func TestRootCommand_AzureDoctorHelpShowsDoctorFlags(t *testing.T) {
	resetRootFlagsForTest()
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		resetRootFlagsForTest()
	})

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"doctor", "azure", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root Execute() error = %v\noutput:\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "--deployment") {
		t.Fatalf("output = %q, want Azure doctor flags", out.String())
	}
	if !strings.Contains(out.String(), "--print-config") {
		t.Fatalf("output = %q, want print-config flag", out.String())
	}
	if !strings.Contains(out.String(), "Diagnose Azure OpenAI configuration") {
		t.Fatalf("output = %q, want Azure doctor help", out.String())
	}
	if strings.Contains(out.String(), "XELYON CLI is an AI coding agent") {
		t.Fatalf("output = %q, should not show root long help", out.String())
	}
}

func TestRootCommand_AzureDoctorFailureDoesNotPrintRootUsage(t *testing.T) {
	resetRootFlagsForTest()
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		resetRootFlagsForTest()
	})
	t.Setenv("HOME", t.TempDir())
	_ = os.Unsetenv("AZURE_OPENAI_BASE_URL")
	_ = os.Unsetenv("AZURE_OPENAI_API_KEY")
	_ = os.Unsetenv("AZURE_OPENAI_AUTH_TOKEN")
	_ = os.Unsetenv("AZURE_OPENAI_AUTH_TOKEN_COMMAND")

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"doctor", "azure"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatalf("root Execute() error = nil, want diagnostics failure\noutput:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "Azure OpenAI doctor") {
		t.Fatalf("output = %q, want doctor report", out.String())
	}
	if strings.Contains(out.String(), "Usage:\n  xelyon [query]") {
		t.Fatalf("output = %q, should not append root usage", out.String())
	}
}

func TestRunKimiDoctorInvocation_JSONReportsExplicitModel(t *testing.T) {
	resetRootFlagsForTest()
	t.Cleanup(resetRootFlagsForTest)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("MOONSHOT_API_KEY", "moonshot-key")
	t.Setenv("KIMI_API_URL", "")
	t.Setenv("XELYON_MODEL", "kimi-k2.6")

	var out bytes.Buffer
	cmd := newKimiDoctorCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := cmd.Flags().Set("model", "kimi-k2.5"); err != nil {
		t.Fatalf("set model flag: %v", err)
	}
	doctorJSONFlag = true

	if err := runKimiDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runKimiDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	var report struct {
		Provider              string `json:"provider"`
		Model                 string `json:"model"`
		PromptCacheKeyPresent bool   `json:"prompt_cache_key_present"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, out.String())
	}
	if report.Provider != "kimi" {
		t.Fatalf("provider = %q, want kimi", report.Provider)
	}
	if report.Model != "kimi-k2.5" {
		t.Fatalf("model = %q, want kimi-k2.5", report.Model)
	}
	if !report.PromptCacheKeyPresent {
		t.Fatal("prompt_cache_key_present = false, want true")
	}
}

func TestRunKimiDoctorInvocation_UsesConfiguredModelWhenFlagOmitted(t *testing.T) {
	resetRootFlagsForTest()
	t.Cleanup(resetRootFlagsForTest)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("MOONSHOT_API_KEY", "moonshot-key")
	t.Setenv("KIMI_API_URL", "")
	t.Setenv("XELYON_MODEL", "kimi-k2.5")

	var out bytes.Buffer
	cmd := newKimiDoctorCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	doctorJSONFlag = true

	if err := runKimiDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runKimiDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	var report struct {
		Model       string `json:"model"`
		ModelSource string `json:"model_source"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, out.String())
	}
	if report.Model != "kimi-k2.5" {
		t.Fatalf("model = %q, want XELYON_MODEL value kimi-k2.5", report.Model)
	}
	if report.ModelSource != "XELYON_MODEL" {
		t.Fatalf("model_source = %q, want XELYON_MODEL", report.ModelSource)
	}
}

func TestRunKimiDoctorInvocation_FailsForMissingKey(t *testing.T) {
	resetRootFlagsForTest()
	t.Cleanup(resetRootFlagsForTest)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("MOONSHOT_API_KEY", "")
	t.Setenv("KIMI_API_URL", "")
	t.Setenv("XELYON_MODEL", "")

	var out bytes.Buffer
	cmd := newKimiDoctorCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	err := runKimiDoctorInvocation(cmd, nil)
	if err == nil {
		t.Fatalf("runKimiDoctorInvocation() error = nil, want diagnostics failure\noutput:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "MOONSHOT_API_KEY") {
		t.Fatalf("output = %q, want MOONSHOT_API_KEY failure", out.String())
	}
}

func TestRootCommand_KimiDoctorCommandParsesFlags(t *testing.T) {
	resetRootFlagsForTest()
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		resetRootFlagsForTest()
	})
	t.Setenv("HOME", t.TempDir())
	t.Setenv("MOONSHOT_API_KEY", "moonshot-key")
	t.Setenv("KIMI_API_URL", "")
	t.Setenv("XELYON_MODEL", "")

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"doctor", "kimi", "--model", "kimi-k2.5", "--json"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root Execute() error = %v\noutput:\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), `"model": "kimi-k2.5"`) {
		t.Fatalf("output = %q, want parsed Kimi model", out.String())
	}
}

func TestRootCommand_KimiDoctorHelpShowsDoctorFlags(t *testing.T) {
	resetRootFlagsForTest()
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		resetRootFlagsForTest()
	})

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"doctor", "kimi", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root Execute() error = %v\noutput:\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "--model") {
		t.Fatalf("output = %q, want Kimi doctor model flag", out.String())
	}
	if !strings.Contains(out.String(), "--tool-smoke") {
		t.Fatalf("output = %q, want Kimi doctor tool smoke flag", out.String())
	}
	if !strings.Contains(out.String(), "Diagnose Kimi native provider configuration") {
		t.Fatalf("output = %q, want Kimi doctor help", out.String())
	}
}
