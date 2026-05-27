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

// GetProviderPrefix はプロバイダー名に応じたプレフィックスを返す
// 未登録プロバイダーは空文字を返す
func GetProviderPrefix(provider string) string {
	return getProviderPrefixForModel(provider, "", nil)
}

func getProviderPrefixForModel(provider string, model string, cfg *config.Config) string {
	return providerPrefixes[resolveProviderPromptKey(provider, model, cfg)]
}

func resolveProviderPromptKey(provider, model string, cfg *config.Config) string {
	catalogModel := model
	if cfg != nil {
		if strings.TrimSpace(model) == "" {
			model = cfg.GetEffectiveModelForProvider(provider)
		}
		catalogModel = cfg.ModelCatalogName(provider, model)
	}
	return llmcatalog.ResolveProviderRoute(provider, model, catalogModel).PromptFamily
}

const (
	workflowRulesHeader    = "\n## Workflow Rules\n"
	providerNotesEndMarker = "<!-- PROVIDER_NOTES_END -->"
)

var providerNotesBlockRe = regexp.MustCompile(`(?s)\n?<!-- PROVIDER_NOTES_START:[^>]+ -->.*?<!-- PROVIDER_NOTES_END -->\n?`)
var legacyGeneratedProviderNotesSectionSet = buildLegacyGeneratedProviderNotesSectionSet()

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
	base = providerNotesBlockRe.ReplaceAllString(base, "")
	return stripLegacyProviderNotesSections(base)
}

func stripLegacyProviderNotesSections(base string) string {
	if !strings.Contains(base, "## Provider Notes") {
		return base
	}

	lines := strings.Split(base, "\n")
	out := make([]string, 0, len(lines))

	for i := 0; i < len(lines); {
		if strings.TrimSpace(lines[i]) == "## Provider Notes" {
			start := i
			i++
			for i < len(lines) {
				trimmed := strings.TrimSpace(lines[i])
				if strings.HasPrefix(trimmed, "## ") {
					break
				}
				i++
			}
			section := strings.Join(lines[start:i], "\n")
			if !isLegacyGeneratedProviderNotesSection(section) {
				out = append(out, lines[start:i]...)
			}
			continue
		}
		out = append(out, lines[i])
		i++
	}

	return strings.Join(out, "\n")
}

func buildLegacyGeneratedProviderNotesSectionSet() map[string]struct{} {
	sections := make(map[string]struct{}, len(providerPrefixes))
	for _, prefix := range providerPrefixes {
		normalized := normalizeProviderNotesSection(prefix)
		if normalized == "" {
			continue
		}
		sections[normalized] = struct{}{}
	}
	return sections
}

func isLegacyGeneratedProviderNotesSection(section string) bool {
	normalized := normalizeProviderNotesSection(section)
	if normalized == "" {
		return false
	}
	_, ok := legacyGeneratedProviderNotesSectionSet[normalized]
	return ok
}

func normalizeProviderNotesSection(section string) string {
	section = strings.ReplaceAll(section, "\r\n", "\n")
	section = strings.TrimSpace(section)
	if section == "" {
		return ""
	}

	lines := strings.Split(section, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	return strings.Join(lines, "\n")
}
