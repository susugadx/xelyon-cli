package prompt

import "strings"

// commonRulesBlock は複数プロバイダーで共通する重要ルール
// Gemini・DeepSeek 等、指示を無視しやすいモデルに冒頭で強制注入する
const commonRulesBlock = `1. **ALWAYS read_file BEFORE str_replace** - NO EXCEPTIONS
2. **NEVER guess file contents** - read first, edit second
3. If you haven't read it this session, you CANNOT edit it
4. **After running bash verification (test/build/lint), WAIT for the result** - do NOT declare completion until you see the output
5. **Before deleting any type or function, use lsp_find to check ALL references first** - deletion without reference check is FORBIDDEN
6. **Before fixing code, use search_code to identify ALL affected locations first** - blind editing is FORBIDDEN
7. **Read project root config files (XELYON.md etc.) before starting any task** - understand project rules first
`

// providerPrefixes はプロバイダー別のシステムプロンプトプレフィックス
// クリティカルルールを冒頭に注入してモデルの遵守率を上げる
// 共通ルール (commonRulesBlock) + プロバイダー固有ルールで構成
var providerPrefixes = map[string]string{
	"gemini": "## ⚠️ ABSOLUTE RULES (NEVER SKIP)\n" + commonRulesBlock +
		"8. **Tool calls must be actual JSON, NOT inside markdown code blocks** - " + "```json...```" + " is for display only\n\n",
	"deepseek": "## ⚠️ ABSOLUTE RULES (NEVER SKIP)\n" + commonRulesBlock +
		"8. **When function calling is enabled, ALWAYS use tool calls for file operations** - do NOT output raw JSON or describe actions in plain text\n" +
		"9. **If LSP errors remain, fix ALL of them before moving to the next file** - never leave errors behind\n" +
		"10. **NEVER leave errors unfixed with excuses** like \"due to time constraints\" or \"for brevity\" - fix every error completely\n" +
		"11. **After str_replace, if unused imports appear, remove them IMMEDIATELY** - do NOT proceed with unused import errors\n\n",
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
