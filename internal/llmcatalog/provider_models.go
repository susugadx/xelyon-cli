package llmcatalog

import "strings"

var knownProviderModels = map[string][]string{
	"deepseek": {
		"deepseek-v4-flash",
		"deepseek-v4-pro",
		"deepseek-chat",
		"deepseek-reasoner",
		"deepseek-coder",
	},
	"kimi": {
		"kimi-k2",
		"kimi-k2.5",
		"kimi-k2.6",
		"kimi-k2-thinking",
	},
	"openai": {
		"gpt-5.5",
		"gpt-5.5-pro",
		"gpt-5.4",
		"gpt-5.4-pro",
		"gpt-5.4-mini",
		"gpt-5.4-nano",
		"gpt-5.3-codex",
		"gpt-5.2",
		"gpt-5.1",
		"gpt-5",
		"gpt-4.1",
		"gpt-4.1-mini",
		"gpt-4.1-nano",
		"gpt-4o",
		"gpt-4o-mini",
		"o3-mini",
		"o1",
	},
	"gemini": {
		"gemini-3.5-flash",
		"gemini-3.1-flash-lite",
		"gemini-3.1-pro-preview-customtools",
		"gemini-2.5-pro",
		"gemini-2.5-flash",
	},
	"claude": {
		"claude-sonnet-4-6",
		"claude-sonnet-4.6",
		"claude-opus-4-7",
		"claude-opus-4.7",
		"claude-sonnet-4-5",
		"claude-sonnet-4.5",
		"claude-opus-4-6",
		"claude-opus-4-5",
		"claude-haiku-4-5-20251001",
		"claude-3-5-sonnet-20241022",
		"claude-3-5-haiku-20241022",
	},
	"ollama": {
		"qwen2.5-coder:7b",
		"qwen2.5-coder:14b",
		"qwen2.5-coder:32b",
		"codellama:7b",
		"codellama:13b",
		"codellama:34b",
		"deepseek-coder-v2",
		"llama3:8b",
		"llama3:70b",
		"mistral:7b",
	},
	"groq": {
		"meta-llama/llama-4-scout-17b-16e-instruct",
		"llama-3.3-70b-versatile",
		"llama-3.1-70b-versatile",
		"llama-3.1-8b-instant",
		"mixtral-8x7b-32768",
		"gemma2-9b-it",
	},
	"openrouter": {
		"anthropic/claude-sonnet-4.6",
		"anthropic/claude-opus-4.7",
		"openai/gpt-5.5",
		"openai/gpt-5.4",
		"openai/gpt-5.4-mini",
		"openai/gpt-5.3-codex",
		"google/gemini-3.1-pro-preview",
		"google/gemini-2.5-flash",
		"deepseek/deepseek-v4-flash",
		"meta-llama/llama-4-scout-17b-16e-instruct",
	},
	"bedrock": {
		"global.anthropic.claude-sonnet-4-6",
		"us.anthropic.claude-sonnet-4-6",
		"eu.anthropic.claude-sonnet-4-6",
		"au.anthropic.claude-sonnet-4-6",
		"global.anthropic.claude-opus-4-7-v1",
		"global.anthropic.claude-opus-4-7-v1:0",
		"amazon.nova-pro-v1:0",
		"amazon.nova-lite-v1:0",
		"us.amazon.nova-pro-v1:0",
		"us.meta.llama4-scout-17b-instruct-v1:0",
		"us.deepseek.r1-v1:0",
		"moonshotai.kimi-k2.5",
		"moonshotai.kimi-k2-thinking",
		"qwen.qwen3-coder-480b-a35b-instruct-v1:0",
		"minimax.minimax-m2",
	},
}

var knownProviderModelPrefixes = map[string][]string{
	"deepseek": {
		"deepseek-v4",
		"deepseek-v3",
		"deepseek-r1",
		"deepseek-coder",
	},
	"kimi": {
		"kimi-",
	},
	"groq": {
		"meta-llama/llama-4-scout",
		"llama-4-scout",
		"llama-3.",
		"llama-3-",
		"mixtral-",
		"gemma2-",
	},
	"gemini": {
		"gemini-",
	},
}

// KnownModelNamesForProvider は picker 表示用の既知 model 名を provider ごとの安定順で返す。
// Azure は deployment 名をユーザー環境が所有するため、ここでは候補を返さない。
func KnownModelNamesForProvider(provider string) []string {
	key := CanonicalProviderKey(provider)
	if key == "azure" {
		return nil
	}
	models := knownProviderModels[key]
	return cloneStrings(models)
}

// IsKnownModelNameForProvider は model が provider 所有の既知 catalog 名または既知 prefix か返す。
func IsKnownModelNameForProvider(provider, model string) bool {
	key := CanonicalProviderKey(provider)
	model = strings.TrimSpace(model)
	if key == "" || model == "" {
		return false
	}

	for _, known := range KnownModelNamesForProvider(key) {
		if strings.EqualFold(model, known) {
			return true
		}
	}

	normalized := strings.ToLower(model)
	for _, prefix := range knownProviderModelPrefixes[key] {
		if strings.HasPrefix(normalized, strings.ToLower(strings.TrimSpace(prefix))) {
			return true
		}
	}
	return false
}
