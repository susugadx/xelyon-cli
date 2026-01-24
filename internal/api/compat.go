package api

import (
	"fmt"
	"os"
)

// このファイルは後方互換性のためのコンストラクタを提供します。
// 注意: 型エイリアスは循環参照を引き起こすため使用できません。
// 具体的な型が必要な場合は providers/* パッケージを直接インポートしてください。

// NewDeepSeekProvider は新しい DeepSeek Provider を作成（後方互換）
// 戻り値は Provider インターフェースです。
func NewDeepSeekProvider(apiKey string) Provider {
	factory, ok := getRegisteredProvider("deepseek")
	if !ok {
		return nil
	}
	provider, err := factory(apiKey)
	if err != nil {
		return nil
	}
	return provider
}

// NewOllamaProvider は新しい Ollama Provider を作成（後方互換）
func NewOllamaProvider(baseURL string) Provider {
	factory, ok := getRegisteredProvider("ollama")
	if !ok {
		return nil
	}
	provider, err := factory(baseURL)
	if err != nil {
		return nil
	}
	return provider
}

// NewGroqProvider は新しい Groq Provider を作成（後方互換）
func NewGroqProvider(apiKey string) Provider {
	factory, ok := getRegisteredProvider("groq")
	if !ok {
		return nil
	}
	provider, err := factory(apiKey)
	if err != nil {
		return nil
	}
	return provider
}

// NewClaudeProvider は新しい Claude Provider を作成（後方互換）
func NewClaudeProvider(apiKey string) Provider {
	factory, ok := getRegisteredProvider("claude")
	if !ok {
		return nil
	}
	provider, err := factory(apiKey)
	if err != nil {
		return nil
	}
	return provider
}

// NewOpenAIProvider は新しい OpenAI Provider を作成（後方互換）
func NewOpenAIProvider(apiKey string) Provider {
	factory, ok := getRegisteredProvider("openai")
	if !ok {
		return nil
	}
	provider, err := factory(apiKey)
	if err != nil {
		return nil
	}
	return provider
}

// NewGeminiProvider は新しい Gemini Provider を作成（後方互換）
func NewGeminiProvider(apiKey string) Provider {
	factory, ok := getRegisteredProvider("gemini")
	if !ok {
		return nil
	}
	provider, err := factory(apiKey)
	if err != nil {
		return nil
	}
	return provider
}

// ChatWithTools はツール対応の会話を行う（ストリーミング）
// Deprecated: 後方互換性のために残されています。Provider + Client を使用してください。
func ChatWithTools(systemPrompt string, history []Message, model string) (string, error) {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("DEEPSEEK_API_KEY not set")
	}

	provider := NewDeepSeekProvider(apiKey)
	if provider == nil {
		return "", fmt.Errorf("failed to create DeepSeek provider")
	}
	client := NewClient(provider)
	return client.ChatWithTools(systemPrompt, history, model)
}

// AskDeepSeekStream は従来のストリーミング質問（後方互換）
// Deprecated: 後方互換性のために残されています。ChatWithTools を使用してください。
func AskDeepSeekStream(query string, context string, model string) (string, error) {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("DEEPSEEK_API_KEY not set")
	}

	systemPrompt := "You are an excellent programming assistant. Answer based on the provided context. When modifying code, always present the complete code wrapped in triple backticks."

	userContent := query
	if context != "" {
		userContent = fmt.Sprintf("## Context:\n%s\n\n## Question:\n%s", context, query)
	}

	provider := NewDeepSeekProvider(apiKey)
	if provider == nil {
		return "", fmt.Errorf("failed to create DeepSeek provider")
	}
	client := NewClient(provider)

	history := []Message{
		{Role: "user", Content: userContent},
	}

	return client.ChatWithTools(systemPrompt, history, model)
}
