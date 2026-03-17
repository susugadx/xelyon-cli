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
