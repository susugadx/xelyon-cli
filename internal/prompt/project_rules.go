package prompt

import (
	"fmt"
	"strings"
)

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

// InjectProjectRules は SystemPrompt の Workflow Rules 内にプロジェクトルールを埋め込む。
// Rule #10 (Verification Protocol) の直後に挿入する。
// rulesBlock が空の場合は systemPrompt をそのまま返す。
func InjectProjectRules(systemPrompt, rulesBlock string) string {
	if rulesBlock == "" {
		return systemPrompt
	}

	// Rule #10 の末尾（"A task is NOT complete until verification passes"）を探す
	marker := "A task is NOT complete until verification passes"
	idx := strings.Index(systemPrompt, marker)
	if idx < 0 {
		// マーカーが見つからない場合は Workflow Rules の末尾に追加
		return systemPrompt + rulesBlock
	}

	insertPos := idx + len(marker)
	return systemPrompt[:insertPos] + rulesBlock + systemPrompt[insertPos:]
}
