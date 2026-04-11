package prompt

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	promptfragments "github.com/susugadx/xelyon-cli/internal/prompt/fragments"
)

// providerPrefixes はプロバイダー別のシステムプロンプトプレフィックス
// Workflow Rules の直前に挿入し、プロバイダー固有ノートのみ含む
// 共通ルールは system.go に一元化されている
var providerPrefixes = map[string]string{
	"gemini": "## Provider Notes\n" +
		"### Gemini-specific\n" +
		"- Tool calls must be raw JSON, not markdown code blocks\n" +
		"- Edit the original file directly; do not create derivative temp files\n" +
		"- " + promptfragments.NoBashSubstituteSentence() + "\n",
	"deepseek": "## Provider Notes\n" +
		"### DeepSeek-specific\n" +
		"- When function calling is enabled, use tool calls for file operations instead of plain-text descriptions\n" +
		"- Fix errors completely; do not leave TODO-style excuses such as \"for brevity\" or \"due to time constraints\"\n" +
		"- After a file edit changes imports, remove any newly unused imports before moving on\n" +
		"- After an investigation read, either edit or gather more context; do not echo file contents back to the user\n" +
		"- You are already in the project root directory; do not prefix commands with `cd /path &&`\n",
	"groq": "## Provider Notes\n" +
		"### Groq-specific\n" +
		"- Tool calls must be raw JSON, not markdown code blocks or XML wrappers\n",
	"openai": "## Provider Notes\n" +
		"### OpenAI-specific\n" +
		"- For edits containing mixed Japanese, JSON, or backticks, split the change into smaller precise chunks to avoid byte corruption\n",
	"claude": "## Provider Notes\n" +
		"### Claude-specific\n" +
		"- " + promptfragments.DedicatedToolUsageSentence() + "\n" +
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
	name := config.NormalizeProviderName(provider)
	if name == "bedrock" {
		name = "claude"
	} else {
		name = config.CanonicalProviderName(name)
	}
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
