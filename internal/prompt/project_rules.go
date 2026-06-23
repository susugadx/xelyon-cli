package prompt

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

var projectConfigBlockRe = regexp.MustCompile(`(?s)\n?<!-- PROJECT_CONFIG_START -->.*?<!-- PROJECT_CONFIG_END -->\n?`)
var trailingProjectConfigBlockRe = regexp.MustCompile(`(?s)\n?<!-- PROJECT_CONFIG_START -->.*?<!-- PROJECT_CONFIG_END -->\s*$`)

// BuildRulesBlockFromList は legacy rules ブロックを構築する互換 helper。
// 新しい project instruction 注入経路では呼ばない。空リストの場合は空文字を返す。
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

// BuildProjectConfigBlock は legacy project rules/context を marker 付きブロックにまとめる互換 helper。
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
	Scope    string
	Source   string
	Content  string
	Strength string // project_guidance / advisory
}

// ProjectInstructionBlockInput は project instruction block 生成入力 DTO。
type ProjectInstructionBlockInput struct {
	ProjectGuidance []ProjectInstructionEntry
	GlobalGuidance  []ProjectInstructionEntry
	Warnings        []string
}

// BuildProjectInstructionBlock は imported guidance と load warning を
// 優先順位説明付きで 1 ブロックに組み立てる。
func BuildProjectInstructionBlock(input ProjectInstructionBlockInput) string {
	section, ok := BuildProjectInstructionSection(input)
	if !ok {
		return ""
	}
	return "\n\n" + section.Content()
}

// BuildProjectInstructionSection は project instructions を repo_instruction section として構築する。
func BuildProjectInstructionSection(input ProjectInstructionBlockInput) (PromptSection, bool) {
	warnings := normalizeProjectWarnings(input.Warnings)
	hasProjectGuidance := len(input.ProjectGuidance) > 0
	hasGlobalGuidance := len(input.GlobalGuidance) > 0
	hasWarnings := len(warnings) > 0

	if !hasProjectGuidance && !hasGlobalGuidance && !hasWarnings {
		return PromptSection{}, false
	}

	var b strings.Builder
	b.WriteString("\n\n<!-- PROJECT_CONFIG_START -->\n")
	b.WriteString("\n## Project Instruction Precedence\n\n")
	b.WriteString(buildProjectInstructionPrecedenceBlock())

	if hasProjectGuidance {
		appendRepositoryGuidanceSection(&b, "## Imported Project Guidance", projectGuidanceText, input.ProjectGuidance)
	}
	if hasGlobalGuidance {
		appendGuidanceSection(&b, "## Enabled Global Guidance", globalGuidanceText, input.GlobalGuidance)
	}
	if hasWarnings {
		b.WriteString("\n\n## Guidance Load Notes\n")
		for _, warning := range warnings {
			b.WriteString("\n- ")
			b.WriteString(neutralizeProjectInstructionBlockDelimiters(warning))
		}
	}
	b.WriteString("\n<!-- PROJECT_CONFIG_END -->")
	return StaticText("xelyon.project_instructions", AuthorityRepoInstruction, b.String()), true
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
		b.WriteString(neutralizeProjectInstructionBlockDelimiters(guidanceHeadingLabel(entry)))
		b.WriteString("\n")
		b.WriteString(neutralizeProjectInstructionBlockDelimiters(content))
	}
}

func appendRepositoryGuidanceSection(b *strings.Builder, heading string, intro string, entries []ProjectInstructionEntry) {
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
		b.WriteString("\n\n<repository_instructions scope=\"")
		b.WriteString(escapeInstructionAttribute(instructionEntryScope(entry)))
		b.WriteString("\" source=\"")
		b.WriteString(escapeInstructionAttribute(instructionEntrySource(entry)))
		b.WriteString("\">\n")
		b.WriteString(neutralizeProjectInstructionBlockDelimiters(content))
		b.WriteString("\n</repository_instructions>")
	}
}

func neutralizeProjectInstructionBlockDelimiters(content string) string {
	return strings.NewReplacer(
		"</repository_instructions>", "&lt;/repository_instructions&gt;",
		"<repository_instructions", "&lt;repository_instructions",
		"<!-- PROJECT_CONFIG_START -->", "&lt;!-- PROJECT_CONFIG_START --&gt;",
		"<!-- PROJECT_CONFIG_END -->", "&lt;!-- PROJECT_CONFIG_END --&gt;",
	).Replace(content)
}

func instructionEntryScope(entry ProjectInstructionEntry) string {
	scope := strings.TrimSpace(entry.Scope)
	if scope == "" {
		return "."
	}
	return scope
}

func instructionEntrySource(entry ProjectInstructionEntry) string {
	source := strings.TrimSpace(entry.Source)
	if source != "" {
		return source
	}
	return strings.TrimSpace(entry.Label)
}

func escapeInstructionAttribute(value string) string {
	return html.EscapeString(strings.TrimSpace(value))
}

// InjectProjectConfigBlock は SystemPrompt の marker 位置に project config ブロックを埋め込む。
// marker のない custom prompt では末尾へ追加する。
// projectBlock が空の場合は systemPrompt をそのまま返す。
func InjectProjectConfigBlock(systemPrompt, projectBlock string) string {
	if projectBlock == "" {
		return systemPrompt
	}

	if idx := strings.Index(systemPrompt, projectConfigAnchorMarker); idx >= 0 {
		insertPos := idx + len(projectConfigAnchorMarker)
		return systemPrompt[:insertPos] + projectBlock + systemPrompt[insertPos:]
	}

	return systemPrompt + projectBlock
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
	stripped := projectConfigBlockRe.ReplaceAllString(systemPrompt, "")
	if stripped != systemPrompt && trailingProjectConfigBlockRe.MatchString(systemPrompt) {
		return strings.TrimRight(stripped, "\n")
	}
	return stripped
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
