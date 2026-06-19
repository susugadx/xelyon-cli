package prompt

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
)

// providerPrefixes はプロバイダー別のシステムプロンプトプレフィックス
// Workflow Rules の直前に挿入し、プロバイダー固有ノートのみ含む
// 共通ルールは system.go に一元化されている
var providerPrefixes = map[string]string{
	"gemini": "## Provider Notes\n" +
		"### Gemini-specific\n" +
		"- Emit native tool calls directly; do not serialize tool calls inside markdown code blocks\n",
	"deepseek": "## Provider Notes\n" +
		"### DeepSeek-specific\n" +
		"- When function calling is enabled, use native tool calls instead of plain-text tool-call descriptions\n",
	"groq": "## Provider Notes\n" +
		"### Groq-specific\n" +
		"- Emit native tool calls directly; do not serialize tool calls inside markdown code blocks or XML wrappers\n",
	"openai": "",
	"claude": "## Provider Notes\n" +
		"### Claude-specific\n" +
		"- Use dedicated repository tools for code investigation and edits when they are available\n",
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
var legacyGeneratedProviderNotesSectionHashes = map[string]struct{}{
	"13a4dfa9d5cda6ad451291234db2b3b200166029559f0b299b6851d070f32576": {},
	"7aaca9f76ba592db058be4ab9a2c679a09a871ca607851d6882eaa79ca012f04": {},
	"967efc64584fe70b3bac24b05851cde1b0a009e6d7fe4456afed7c39e5515f66": {},
	"7e7c30daf93caed4c783c5b22f1d3be9c06dc919a528c087efe2a939c55a0357": {},
	"aded122e0fdd545116d5936cac04494becbba6de388f8ab373818c84d047ba19": {},
}

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
	if ok {
		return true
	}
	_, ok = legacyGeneratedProviderNotesSectionHashes[hashProviderNotesSection(normalized)]
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

func hashProviderNotesSection(section string) string {
	sum := sha256.Sum256([]byte(section))
	return fmt.Sprintf("%x", sum)
}
