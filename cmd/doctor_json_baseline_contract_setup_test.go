package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func setupDeepSeekJSONBaselineContract(t *testing.T, cmd *cobra.Command) doctorJSONBaselineSetupContract {
	t.Helper()
	setCommonDoctorJSONBaselineEnv(t)
	t.Setenv("DEEPSEEK_API_KEY", "sk-test")
	t.Setenv("DEEPSEEK_API_URL", "")
	t.Setenv("DEEPSEEK_FUNCTION_CALLING", "1")
	setDoctorCommandFlag(t, cmd, "model", "corp-deepseek-model")
	setDoctorCommandFlag(t, cmd, "catalog-model", "deepseek-v4-flash")
	return doctorJSONBaselineSetupContract{}
}

func setupGroqJSONBaselineContract(t *testing.T, cmd *cobra.Command) doctorJSONBaselineSetupContract {
	t.Helper()
	setCommonDoctorJSONBaselineEnv(t)
	t.Setenv("GROQ_API_KEY", "gsk-test")
	t.Setenv("GROQ_API_URL", "")
	t.Setenv("GROQ_FUNCTION_CALLING", "1")
	setDoctorCommandFlag(t, cmd, "model", "corp-groq-model")
	setDoctorCommandFlag(t, cmd, "catalog-model", "meta-llama/llama-4-scout-17b-16e-instruct")
	return doctorJSONBaselineSetupContract{}
}

func setupOllamaJSONBaselineContract(t *testing.T, cmd *cobra.Command) doctorJSONBaselineSetupContract {
	t.Helper()
	setCommonDoctorJSONBaselineEnv(t)
	server := newOllamaDoctorCommandTestServer(t, []string{"qwen2.5-coder:7b"}, nil)
	t.Cleanup(server.Close)
	t.Setenv("OLLAMA_BASE_URL", server.URL)
	t.Setenv("OLLAMA_FUNCTION_CALLING", "1")
	setDoctorCommandFlag(t, cmd, "model", "qwen2.5-coder:7b")
	setDoctorCommandFlag(t, cmd, "catalog-model", "qwen2.5-coder:7b")
	return doctorJSONBaselineSetupContract{apiURL: server.URL}
}

func setupOpenRouterJSONBaselineContract(t *testing.T, cmd *cobra.Command) doctorJSONBaselineSetupContract {
	t.Helper()
	setCommonDoctorJSONBaselineEnv(t)
	t.Setenv("OPENROUTER_API_KEY", "sk-or-test")
	t.Setenv("OPENROUTER_API_URL", "")
	t.Setenv("OPENROUTER_FUNCTION_CALLING", "1")
	setDoctorCommandFlag(t, cmd, "model", "anthropic/claude-sonnet-4.6")
	setDoctorCommandFlag(t, cmd, "catalog-model", "anthropic/claude-sonnet-4.6")
	return doctorJSONBaselineSetupContract{}
}

func setupOpenAIJSONBaselineContract(t *testing.T, cmd *cobra.Command) doctorJSONBaselineSetupContract {
	t.Helper()
	setCommonDoctorJSONBaselineEnv(t)
	t.Setenv("OPENAI_API_KEY", "sk-test")
	t.Setenv("OPENAI_API_URL", "")
	t.Setenv("OPENAI_RESPONSES_URL", "")
	t.Setenv("OPENAI_FUNCTION_CALLING", "1")
	setDoctorCommandFlag(t, cmd, "model", "corp-openai-deployment")
	setDoctorCommandFlag(t, cmd, "catalog-model", "gpt-5.4")
	return doctorJSONBaselineSetupContract{}
}

func setupAzureJSONBaselineContract(t *testing.T, cmd *cobra.Command) doctorJSONBaselineSetupContract {
	t.Helper()
	setCommonDoctorJSONBaselineEnv(t)
	t.Setenv("AZURE_OPENAI_BASE_URL", "https://example.openai.azure.com/openai/v1")
	t.Setenv("AZURE_OPENAI_API_KEY", "azure-key")
	t.Setenv("AZURE_OPENAI_AUTH_TOKEN", "")
	t.Setenv("AZURE_OPENAI_AUTH_TOKEN_COMMAND", "")
	t.Setenv("AZURE_OPENAI_FUNCTION_CALLING", "1")
	setDoctorCommandFlag(t, cmd, "deployment", "corp-codex-deployment")
	setDoctorCommandFlag(t, cmd, "catalog-model", "gpt-5.3-codex")
	return doctorJSONBaselineSetupContract{}
}

func setupKimiJSONBaselineContract(t *testing.T, cmd *cobra.Command) doctorJSONBaselineSetupContract {
	t.Helper()
	setKimiDoctorCommandTestEnv(t, "moonshot-key")
	t.Setenv("XELYON_MODEL", "kimi-k2.6")
	t.Setenv("KIMI_FUNCTION_CALLING", "1")
	setDoctorCommandFlag(t, cmd, "model", "kimi-k2.5")
	setDoctorCommandFlag(t, cmd, "catalog-model", "kimi-k2.6")
	return doctorJSONBaselineSetupContract{}
}

func setupGeminiJSONBaselineContract(t *testing.T, cmd *cobra.Command) doctorJSONBaselineSetupContract {
	t.Helper()
	setGeminiDoctorCommandTestEnv(t, "gemini-key")
	t.Setenv("GEMINI_CONTEXT_CACHING", "")
	t.Setenv("GEMINI_FC_MODE", "")
	setDoctorCommandFlag(t, cmd, "model", "corp-gemini-model")
	setDoctorCommandFlag(t, cmd, "catalog-model", "gemini-3.1-pro-preview-customtools")
	return doctorJSONBaselineSetupContract{}
}

func setupClaudeJSONBaselineContract(t *testing.T, cmd *cobra.Command) doctorJSONBaselineSetupContract {
	t.Helper()
	setClaudeDoctorCommandTestEnv(t, "claude-key")
	t.Setenv("CLAUDE_FUNCTION_CALLING", "1")
	setDoctorCommandFlag(t, cmd, "model", "corp-claude-model")
	setDoctorCommandFlag(t, cmd, "catalog-model", "claude-sonnet-4-6")
	return doctorJSONBaselineSetupContract{}
}

func setupBedrockJSONBaselineContract(t *testing.T, cmd *cobra.Command) doctorJSONBaselineSetupContract {
	t.Helper()
	setBedrockDoctorCommandTestEnv(t)
	t.Setenv("BEDROCK_FUNCTION_CALLING", "1")
	setDoctorCommandFlag(t, cmd, "model", "corp-bedrock-sonnet")
	setDoctorCommandFlag(t, cmd, "catalog-model", bedrockDoctorCatalogModelForTest)
	return doctorJSONBaselineSetupContract{}
}

func setCommonDoctorJSONBaselineEnv(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XELYON_MODEL", "")
}
