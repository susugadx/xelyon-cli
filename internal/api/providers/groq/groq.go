package groq

import (
	"context"
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/providers/openai"
	openaicompat "github.com/susugadx/xelyon-cli/internal/api/providers/openai_compat"
)

func init() {
	api.RegisterProvider("groq", func(apiKey string) (api.Provider, error) {
		if apiKey == "" {
			return nil, fmt.Errorf("GROQ_API_KEY not set")
		}
		return New(apiKey), nil
	})
}

const (
	groqChatCompletionsEndpointPath = "/openai/v1/chat/completions"
	defaultGroqURL                  = "https://api.groq.com" + groqChatCompletionsEndpointPath
	groqFunctionCallingEnv          = "GROQ_FUNCTION_CALLING"
)

// Provider は Groq API の OpenAI 互換 provider 実装。
type Provider = openaicompat.SimpleProvider

// New は新しい Provider を作成する。
func New(apiKey string) *Provider {
	return openaicompat.NewSimpleProvider(apiKey, openaicompat.SimpleProviderSpec{
		ProviderKey:                       "groq",
		DisplayName:                       "Groq",
		DefaultURL:                        defaultGroqURL,
		URLOverrideEnv:                    "GROQ_API_URL",
		FunctionCallingEnv:                groqFunctionCallingEnv,
		SupportsImages:                    false,
		ThinkingUnsupportedWarning:        "⚠️  Warning: Groq does not support Extended Thinking. Proceeding without it.",
		ImageUnsupportedWarning:           "Warning: Groq does not support image input. The image will be ignored.",
		WarnAndContinueOnStreamParseError: true,
		BuildTools: func(ctx context.Context, tools []api.ToolDefinition) []api.OpenAITool {
			return openai.GetCombinedOpenAIToolsWithContext(ctx, tools)
		},
		EncodeToolCall: openai.ConvertToolCallToToolJSON,
	})
}
