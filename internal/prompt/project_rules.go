package prompt

import (
	"fmt"
	"regexp"
	"strings"
)

var projectConfigBlockRe = regexp.MustCompile(`(?s)\n?<!-- PROJECT_CONFIG_START -->.*?<!-- PROJECT_CONFIG_END -->\n?`)

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

	var contextParts []string
	for _, context := range contexts {
		context = strings.TrimSpace(context)
		if context == "" {
			continue
		}
		contextParts = append(contextParts, context)
	}

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
}

// BuildProjectInstructionBlock は xelyon.yaml mandatory rules と imported guidance を
// 優先順位説明付きで 1 ブロックに組み立てる。
func BuildProjectInstructionBlock(input ProjectInstructionBlockInput) string {
	rulesBlock := BuildRulesBlockFromList(input.MandatoryRules)

	var contextParts []string
	for _, context := range input.ProjectContexts {
		context = strings.TrimSpace(context)
		if context == "" {
			continue
		}
		contextParts = append(contextParts, context)
	}

	hasProjectGuidance := len(input.ProjectGuidance) > 0
	hasGlobalGuidance := len(input.GlobalGuidance) > 0

	if rulesBlock == "" && len(contextParts) == 0 && !hasProjectGuidance && !hasGlobalGuidance {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n\n<!-- PROJECT_CONFIG_START -->\n")
	b.WriteString("\n## Project Instruction Precedence\n\n")
	b.WriteString("- XELYON system/tool/safety rules are highest priority.\n")
	b.WriteString("- The current user request is higher priority than project guidance unless it conflicts with XELYON safety, tool, investigation, or verification invariants.\n")
	b.WriteString("- xelyon.yaml rules are mandatory project policy.\n")
	b.WriteString("- Imported AGENTS.md / CLAUDE.md files are project guidance when no xelyon.yaml exists.\n")
	b.WriteString("- Imported AGENTS.md / CLAUDE.md files are advisory guidance when xelyon.yaml exists and project.mode=always is enabled.\n")
	b.WriteString("- Global guidance is personal preference and lower priority than repo-local guidance.\n")

	if rulesBlock != "" {
		b.WriteString(rulesBlock)
	}
	if len(contextParts) > 0 {
		b.WriteString("\n\n## Project Context\n")
		b.WriteString(strings.Join(contextParts, "\n\n"))
	}
	if hasProjectGuidance {
		b.WriteString("\n\n## Imported Project Guidance\n\n")
		if input.HasProjectConfig {
			b.WriteString("xelyon.yaml was found for this workspace.\n")
			b.WriteString("The following imported files are treated as advisory guidance. Use them when relevant, but do not override xelyon.yaml mandatory rules, XELYON internal rules, or the current user request.\n")
		} else {
			b.WriteString("No xelyon.yaml was found for this workspace.\n")
			b.WriteString("The following files are treated as authoritative project guidance for this workspace.\n")
			b.WriteString("Follow them when they are clear and relevant, but do not override XELYON internal rules or the current user request.\n")
		}
		for _, entry := range input.ProjectGuidance {
			content := strings.TrimSpace(entry.Content)
			if content == "" {
				continue
			}
			b.WriteString("\n\n### ")
			b.WriteString(strings.TrimSpace(entry.Label))
			b.WriteString("\n")
			b.WriteString(content)
		}
	}
	if hasGlobalGuidance {
		b.WriteString("\n\n## Enabled Global Guidance\n\n")
		b.WriteString("Global guidance is advisory personal preference.\n")
		for _, entry := range input.GlobalGuidance {
			content := strings.TrimSpace(entry.Content)
			if content == "" {
				continue
			}
			b.WriteString("\n\n### ")
			b.WriteString(strings.TrimSpace(entry.Label))
			b.WriteString("\n")
			b.WriteString(content)
		}
	}
	b.WriteString("\n<!-- PROJECT_CONFIG_END -->")
	return b.String()
}

// InjectProjectConfigBlock は SystemPrompt の Workflow Rules 内に project config ブロックを埋め込む。
// Rule #10 (Verification Protocol) の直後に挿入する。
// projectBlock が空の場合は systemPrompt をそのまま返す。
func InjectProjectConfigBlock(systemPrompt, projectBlock string) string {
	if projectBlock == "" {
		return systemPrompt
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
