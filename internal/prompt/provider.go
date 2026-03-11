package prompt

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// providerPrefixes はプロバイダー別のシステムプロンプトプレフィックス
// Workflow Rules の直前に挿入し、プロバイダー固有ノートのみ含む
// 共通ルールは system.go に一元化されている
var providerPrefixes = map[string]string{
	"gemini": "## Provider Notes\n" +
		"### Gemini-specific\n" +
		"- Tool calls must be raw JSON, not markdown code blocks\n" +
		"- Do not claim a command ran unless you actually called bash\n" +
		"- Respond in the same language as the user's message\n" +
		"- Call tools directly for action requests instead of previewing hypothetical edits\n" +
		"- Edit the original file directly; do not create derivative temp files\n" +
		"- Make str_replace old_str unique by including enough surrounding context\n",
	"deepseek": "## Provider Notes\n" +
		"### DeepSeek-specific\n" +
		"- When function calling is enabled, use tool calls for file operations instead of plain-text descriptions\n" +
		"- Fix errors completely; do not leave TODO-style excuses such as \"for brevity\" or \"due to time constraints\"\n" +
		"- After str_replace, remove any unused imports introduced by the edit before moving on\n" +
		"- After read_file, either edit or gather more context; do not echo file contents back to the user\n" +
		"- You are already in the project root directory; do not prefix commands with `cd /path &&`\n",
	"groq": "## Provider Notes\n" +
		"### Groq-specific\n" +
		"- Tool calls must be raw JSON, not markdown code blocks or XML wrappers\n" +
		"- Respond in the same language as the user's message\n",
	"openai": "## Provider Notes\n" +
		"### OpenAI-specific\n" +
		"- When the user instructs execution, proceed without asking for confirmation; choose the safer reasonable option and move forward\n" +
		"- Fix compile errors at the root cause; do not delete existing logic just to make the build pass\n" +
		"- When recovering from errors, prefer targeted str_replace edits over whole-file rewrites\n" +
		"- For str_replace with mixed Japanese, JSON, or backticks, split the change into smaller edits to avoid byte corruption\n" +
		"- When removing functionality, clean up dead imports, dead code, and orphaned dependencies in the same task\n",
	// "claude" と "anthropic" は同一プロバイダー（anthropic はエイリアス）
	// GetProviderPrefix() 内でエイリアス正規化するため "claude" のみ定義
	"claude": "## Provider Notes\n" +
		"### Claude-specific\n" +
		"- Use search_code for file search instead of bash grep/rg/find\n" +
		"- Use read_file for file contents instead of cat/head/tail/sed\n" +
		"- When operations are independent, use parallel tool calls in a single response\n",
	"openrouter": "",
	"ollama":     "",
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

// BuildProviderSystemPromptWithConfig は明示指定した設定を使ってプロバイダー別ノートを挿入する。
// シグネチャは呼び出し元との互換性のため model, cfg を維持する。
func BuildProviderSystemPromptWithConfig(base, providerName, model string, cfg *config.Config) string {
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
