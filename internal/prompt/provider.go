package prompt

import "strings"

// commonRulesBlock は複数プロバイダーで共通する重要ルール
// Gemini・DeepSeek 等、指示を無視しやすいモデルに冒頭で強制注入する
const commonRulesBlock = `1. **read_file BEFORE str_replace (old_str mode)** - if you haven't read the file, you CANNOT edit it. Exception: line-range mode (start_line/end_line) works after search_code for matched lines
2. **Before changing/deleting any function or type**: search_code to check ALL usages first — NOT bash (grep/rg)
3. **After bash verification (test/build), WAIT for output** - do NOT declare completion before seeing results
4. **Follow project rules in Project Context** - they are LAW - override all other guidelines
5. **Same change across files? Use grep_replace (1 call)** - do NOT repeat str_replace for each file
   Example: {"tool":"grep_replace","args":{"pattern":"oldFunc\\(","replacement":"newFunc(","path":".","file_pattern":"*.go"}}
6. **Code search → search_code, NOT bash (grep/rg)** - search_code caches results, marks read-ranges for str_replace, and detects [def]/[ref] blocks
`

// providerPrefixes はプロバイダー別のシステムプロンプトプレフィックス
// クリティカルルールを冒頭に注入してモデルの遵守率を上げる
// 共通ルール (commonRulesBlock) + プロバイダー固有ルールで構成
var providerPrefixes = map[string]string{
	"gemini": "## ⚠️ ABSOLUTE RULES (NEVER SKIP)\n" + commonRulesBlock +
		"7. **Tool calls must be actual JSON, NOT inside markdown code blocks** - " + "```json...```" + " is for display only\n" +
		"8. **NEVER claim you ran a command without actually calling bash** - always show the actual tool call\n" +
		"9. **ALWAYS respond in the same language as the user's message** - if the user writes in Japanese, respond in Japanese\n" +
		"10. **NEVER explain or show code before tool calls** - Just call the tool directly without code blocks or previews\n" +
		"11. **NEVER create derivative/copy files** (e.g. file.go_temp, file.go.new) - edit the original file directly with str_replace\n" +
		"12. **str_replace old_str must be UNIQUE** - include surrounding lines (before/after) so the match is unambiguous. If the same string appears in multiple places, use grep_replace instead\n\n",
	"deepseek": "## ⚠️ ABSOLUTE RULES (NEVER SKIP)\n" + commonRulesBlock +
		"7. **When function calling is enabled, ALWAYS use tool calls for file operations** - do NOT output raw JSON or describe actions in plain text\n" +
		"8. **Fix ALL errors completely** - NEVER leave errors with excuses like \"due to time constraints\" or \"for brevity\"\n" +
		"9. **After str_replace, if unused imports appear, remove them IMMEDIATELY** - do NOT proceed with unused import errors\n" +
		"10. **You are already in the project root directory. Do NOT prefix commands with `cd /path && `** - just run commands directly (e.g. `grep -n 'pattern' file.go`, NOT `cd /home/user/project && grep -n 'pattern' file.go`)\n\n",
	"groq": "## ⚠️ ABSOLUTE RULES (NEVER SKIP)\n" + commonRulesBlock +
		"7. **Tool calls MUST be raw JSON** - NEVER wrap in markdown code blocks or use XML like `<tool_name><param>value</param></tool_name>`\n" +
		"8. **ALWAYS respond in the same language as the user's message** - if the user writes in Japanese, respond in Japanese\n\n",
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
