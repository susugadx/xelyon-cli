package prompt

import "strings"

// commonExecutionBlock は複数プロバイダーで共通する実行指針
// ガードよりも、調査と編集の進め方を短く揃える
const commonExecutionBlock = `### Shared Notes
- When the exact edit target is already known, prefer search_code -> str_replace(line-range)
- When scope is unclear, read broadly first; omit line ranges to read up to 300 lines
- Check all usages before changing shared functions or types
- Wait for build/test output before declaring completion
- Follow project-specific rules from Project Context when present
`

// commonBoostBlock は全プロバイダー共通の品質向上ルール
// ガード（〜するな）ではなく、判断力を引き出すブースト指示
const commonBoostBlock = `### Quality Boost
- For investigation tasks, read enough surrounding code to understand callers, types, helpers, and tests
- If you are unsure between two edit scopes, gather more context before editing; an extra read is cheaper than a wrong patch
- Match existing project patterns for validation, defaults, and tests
- Before reporting completion, self-review changed call sites, imports, and error paths
`

// providerPrefixes はプロバイダー別のシステムプロンプトプレフィックス
// Workflow Rules の直前に挿入し、共通指針 + プロバイダー固有ノート + 共通ブーストで構成する
var providerPrefixes = map[string]string{
	"gemini": "## Provider Notes\n" + commonExecutionBlock +
		"### Gemini-specific\n" +
		"- Tool calls must be raw JSON, not markdown code blocks\n" +
		"- Do not claim a command ran unless you actually called bash\n" +
		"- Respond in the same language as the user's message\n" +
		"- Call tools directly for action requests instead of previewing hypothetical edits\n" +
		"- Edit the original file directly; do not create derivative temp files\n" +
		"- Make str_replace old_str unique by including enough surrounding context\n\n" +
		commonBoostBlock,
	"deepseek": "## Provider Notes\n" + commonExecutionBlock +
		"### DeepSeek-specific\n" +
		"- When function calling is enabled, use tool calls for file operations instead of plain-text descriptions\n" +
		"- Fix errors completely; do not leave TODO-style excuses such as \"for brevity\" or \"due to time constraints\"\n" +
		"- After str_replace, remove any unused imports introduced by the edit before moving on\n" +
		"- After read_file, either edit or gather more context; do not echo file contents back to the user\n" +
		"- You are already in the project root directory; do not prefix commands with `cd /path &&`\n\n" +
		commonBoostBlock,
	"groq": "## Provider Notes\n" + commonExecutionBlock +
		"### Groq-specific\n" +
		"- Tool calls must be raw JSON, not markdown code blocks or XML wrappers\n" +
		"- Respond in the same language as the user's message\n\n" +
		commonBoostBlock,
	"openai": "## Provider Notes\n" + commonExecutionBlock +
		"### OpenAI-specific\n" +
		"- When the user instructs execution, proceed without asking for confirmation; choose the safer reasonable option and move forward\n" +
		"- Fix compile errors at the root cause; do not delete existing logic just to make the build pass\n" +
		"- When recovering from errors, prefer targeted str_replace edits over whole-file rewrites\n" +
		"- For str_replace with mixed Japanese, JSON, or backticks, split the change into smaller edits to avoid byte corruption\n" +
		"- When removing functionality, clean up dead imports, dead code, and orphaned dependencies in the same task\n\n" +
		commonBoostBlock,
	// "claude" と "anthropic" は同一プロバイダー（anthropic はエイリアス）
	// GetProviderPrefix() 内でエイリアス正規化するため "claude" のみ定義
	"claude": "## Provider Notes\n" + commonExecutionBlock +
		"### Claude-specific\n" +
		"- Use search_code for file search instead of bash grep/rg/find\n" +
		"- Use read_file for file contents instead of cat/head/tail/sed\n" +
		"- When operations are independent, use parallel tool calls in a single response\n\n" +
		commonBoostBlock,
	// openrouter / ollama: 裏のモデルが不定だが、共通ルール + ブーストは害がない
	"openrouter": "## Provider Notes\n" + commonExecutionBlock + "\n" + commonBoostBlock,
	"ollama":     "## Provider Notes\n" + commonExecutionBlock + "\n" + commonBoostBlock,
}

// providerAliases はプロバイダー名のエイリアスを正規名に変換するマップ
var providerAliases = map[string]string{
	"anthropic": "claude",
	"bedrock":   "claude", // AWS Bedrock の裏は Claude
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

const workflowRulesHeader = "\n## Workflow Rules\n"

// BuildProviderSystemPrompt はプロバイダー別ノートを Workflow Rules の直前に挿入する
func BuildProviderSystemPrompt(base, providerName string) string {
	prefix := strings.TrimSpace(GetProviderPrefix(providerName))
	if prefix == "" {
		return base
	}
	idx := strings.Index(base, workflowRulesHeader)
	if idx < 0 {
		return prefix + "\n\n" + base
	}
	return base[:idx] + "\n\n" + prefix + "\n" + base[idx:]
}
