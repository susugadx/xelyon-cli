package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestDefaultCompressionModel_OpenAI(t *testing.T) {
	if got := defaultCompressionModel("openai"); got != "gpt-5.4-mini" {
		t.Fatalf("defaultCompressionModel(openai) = %q, want %q", got, "gpt-5.4-mini")
	}
}

func TestDefaultCompressionModel_Gemini(t *testing.T) {
	if got := defaultCompressionModel("gemini"); got != "gemini-3.1-flash-lite-preview" {
		t.Fatalf("defaultCompressionModel(gemini) = %q, want %q", got, "gemini-3.1-flash-lite-preview")
	}
}

func TestDefaultCompressionModel_Claude(t *testing.T) {
	if got := defaultCompressionModel("claude"); got != "claude-haiku-4-5" {
		t.Fatalf("defaultCompressionModel(claude) = %q, want %q", got, "claude-haiku-4-5")
	}
}

func TestDefaultCompressionModel_AnthropicAlias(t *testing.T) {
	if got := defaultCompressionModel("anthropic"); got != "claude-haiku-4-5" {
		t.Fatalf("defaultCompressionModel(anthropic) = %q, want %q", got, "claude-haiku-4-5")
	}
}

func TestDefaultCompressionModel_AzureUsesCurrentDeploymentFallback(t *testing.T) {
	if got := defaultCompressionModel("azure"); got != "" {
		t.Fatalf("defaultCompressionModel(azure) = %q, want empty so current deployment is used", got)
	}
}

func TestCompressWithModel_Fallback(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Compression.Model = ""

	agent := &Agent{
		CurrentModel: "llama-3.3-70b",
		ProviderName: "groq",
		Runtime:      NewAgentRuntimeWithConfig(cfg),
	}

	if got := agent.getCompressionModel(); got != "llama-3.3-70b" {
		t.Fatalf("getCompressionModel() = %q, want current model", got)
	}
}

func TestCompressWithModel_AzureUsesCurrentDeployment(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Compression.Model = ""

	agent := &Agent{
		CurrentModel: "corp-gpt55-deployment",
		ProviderName: "azure",
		Runtime:      NewAgentRuntimeWithConfig(cfg),
	}

	if got := agent.getCompressionModel(); got != "corp-gpt55-deployment" {
		t.Fatalf("getCompressionModel() = %q, want current Azure deployment", got)
	}
}
