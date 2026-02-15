package prompt

import (
	"strings"
	"testing"
)

const testXelyonMD = `# XELYON CLI

> AI 用コンテキスト

## 概要

Go 製 AI コーディングアシスタント CLI。

## 技術スタック

- Go 1.24+
- Cobra（CLI）

## 開発ルール

- コミット前は ` + "`make ci-check`" + ` 必須
- 新機能・バグ修正時はテストも追加
- エラーハンドリング必須

## Verification Commands

コード変更後に必ず実行するコマンド。

` + "```bash\nmake ci-check\n```" + `

フォールバック:

` + "```bash\ngo fmt ./... && go build ./... && go test ./...\n```" + `

## 参照先

詳細は README.md を参照
`

func TestExtractSection_DevRules(t *testing.T) {
	result := ExtractSection(testXelyonMD, "開発ルール")
	if result == "" {
		t.Fatal("expected non-empty result for 開発ルール")
	}
	if !strings.Contains(result, "make ci-check") {
		t.Errorf("should contain 'make ci-check', got: %s", result)
	}
	if !strings.Contains(result, "エラーハンドリング必須") {
		t.Errorf("should contain last rule, got: %s", result)
	}
	// 次のセクションの内容は含まない
	if strings.Contains(result, "Verification Commands") {
		t.Errorf("should NOT contain next section header, got: %s", result)
	}
}

func TestExtractSection_VerificationCommands(t *testing.T) {
	result := ExtractSection(testXelyonMD, "Verification Commands")
	if result == "" {
		t.Fatal("expected non-empty result for Verification Commands")
	}
	if !strings.Contains(result, "make ci-check") {
		t.Errorf("should contain verification command, got: %s", result)
	}
	if !strings.Contains(result, "フォールバック") {
		t.Errorf("should contain fallback section, got: %s", result)
	}
	// 次のセクションの内容は含まない
	if strings.Contains(result, "README.md を参照") {
		t.Errorf("should NOT contain next section content, got: %s", result)
	}
}

func TestExtractSection_NotFound(t *testing.T) {
	result := ExtractSection(testXelyonMD, "存在しないセクション")
	if result != "" {
		t.Errorf("expected empty string for missing section, got: %s", result)
	}
}

func TestExtractSection_EmptyContent(t *testing.T) {
	result := ExtractSection("", "開発ルール")
	if result != "" {
		t.Errorf("expected empty string for empty content, got: %s", result)
	}
}

func TestExtractSection_LastSection(t *testing.T) {
	// 最後のセクション（EOF で終わる）
	result := ExtractSection(testXelyonMD, "参照先")
	if result == "" {
		t.Fatal("expected non-empty result for last section")
	}
	if !strings.Contains(result, "README.md") {
		t.Errorf("should contain README.md reference, got: %s", result)
	}
}

func TestBuildProjectRulesBlock_Normal(t *testing.T) {
	block := BuildProjectRulesBlock(testXelyonMD)
	if block == "" {
		t.Fatal("expected non-empty rules block")
	}
	if !strings.Contains(block, "PROJECT-SPECIFIC RULES (MANDATORY)") {
		t.Errorf("should contain MANDATORY header, got: %s", block)
	}
	if !strings.Contains(block, "VERIFICATION COMMANDS (MUST run after changes)") {
		t.Errorf("should contain VERIFICATION header, got: %s", block)
	}
	if !strings.Contains(block, "Violating ANY of these rules is a critical failure.") {
		t.Errorf("should contain violation warning, got: %s", block)
	}
	if !strings.Contains(block, "make ci-check") {
		t.Errorf("should contain actual rule content, got: %s", block)
	}
}

func TestBuildProjectRulesBlock_Empty(t *testing.T) {
	block := BuildProjectRulesBlock("")
	if block != "" {
		t.Errorf("expected empty block for empty content, got: %s", block)
	}
}

func TestBuildProjectRulesBlock_NoRuleSections(t *testing.T) {
	content := `# Project

## 概要

Just a project.

## 参照先

See README.
`
	block := BuildProjectRulesBlock(content)
	if block != "" {
		t.Errorf("expected empty block when no rule sections, got: %s", block)
	}
}

func TestBuildProjectRulesBlock_OnlyDevRules(t *testing.T) {
	content := `# Project

## 開発ルール

- テストを書け
- lint を通せ
`
	block := BuildProjectRulesBlock(content)
	if block == "" {
		t.Fatal("expected non-empty block")
	}
	if !strings.Contains(block, "PROJECT-SPECIFIC RULES") {
		t.Error("should contain rules header")
	}
	if strings.Contains(block, "VERIFICATION COMMANDS") {
		t.Error("should NOT contain verification header when section is missing")
	}
	if !strings.Contains(block, "Violating ANY") {
		t.Error("should contain violation warning")
	}
}

func TestBuildProjectRulesBlock_OnlyVerification(t *testing.T) {
	content := `# Project

## Verification Commands

` + "```bash\nmake test\n```" + `
`
	block := BuildProjectRulesBlock(content)
	if block == "" {
		t.Fatal("expected non-empty block")
	}
	if !strings.Contains(block, "VERIFICATION COMMANDS") {
		t.Error("should contain verification header")
	}
	if !strings.Contains(block, "make test") {
		t.Error("should contain verification command content")
	}
}

func TestStripRuleSections(t *testing.T) {
	result := StripRuleSections(testXelyonMD)

	// ルール系セクションが除去されていること
	if strings.Contains(result, "## 開発ルール") {
		t.Error("should NOT contain 開発ルール section")
	}
	if strings.Contains(result, "## Verification Commands") {
		t.Error("should NOT contain Verification Commands section")
	}

	// 他のセクションは残っていること
	if !strings.Contains(result, "## 概要") {
		t.Error("should still contain 概要 section")
	}
	if !strings.Contains(result, "## 技術スタック") {
		t.Error("should still contain 技術スタック section")
	}
	if !strings.Contains(result, "## 参照先") {
		t.Error("should still contain 参照先 section")
	}
}

func TestStripRuleSections_NoRuleSections(t *testing.T) {
	content := `# Project

## 概要

Just a project.
`
	result := StripRuleSections(content)
	if !strings.Contains(result, "Just a project") {
		t.Error("content should remain unchanged when no rule sections")
	}
}

func TestStripRuleSections_Empty(t *testing.T) {
	result := StripRuleSections("")
	if result != "" {
		t.Errorf("expected empty result for empty input, got: %s", result)
	}
}

func TestInjectProjectRules_Normal(t *testing.T) {
	systemPrompt := `## Workflow Rules

### 10. Verification Protocol (MANDATORY)
A task is NOT complete until verification passes

### 11. Impact Analysis
Check references before changes`

	rulesBlock := "\n\n=== PROJECT-SPECIFIC RULES (MANDATORY) ===\nTest rule\nViolating ANY of these rules is a critical failure."

	result := InjectProjectRules(systemPrompt, rulesBlock)

	// Rule #10 の直後に挿入されること
	idx10 := strings.Index(result, "verification passes")
	idxRules := strings.Index(result, "PROJECT-SPECIFIC RULES")
	idx11 := strings.Index(result, "### 11. Impact Analysis")

	if idxRules < 0 {
		t.Fatal("rules block not found in result")
	}
	if idxRules < idx10 {
		t.Error("rules block should come AFTER Rule #10")
	}
	if idx11 < idxRules {
		t.Error("rules block should come BEFORE Rule #11")
	}
}

func TestInjectProjectRules_EmptyBlock(t *testing.T) {
	systemPrompt := "original prompt"
	result := InjectProjectRules(systemPrompt, "")
	if result != systemPrompt {
		t.Error("should return original prompt when block is empty")
	}
}

func TestInjectProjectRules_MarkerNotFound(t *testing.T) {
	systemPrompt := "some prompt without the marker"
	rulesBlock := "\n\nRULES HERE"

	result := InjectProjectRules(systemPrompt, rulesBlock)
	// マーカーがなければ末尾に追加
	if !strings.HasSuffix(result, rulesBlock) {
		t.Error("should append to end when marker not found")
	}
}

func TestEndToEnd_Integration(t *testing.T) {
	// SystemPrompt + XELYON.md の統合テスト
	rulesBlock := BuildProjectRulesBlock(testXelyonMD)
	if rulesBlock == "" {
		t.Fatal("expected non-empty rules block")
	}

	result := InjectProjectRules(SystemPrompt, rulesBlock)

	// Workflow Rules 内に挿入されていること
	idxVerification := strings.Index(result, "verification passes")
	idxProjectRules := strings.Index(result, "PROJECT-SPECIFIC RULES")
	idxImpact := strings.Index(result, "### 11. Impact Analysis")

	if idxProjectRules < 0 {
		t.Fatal("project rules not injected")
	}
	if idxProjectRules < idxVerification {
		t.Error("project rules should come after Rule #10")
	}
	if idxImpact > 0 && idxImpact < idxProjectRules {
		t.Error("project rules should come before Rule #11")
	}

	// ルール内容が含まれていること
	if !strings.Contains(result, "make ci-check") {
		t.Error("should contain verification command")
	}

	// 末尾 Project Context からルールが除去されていること
	stripped := StripRuleSections(testXelyonMD)
	if strings.Contains(stripped, "## 開発ルール") {
		t.Error("stripped content should not contain rule sections")
	}
}

func TestBuildRulesBlockFromList_Empty(t *testing.T) {
	result := BuildRulesBlockFromList(nil)
	if result != "" {
		t.Errorf("expected empty string for nil rules, got %q", result)
	}
	result = BuildRulesBlockFromList([]string{})
	if result != "" {
		t.Errorf("expected empty string for empty rules, got %q", result)
	}
}

func TestBuildRulesBlockFromList_Single(t *testing.T) {
	result := BuildRulesBlockFromList([]string{"Always run tests"})
	if !strings.Contains(result, "PROJECT-SPECIFIC RULES (MANDATORY)") {
		t.Error("should contain MANDATORY header")
	}
	if !strings.Contains(result, "1. Always run tests") {
		t.Error("should contain numbered rule")
	}
	if !strings.Contains(result, "Violating ANY") {
		t.Error("should contain violation warning")
	}
}

func TestBuildRulesBlockFromList_Multiple(t *testing.T) {
	rules := []string{
		"Run go fmt before commit",
		"All tests must pass",
		"No hardcoded secrets",
	}
	result := BuildRulesBlockFromList(rules)
	if !strings.Contains(result, "1. Run go fmt before commit") {
		t.Error("should contain rule 1")
	}
	if !strings.Contains(result, "2. All tests must pass") {
		t.Error("should contain rule 2")
	}
	if !strings.Contains(result, "3. No hardcoded secrets") {
		t.Error("should contain rule 3")
	}
}
