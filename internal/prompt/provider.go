package prompt

import "strings"

// commonRulesBlock は複数プロバイダーで共通する重要ルール
// Gemini・DeepSeek 等、指示を無視しやすいモデルに冒頭で強制注入する
const commonRulesBlock = `1. **search_code → str_replace(line-range) is PREFERRED** — search_code marks matched lines as read, no read_file needed. Use read_file → str_replace(old_str) only when line-range is not applicable
2. **Before changing/deleting any function or type**: search_code to check ALL usages first — NOT bash (grep/rg)
3. **After bash verification (test/build), WAIT for output** - do NOT declare completion before seeing results
4. **Follow project rules in Project Context** - they are LAW - override all other guidelines
5. **Code search → search_code, NOT bash (grep/rg)** - search_code caches results, marks read-ranges for str_replace, and detects [def]/[ref] blocks
6. **Same change in multiple places? Use str_replace batch mode** - pass edits=[{old_str,new_str},...] in one call instead of repeating str_replace for each location
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
		"12. **str_replace old_str must be UNIQUE** - include surrounding lines (before/after) so the match is unambiguous\n\n",
	"deepseek": "## ⚠️ ABSOLUTE RULES (NEVER SKIP)\n" + commonRulesBlock +
		"7. **When function calling is enabled, ALWAYS use tool calls for file operations** - do NOT output raw JSON or describe actions in plain text\n" +
		"8. **Fix ALL errors completely** - NEVER leave errors with excuses like \"due to time constraints\" or \"for brevity\"\n" +
		"9. **After str_replace, if unused imports appear, remove them IMMEDIATELY** - do NOT proceed with unused import errors\n" +
		"10. **NEVER output file contents or explain code between read_file and str_replace — call str_replace IMMEDIATELY after reading** - no commentary, no summaries, just the tool call\n" +
		"11. **You are already in the project root directory. Do NOT prefix commands with `cd /path && `** - just run commands directly (e.g. `grep -n 'pattern' file.go`, NOT `cd /home/user/project && grep -n 'pattern' file.go`)\n\n",
	"groq": "## ⚠️ ABSOLUTE RULES (NEVER SKIP)\n" + commonRulesBlock +
		"7. **Tool calls MUST be raw JSON** - NEVER wrap in markdown code blocks or use XML like `<tool_name><param>value</param></tool_name>`\n" +
		"8. **ALWAYS respond in the same language as the user's message** - if the user writes in Japanese, respond in Japanese\n\n",
	"openai": "## ⚠️ ABSOLUTE RULES (NEVER SKIP)\n" + commonRulesBlock +
		"7. **When user instructs execution, execute immediately WITHOUT asking for confirmation** - do NOT say \"shall I proceed?\" or \"is it OK to continue?\". Make your own judgment and proceed. When in doubt, choose the safer option and move forward\n" +
		"8. **NEVER delete existing logic to fix a compile error** - if a function call, branch, or import causes a compile error, fix the root cause (wrong signature, missing import, type mismatch). Do NOT remove the code and justify it with comments like \"handled elsewhere\" or \"not needed here\"\n" +
		"9. **When recovering from errors, fix ONLY the broken line/call** - do NOT rewrite the entire file with write_file. Use str_replace on the specific error site. Preserve all existing branches and logic\n" +
		"10. **For str_replace with mixed Japanese/JSON/backticks, split into small edits** - large multi-encoding replacements risk byte corruption. Apply one logical change per str_replace call\n\n",
	// "claude" と "anthropic" は同一プロバイダー（anthropic はエイリアス）
	// GetProviderPrefix() 内でエイリアス正規化するため "claude" のみ定義
	"claude": "## ⚠️ ABSOLUTE RULES (NEVER SKIP)\n" + commonRulesBlock +
		"7. **File search → search_code, NOT bash (grep/rg/find)** - bash is for build/test/run commands ONLY (e.g. make, go test, npm run)\n" +
		"8. **File reading → read_file, NOT bash (cat/head/tail/sed)** - use read_file with line ranges for targeted reading\n" +
		"9. **ALWAYS use parallel tool calls** - When operations are independent (e.g., searching def + ref, editing multiple files, reading + searching), call ALL tools in a single response. One-at-a-time calls for independent operations are FORBIDDEN — they double token costs\n\n",
}

// providerAliases はプロバイダー名のエイリアスを正規名に変換するマップ
var providerAliases = map[string]string{
	"anthropic": "claude",
}

// GetProviderPrefix はプロバイダー名に応じたプレフィックスを返す
// 未登録プロバイダーは空文字を返す
func GetProviderPrefix(provider string) string {
	name := strings.ToLower(provider)
	if canonical, ok := providerAliases[name]; ok {
		name = canonical
	}
	return providerPrefixes[name]
}

// BuildProviderSystemPrompt はプロバイダー別プレフィックスをシステムプロンプトの冒頭に注入する
func BuildProviderSystemPrompt(base, providerName string) string {
	prefix := GetProviderPrefix(providerName)
	if prefix == "" {
		return base
	}
	return prefix + base
}
