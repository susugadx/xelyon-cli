package prompt

import "strings"

// commonRulesBlock は複数プロバイダーで共通する重要ルール
// Gemini・DeepSeek 等、指示を無視しやすいモデルに冒頭で強制注入する
const commonRulesBlock = `1. **ALWAYS read_file BEFORE str_replace** - if you haven't read it, you CANNOT edit it
2. **Before changing/deleting any function or type**: lsp_find(references) or search_code to check ALL usages first
3. **After bash verification (test/build), WAIT for output** - do NOT declare completion before seeing results
4. **Read XELYON.md before starting any task** - its rules are LAW - override all other guidelines
5. **Same change across 3+ files? Use grep_replace (1 call)** - do NOT repeat str_replace for each file
`

// providerPrefixes はプロバイダー別のシステムプロンプトプレフィックス
// クリティカルルールを冒頭に注入してモデルの遵守率を上げる
// 共通ルール (commonRulesBlock) + プロバイダー固有ルールで構成
var providerPrefixes = map[string]string{
	"gemini": "## ⚠️ ABSOLUTE RULES (NEVER SKIP)\n" + commonRulesBlock +
		"6. **Tool calls must be actual JSON, NOT inside markdown code blocks** - " + "```json...```" + " is for display only\n" +
		"7. **NEVER claim you ran a command without actually calling bash** - always show the actual tool call\n" +
		"8. **ALWAYS respond in the same language as the user's message** - if the user writes in Japanese, respond in Japanese\n\n",
	"deepseek": "## ⚠️ ABSOLUTE RULES (NEVER SKIP)\n" + commonRulesBlock +
		"6. **When function calling is enabled, ALWAYS use tool calls for file operations** - do NOT output raw JSON or describe actions in plain text\n" +
		"7. **Fix ALL errors completely** - NEVER leave errors with excuses like \"due to time constraints\" or \"for brevity\"\n" +
		"8. **After str_replace, if unused imports appear, remove them IMMEDIATELY** - do NOT proceed with unused import errors\n\n",
	"groq": "## ⚠️ ABSOLUTE RULES (NEVER SKIP)\n" + commonRulesBlock +
		"6. **Tool calls MUST be raw JSON** - NEVER wrap in markdown code blocks or use XML like `<tool_name><param>value</param></tool_name>`\n" +
		"7. **ALWAYS respond in the same language as the user's message** - if the user writes in Japanese, respond in Japanese\n\n",
}

// GetProviderPrefix はプロバイダー名に応じたプレフィックスを返す
// 未登録プロバイダーは空文字を返す
func GetProviderPrefix(provider string) string {
	return providerPrefixes[strings.ToLower(provider)]
}

// BuildProviderSystemPrompt はプロバイダー別プレフィックスをシステムプロンプトの冒頭に注入する
func BuildProviderSystemPrompt(base, providerName string) string {
	prefix := GetProviderPrefix(providerName)
	if prefix == "" {
		return base
	}
	return prefix + base
}
