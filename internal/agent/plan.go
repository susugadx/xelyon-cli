package agent

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Plan は実行計画を表す
type Plan struct {
	Steps []PlanStep `json:"steps"`
}

// PlanStep は計画の1ステップを表す
type PlanStep struct {
	ID          int      `json:"id"`
	Description string   `json:"description"`
	Tools       []string `json:"tools"`      // 使用予定ツール
	DependsOn   []int    `json:"depends_on"` // 依存するステップID
	Parallel    bool     `json:"parallel"`   // 並列実行可能か
	Status      string   `json:"status"`     // "pending", "running", "completed", "failed"
	Result      string   `json:"result"`     // 実行結果
}

// ParsePlan はJSON文字列からPlanを解析
func ParsePlan(jsonStr string) (*Plan, error) {
	var plan Plan
	if err := json.Unmarshal([]byte(jsonStr), &plan); err != nil {
		return nil, fmt.Errorf("failed to parse plan: %w", err)
	}

	// 初期ステータス設定
	for i := range plan.Steps {
		plan.Steps[i].Status = "pending"
	}

	return &plan, nil
}

// ExtractPlanJSON はレスポンスからPlan JSONを抽出
func ExtractPlanJSON(response string) (string, error) {
	// 方法1: ```json ブロック内のJSONを探す
	if idx := strings.Index(response, "```json"); idx != -1 {
		start := strings.Index(response[idx:], "{")
		if start != -1 {
			start += idx
			end := findClosingBrace(response, start)
			if end != -1 {
				return response[start:end], nil
			}
		}
	}

	// 方法2: "steps" キーを含むJSONオブジェクトを探す（空白を無視）
	// {"steps" または { "steps" または {\n"steps" などに対応
	stepsPattern := []string{
		"{\"steps\":",
		"{ \"steps\":",
		"{\n\"steps\":",
		"{\n \"steps\":",
		"{\n  \"steps\":",
	}

	start := -1
	for _, pattern := range stepsPattern {
		if idx := strings.Index(response, pattern); idx != -1 {
			start = idx
			break
		}
	}

	// 方法3: 単純に最初の { を探す（最終手段）
	if start == -1 {
		start = strings.Index(response, "{")
	}

	if start == -1 {
		return "", fmt.Errorf("plan JSON not found in response")
	}

	// 対応する閉じ括弧を探す
	end := findClosingBrace(response, start)
	if end == -1 {
		return "", fmt.Errorf("incomplete plan JSON")
	}

	return response[start:end], nil
}

// findClosingBrace は対応する閉じ括弧の位置を探す
func findClosingBrace(response string, start int) int {
	depth := 0
	inString := false
	escaped := false

	for i := start; i < len(response); i++ {
		ch := response[i]

		// エスケープ処理
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}

		// 文字列内チェック
		if ch == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		// 括弧の深さチェック
		switch ch {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}

	return -1
}

// FormatPlan は計画を見やすく整形
func FormatPlan(plan *Plan) string {
	var sb strings.Builder
	sb.WriteString("📋 Plan:\n")

	for _, step := range plan.Steps {
		parallelTag := "[順次]"
		if step.Parallel {
			parallelTag = "[並列]"
		}

		sb.WriteString(fmt.Sprintf("  %d. %s %s\n", step.ID, parallelTag, step.Description))

		if len(step.Tools) > 0 {
			sb.WriteString(fmt.Sprintf("     Tools: %s\n", strings.Join(step.Tools, ", ")))
		}

		if len(step.DependsOn) > 0 {
			sb.WriteString(fmt.Sprintf("     Depends on: %v\n", step.DependsOn))
		}
	}

	return sb.String()
}

// CanExecute はステップが実行可能かチェック
func (p *Plan) CanExecute(stepID int) bool {
	step := p.GetStep(stepID)
	if step == nil {
		return false
	}

	// 依存するステップがすべて完了しているかチェック
	for _, depID := range step.DependsOn {
		depStep := p.GetStep(depID)
		if depStep == nil || depStep.Status != "completed" {
			return false
		}
	}

	return step.Status == "pending"
}

// GetStep は指定IDのステップを取得
func (p *Plan) GetStep(id int) *PlanStep {
	for i := range p.Steps {
		if p.Steps[i].ID == id {
			return &p.Steps[i]
		}
	}
	return nil
}

// GetParallelSteps は並列実行可能なステップを取得
func (p *Plan) GetParallelSteps() []int {
	var ids []int
	for _, step := range p.Steps {
		if step.Parallel && p.CanExecute(step.ID) {
			ids = append(ids, step.ID)
		}
	}
	return ids
}

// GetNextStep は次に実行すべきステップIDを取得
func (p *Plan) GetNextStep() int {
	for _, step := range p.Steps {
		if p.CanExecute(step.ID) {
			return step.ID
		}
	}
	return -1 // すべて完了またはブロック中
}

// UpdateStatus はステップのステータスを更新
func (p *Plan) UpdateStatus(stepID int, status, result string) {
	step := p.GetStep(stepID)
	if step != nil {
		step.Status = status
		step.Result = result
	}
}

// IsCompleted はすべてのステップが完了したかチェック
func (p *Plan) IsCompleted() bool {
	for _, step := range p.Steps {
		if step.Status != "completed" {
			return false
		}
	}
	return true
}

// HasFailed は失敗したステップがあるかチェック
func (p *Plan) HasFailed() bool {
	for _, step := range p.Steps {
		if step.Status == "failed" {
			return true
		}
	}
	return false
}
