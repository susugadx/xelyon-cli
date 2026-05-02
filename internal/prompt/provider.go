package prompt

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
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
	"azure":     "openai", // Azure OpenAI は OpenAI Responses 系
}

// GetProviderPrefix はプロバイダー名に応じたプレフィックスを返す
// 未登録プロバイダーは空文字を返す
func GetProviderPrefix(provider string) string {
	return getProviderPrefixForModel(provider, "", nil)
}

func getProviderPrefixForModel(provider string, model string, cfg *config.Config) string {
	return providerPrefixes[resolveProviderPromptKey(provider, model, cfg)]
}

func resolveProviderPromptKey(provider, model string, cfg *config.Config) string {
	name := config.NormalizeProviderName(provider)
	if name != "bedrock" {
		name = config.CanonicalProviderName(name)
	}
	if canonical, ok := providerAliases[name]; ok {
		name = canonical
	}
	if name == "bedrock" {
		if bedrockPromptFamily(model, cfg) == llmcatalog.BedrockModelFamilyClaude {
			name = "claude"
		}
	}
	return name
}

func bedrockPromptFamily(model string, cfg *config.Config) llmcatalog.BedrockModelFamily {
	catalogModel := model
	if cfg != nil {
		if strings.TrimSpace(model) == "" {
			model = cfg.GetEffectiveModelForProvider("bedrock")
		}
		catalogModel = cfg.ModelCatalogName("bedrock", model)
	}
	return llmcatalog.BedrockModelFamilyFor(model, catalogModel)
}

const (
	workflowRulesHeader    = "\n## Workflow Rules\n"
	providerNotesEndMarker = "<!-- PROVIDER_NOTES_END -->"
)

var providerNotesBlockRe = regexp.MustCompile(`(?s)\n?<!-- PROVIDER_NOTES_START:[^>]+ -->.*?<!-- PROVIDER_NOTES_END -->\n?`)

// BuildProviderSystemPromptWithConfig は明示指定した設定を使ってプロバイダー別ノートを挿入する。
// シグネチャは呼び出し元との互換性のため model, cfg を維持する。
func BuildProviderSystemPromptWithConfig(base, providerName, model string, cfg *config.Config) string {
	base = strings.TrimRight(stripProviderNotesBlocks(base), "\n")
	prefix := strings.TrimSpace(getProviderPrefixForModel(providerName, model, cfg))
	if prefix == "" {
		return base
	}
	key := resolveProviderPromptKey(providerName, model, cfg)
	if key == "" {
		key = "unknown"
	}
	providerBlock := buildProviderNotesBlock(key, prefix)

	idx := strings.Index(base, workflowRulesHeader)
	if idx < 0 {
		if strings.TrimSpace(base) == "" {
			return providerBlock
		}
		return providerBlock + "\n\n" + base
	}
	return base[:idx] + "\n\n" + providerBlock + "\n" + base[idx:]
}

func buildProviderNotesBlock(providerKey, prefix string) string {
	return fmt.Sprintf("<!-- PROVIDER_NOTES_START:%s -->\n%s\n%s", providerKey, prefix, providerNotesEndMarker)
}

func stripProviderNotesBlocks(base string) string {
	return providerNotesBlockRe.ReplaceAllString(base, "")
}
