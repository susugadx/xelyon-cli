package prompt

import (
	"fmt"
	"strings"
)

// ruleSections はプロジェクトルールとして抽出するセクション名のリスト
var ruleSections = []string{
	"開発ルール",
	"Verification Commands",
}

// ExtractSection は Markdown テキストから指定セクション（## セクション名）の内容を抽出する。
// セクションは「## セクション名」で始まり、次の「## 」または EOF で終わる。
// セクションが見つからない場合は空文字を返す。
func ExtractSection(content, sectionName string) string {
	header := "## " + sectionName
	idx := strings.Index(content, header)
	if idx < 0 {
		return ""
	}

	// ヘッダー行の直後から開始
	start := idx + len(header)
	// ヘッダー行の残り（改行まで）をスキップ
	if nlIdx := strings.Index(content[start:], "\n"); nlIdx >= 0 {
		start += nlIdx + 1
	} else {
		return ""
	}

	// 次の "## " を探す
	rest := content[start:]
	if nextIdx := strings.Index(rest, "\n## "); nextIdx >= 0 {
		return strings.TrimSpace(rest[:nextIdx])
	}
	return strings.TrimSpace(rest)
}

// BuildProjectRulesBlock は XELYON.md の内容からプロジェクトルールブロックを構築する。
// 「開発ルール」と「Verification Commands」セクションを抽出し、
// SystemPrompt の Workflow Rules 内に強制挿入するためのテキストを返す。
// いずれのセクションも見つからない場合は空文字を返す。
func BuildProjectRulesBlock(xelyonMDContent string) string {
	if xelyonMDContent == "" {
		return ""
	}

	devRules := ExtractSection(xelyonMDContent, "開発ルール")
	verifyCmd := ExtractSection(xelyonMDContent, "Verification Commands")

	if devRules == "" && verifyCmd == "" {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n\n=== PROJECT-SPECIFIC RULES (MANDATORY) ===\n")
	if devRules != "" {
		b.WriteString(devRules)
		b.WriteString("\n")
	}
	if verifyCmd != "" {
		b.WriteString("\n=== VERIFICATION COMMANDS (MUST run after changes) ===\n")
		b.WriteString(verifyCmd)
		b.WriteString("\n")
	}
	b.WriteString("Violating ANY of these rules is a critical failure.")

	return b.String()
}

// StripRuleSections は XELYON.md の内容からルール系セクションを除去した内容を返す。
// 末尾の Project Context 追加時にルールが二重にならないようにするため。
func StripRuleSections(xelyonMDContent string) string {
	result := xelyonMDContent
	for _, name := range ruleSections {
		result = stripSection(result, name)
	}
	return strings.TrimSpace(result)
}

// stripSection は Markdown テキストから指定セクション全体（ヘッダー含む）を除去する。
func stripSection(content, sectionName string) string {
	header := "## " + sectionName
	idx := strings.Index(content, header)
	if idx < 0 {
		return content
	}

	// セクションの終わり（次の "## " または EOF）
	rest := content[idx:]
	// ヘッダー行をスキップ
	afterHeader := strings.Index(rest, "\n")
	if afterHeader < 0 {
		// ヘッダーだけで終わり
		return strings.TrimSpace(content[:idx])
	}

	bodyStart := rest[afterHeader+1:]
	if nextIdx := strings.Index(bodyStart, "\n## "); nextIdx >= 0 {
		// 次のセクション開始の \n を含む位置まで除去
		endIdx := idx + afterHeader + 1 + nextIdx + 1
		return content[:idx] + content[endIdx:]
	}
	// EOF まで
	return strings.TrimSpace(content[:idx])
}

// BuildRulesBlockFromList は []string のルールリストから mandatory rules ブロックを構築する。
// xelyon.yaml の rules フィールド用。空リストの場合は空文字を返す。
func BuildRulesBlockFromList(rules []string) string {
	if len(rules) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n\n=== PROJECT-SPECIFIC RULES (MANDATORY) ===\n")
	for i, rule := range rules {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, rule))
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
