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
		"- Edit the original file directly; do not create derivative temp files\n" +
		"- Before each tool call, briefly state what you are about to do and why in one sentence (e.g. 'Reading auth.go to check the validation logic')\n",
	"deepseek": "## Provider Notes\n" +
		"### DeepSeek-specific\n" +
		"- When function calling is enabled, use tool calls for file operations instead of plain-text descriptions\n" +
		"- Fix errors completely; do not leave TODO-style excuses such as \"for brevity\" or \"due to time constraints\"\n" +
		"- After str_replace, remove any unused imports introduced by the edit before moving on\n" +
		"- After read_file, either edit or gather more context; do not echo file contents back to the user\n" +
		"- You are already in the project root directory; do not prefix commands with `cd /path &&`\n",
	"groq": "## Provider Notes\n" +
		"### Groq-specific\n" +
		"- Tool calls must be raw JSON, not markdown code blocks or XML wrappers\n",
	"openai": "## Provider Notes\n" +
		"### OpenAI-specific\n" +
		"- For str_replace with mixed Japanese, JSON, or backticks, split the change into smaller edits to avoid byte corruption\n" +
		"- Before each tool call, briefly state what you are about to do and why in one sentence (e.g. 'Reading auth.go to check the validation logic')\n",
	"claude": "## Provider Notes\n" +
		"### Claude-specific\n" +
		"- Always use dedicated tools (read_file, search_code, list_dir) instead of bash equivalents; tools provide caching, range tracking, and structured output\n" +
		"- You are already in the project root directory; do not prefix commands with `cd /path &&`\n",
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
