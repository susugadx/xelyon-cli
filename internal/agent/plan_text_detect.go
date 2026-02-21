package agent

import (
	"regexp"
	"strconv"
	"strings"
)

var textPlanPatterns = []*regexp.Regexp{
	// "1. xxx\n2. xxx\n3. xxx"
	regexp.MustCompile(`(?m)^\s*(\d+)\.\s+(.+)$`),
	// "Step 1: xxx" / "Step 1. xxx"
	regexp.MustCompile(`(?mi)^\s*step\s+(\d+)[.:]\s*(.+)$`),
	// "ステップ1: xxx" / "ステップ 1: xxx"
	regexp.MustCompile(`(?m)^\s*ステップ\s*(\d+)[.:：]\s*(.+)$`),
}

// extractTextPlan extracts numbered steps from text response.
// It returns a slice of step descriptions if 3+ sequential steps are found.
func extractTextPlan(response string) []string {
	for _, pattern := range textPlanPatterns {
		matches := pattern.FindAllStringSubmatch(response, -1)
		if len(matches) >= 3 && isSequential(matches) {
			var steps []string
			for _, m := range matches {
				steps = append(steps, strings.TrimSpace(m[2]))
			}
			return steps
		}
	}
	return nil
}

// isSequential checks if steps are numbered 1, 2, 3...
func isSequential(matches [][]string) bool {
	for i, m := range matches {
		num, err := strconv.Atoi(m[1])
		if err != nil || num != i+1 {
			return false
		}
	}
	return true
}

var actionVerbs = []string{
	// File operations
	"作成", "追加", "修正", "更新", "変更", "削除", "書き換え", "リファクタ", "実装", "移動",
	"create", "add", "fix", "update", "modify", "delete", "replace", "refactor", "implement", "move",
	// Bash operations
	"実行", "インストール", "ビルド", "テスト", "設定",
	"run", "install", "build", "test", "configure",
	// Write operations
	"write", "edit", "rename", "migrate", "optimize", "debug", "remove", "merge", "check",
	"書く", "編集", "リネーム", "移行", "最適化", "デバッグ", "削除", "マージ", "確認",
}

var analysisVerbs = []string{
	"原因", "可能性", "理由", "考えられ", "について", "確認", "調査", "分析",
	"cause", "reason", "because", "might", "could be", "about", "investigate", "analyze",
}

// isActionPlan checks if the steps are actionable tasks rather than analysis/explanation.
func isActionPlan(steps []string) bool {
	actionCount := 0
	for _, step := range steps {
		lowered := strings.ToLower(step)
		hasAction := false
		for _, verb := range actionVerbs {
			if strings.Contains(lowered, strings.ToLower(verb)) {
				hasAction = true
				break
			}
		}
		hasAnalysis := false
		for _, verb := range analysisVerbs {
			if strings.Contains(lowered, strings.ToLower(verb)) {
				hasAnalysis = true
				break
			}
		}
		if hasAction && !hasAnalysis {
			actionCount++
		}
	}
	return actionCount > len(steps)/2
}

// isPlanTool はプラン管理系ツールかを判定する。
// delete_plan / create_plan / update_plan 等は Step Tracking のカウント対象とする。
func isPlanTool(toolName string) bool {
	switch toolName {
	case "create_plan", "update_plan", "delete_plan", "get_plan", "list_plans":
		return true
	}
	return false
}
