package prompt

import (
	"fmt"
	"regexp"
	"strings"
)

var projectConfigBlockRe = regexp.MustCompile(`(?s)\n?<!-- PROJECT_CONFIG_START -->.*?<!-- PROJECT_CONFIG_END -->\n?`)
var verificationRuleBlockRe = regexp.MustCompile(`(?s)(### 10\. Verification Protocol \(MANDATORY\).*?)(\n### [0-9]+\.\s|\z)`)

// BuildRulesBlockFromList は []string のルールリストから mandatory rules ブロックを構築する。
// xelyon.yaml の rules フィールド用。空リストの場合は空文字を返す。
func BuildRulesBlockFromList(rules []string) string {
	if len(rules) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n\n=== PROJECT-SPECIFIC RULES (MANDATORY) ===\n")
	for i, rule := range rules {
		fmt.Fprintf(&b, "%d. %s\n", i+1, rule)
	}
	b.WriteString("Violating ANY of these rules is a critical failure.")

	return b.String()
}

// BuildProjectConfigBlock は project rules/context を system prompt 用の1ブロックにまとめる。
func BuildProjectConfigBlock(rules []string, contexts []string) string {
	rulesBlock := BuildRulesBlockFromList(rules)
	contextParts := normalizeProjectContexts(contexts)

	if rulesBlock == "" && len(contextParts) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n\n<!-- PROJECT_CONFIG_START -->")
	if rulesBlock != "" {
		b.WriteString(rulesBlock)
	}
	if len(contextParts) > 0 {
		b.WriteString("\n\n## Project Context:\n")
		b.WriteString(strings.Join(contextParts, "\n\n"))
	}
	b.WriteString("\n<!-- PROJECT_CONFIG_END -->")
	return b.String()
}

// ProjectInstructionEntry は imported guidance の注入用 DTO。
type ProjectInstructionEntry struct {
	Label    string
	Content  string
	Strength string // project_guidance / advisory
}

// ProjectInstructionBlockInput は project instruction block 生成入力 DTO。
type ProjectInstructionBlockInput struct {
	HasProjectConfig bool
	MandatoryRules   []string
	ProjectContexts  []string
	ProjectGuidance  []ProjectInstructionEntry
	GlobalGuidance   []ProjectInstructionEntry
	Warnings         []string
}

// BuildProjectInstructionBlock は xelyon.yaml mandatory rules と imported guidance を
// 優先順位説明付きで 1 ブロックに組み立てる。
func BuildProjectInstructionBlock(input ProjectInstructionBlockInput) string {
	rulesBlock := BuildRulesBlockFromList(input.MandatoryRules)
	contextParts := normalizeProjectContexts(input.ProjectContexts)
	warnings := normalizeProjectWarnings(input.Warnings)

	hasProjectGuidance := len(input.ProjectGuidance) > 0
	hasGlobalGuidance := len(input.GlobalGuidance) > 0
	hasWarnings := len(warnings) > 0

	if rulesBlock == "" && len(contextParts) == 0 && !hasProjectGuidance && !hasGlobalGuidance && !hasWarnings {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n\n<!-- PROJECT_CONFIG_START -->\n")
	b.WriteString("\n## Project Instruction Precedence\n\n")
	b.WriteString(buildProjectInstructionPrecedenceBlock())

	if rulesBlock != "" {
		b.WriteString(rulesBlock)
	}
	if len(contextParts) > 0 {
		b.WriteString("\n\n## Project Context\n")
		b.WriteString(strings.Join(contextParts, "\n\n"))
	}
	if hasProjectGuidance {
		sectionText := projectGuidanceWithoutConfigText
		if input.HasProjectConfig {
			sectionText = projectGuidanceWithConfigText
		}
		appendGuidanceSection(&b, "## Imported Project Guidance", sectionText, input.ProjectGuidance)
	}
	if hasGlobalGuidance {
		appendGuidanceSection(&b, "## Enabled Global Guidance", globalGuidanceText, input.GlobalGuidance)
	}
	if hasWarnings {
		b.WriteString("\n\n## Guidance Load Notes\n")
		for _, warning := range warnings {
			b.WriteString("\n- ")
			b.WriteString(warning)
		}
	}
	b.WriteString("\n<!-- PROJECT_CONFIG_END -->")
	return b.String()
}

func appendGuidanceSection(b *strings.Builder, heading string, intro string, entries []ProjectInstructionEntry) {
	if b == nil || strings.TrimSpace(heading) == "" {
		return
	}
	b.WriteString("\n\n")
	b.WriteString(heading)
	b.WriteString("\n\n")
	b.WriteString(intro)
	for _, entry := range entries {
		content := strings.TrimSpace(entry.Content)
		if content == "" {
			continue
		}
		b.WriteString("\n\n### ")
		b.WriteString(guidanceHeadingLabel(entry))
		b.WriteString("\n")
		b.WriteString(content)
	}
}

// InjectProjectConfigBlock は SystemPrompt の Workflow Rules 内に project config ブロックを埋め込む。
// Rule #10 (Verification Protocol) の直後に挿入する。
// projectBlock が空の場合は systemPrompt をそのまま返す。
func InjectProjectConfigBlock(systemPrompt, projectBlock string) string {
	if projectBlock == "" {
		return systemPrompt
	}

	// 優先: Rule #10 ブロック境界を正規表現で見つけて、その直後に挿入する。
	if match := verificationRuleBlockRe.FindStringSubmatchIndex(systemPrompt); len(match) >= 4 {
		insertPos := match[3]
		return systemPrompt[:insertPos] + projectBlock + systemPrompt[insertPos:]
	}

	// Rule #10 の末尾（"A task is NOT complete until verification passes"）を探す
	marker := "A task is NOT complete until verification passes"
	idx := strings.Index(systemPrompt, marker)
	if idx < 0 {
		// マーカーが見つからない場合は Workflow Rules の末尾に追加
		return systemPrompt + projectBlock
	}

	insertPos := idx + len(marker)
	return systemPrompt[:insertPos] + projectBlock + systemPrompt[insertPos:]
}

// InjectProjectRules は後方互換のため rules のみを注入する。
func InjectProjectRules(systemPrompt, rulesBlock string) string {
	if rulesBlock == "" {
		return systemPrompt
	}
	return InjectProjectConfigBlock(systemPrompt, "\n\n<!-- PROJECT_CONFIG_START -->"+rulesBlock+"\n<!-- PROJECT_CONFIG_END -->")
}

// StripProjectConfigSections は以前注入した project config ブロックを除去する。
func StripProjectConfigSections(systemPrompt string) string {
	return projectConfigBlockRe.ReplaceAllString(systemPrompt, "")
}

// ExtractProjectConfigBlock は system prompt から project config ブロックを抽出する。
// 見つからなければ空文字を返す。
func ExtractProjectConfigBlock(systemPrompt string) string {
	match := projectConfigBlockRe.FindString(systemPrompt)
	return strings.TrimSpace(match)
}

func normalizeProjectContexts(contexts []string) []string {
	if len(contexts) == 0 {
		return nil
	}
	result := make([]string, 0, len(contexts))
	for _, context := range contexts {
		context = strings.TrimSpace(context)
		if context == "" {
			continue
		}
		result = append(result, context)
	}
	return result
}

func normalizeProjectWarnings(warnings []string) []string {
	if len(warnings) == 0 {
		return nil
	}
	result := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		warning = strings.TrimSpace(warning)
		if warning == "" {
			continue
		}
		result = append(result, warning)
	}
	return result
}
