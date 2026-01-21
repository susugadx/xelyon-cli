package agent

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Plan は実行計画を表す
type Plan struct {
	Summary string     `json:"summary"`
	Steps   []PlanStep `json:"steps"`
}

// PlanStep は計画の1ステップを表す
type PlanStep struct {
	ID            int      `json:"id"`
	Description   string   `json:"description"`
	Tools         []string `json:"tools"`      // 使用予定ツール
	DependsOn     []int    `json:"depends_on"` // 依存するステップID
	Status        string   `json:"status"`     // "pending", "running", "completed", "failed"
	Result        string   `json:"result"`     // 実行結果
	ToolsExecuted int      `json:"-"`          // 実際に実行されたツール数

	// 将来の依存関係解析用（現時点では未使用）
	TargetFiles []string `json:"-"` // 操作対象ファイル（推論結果）
	ReadFiles   []string `json:"-"` // 読み取りファイル
	WriteFiles  []string `json:"-"` // 書き込みファイル
}

// ParsePlan はJSON文字列からPlanを解析
//
// 互換対応:
// - V2形式: {"plan": {...}}
// - 旧形式: {"summary": "...", "steps": [...]}
func ParsePlan(jsonStr string) (*Plan, error) {
	// 1) V2 wrapper形式を先に試す
	type wrapper struct {
		Plan Plan `json:"plan"`
	}
	var w wrapper
	if err := json.Unmarshal([]byte(jsonStr), &w); err == nil && (w.Plan.Summary != "" || len(w.Plan.Steps) > 0) {
		for i := range w.Plan.Steps {
			w.Plan.Steps[i].Status = "pending"
		}
		return &w.Plan, nil
	}

	// 2) 旧形式（wrapperなし）
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

// ParsePlanWithDependencyAnalysis はPlanを解析し、依存関係を自動推論
// lspClient が nil の場合もファイルベースの依存関係推論は動作する
func ParsePlanWithDependencyAnalysis(jsonStr string, lspClient interface{}) (*Plan, []string, error) {
	plan, err := ParsePlan(jsonStr)
	if err != nil {
		return nil, nil, err
	}

	// 依存関係解析
	var warnings []string
	plan, warnings = AnalyzePlanDependencies(plan, lspClient)

	return plan, warnings, nil
}

// AnalyzePlanDependencies はプランの依存関係を解析・補完
func AnalyzePlanDependencies(plan *Plan, lspClient interface{}) (*Plan, []string) {
	if plan == nil || len(plan.Steps) == 0 {
		return plan, nil
	}

	// LSPクライアントの型アサーション
	var typedLSPClient interface {
		// 必要なメソッドがあればここに定義
	}
	if lspClient != nil {
		typedLSPClient = lspClient
	}
	_ = typedLSPClient // 将来の LSP 強化用

	// DependencyAnalyzer を使用（LSPクライアントはnilでも動作）
	analyzer := NewDependencyAnalyzer(nil) // LSP統合は後で実装
	result := analyzer.Analyze(plan.Steps)

	plan.Steps = result.Steps
	return plan, result.Warnings
}

// ExtractPlanJSON はレスポンスからPlan JSONを抽出
// 見つからない場合は空文字列を返す
// NOTE: ツール呼び出し JSON ({"tool": ...}) は plan ではないので除外する
func ExtractPlanJSON(response string) string {
	// 方法1: {"plan": ... } パターンを探す（V2形式）
	planPatterns := []string{
		`{"plan"`,
		`{ "plan"`,
	}

	for _, pattern := range planPatterns {
		idx := strings.Index(response, pattern)
		if idx != -1 {
			end := findClosingBrace(response, idx)
			if end != -1 {
				return response[idx:end]
			}
		}
	}

	// 方法2: ```json ブロック内のJSONを探す（plan関連の内容のみ）
	if idx := strings.Index(response, "```json"); idx != -1 {
		start := strings.Index(response[idx:], "{")
		if start != -1 {
			start += idx
			end := findClosingBrace(response, start)
			if end != -1 {
				jsonStr := response[start:end]
				// ツール呼び出しは plan ではない
				if !isToolCallJSON(jsonStr) {
					return jsonStr
				}
			}
		}
	}

	// 方法3: "steps" キーを含むJSONオブジェクトを探す（空白を無視）
	// {"steps" または { "steps" または {\n"steps" などに対応
	stepsPattern := []string{
		"{\"steps\":",
		"{ \"steps\":",
		"{\n\"steps\":",
		"{\n \"steps\":",
		"{\n  \"steps\":",
	}

	for _, pattern := range stepsPattern {
		if idx := strings.Index(response, pattern); idx != -1 {
			end := findClosingBrace(response, idx)
			if end != -1 {
				return response[idx:end]
			}
		}
	}

	// 方法4は削除: 単純な { 検出は誤検出が多いため
	// ツール呼び出し JSON ({"tool": ...}) を plan として誤検出していた

	return ""
}

// isToolCallJSON はJSONがツール呼び出しかどうかを判定
func isToolCallJSON(jsonStr string) bool {
	// ツール呼び出しのパターン
	toolPatterns := []string{
		`"tool"`,
		`"tool":`,
	}
	for _, pattern := range toolPatterns {
		if strings.Contains(jsonStr, pattern) {
			return true
		}
	}
	return false
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
		sb.WriteString(fmt.Sprintf("  %d. %s\n", step.ID, step.Description))

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
// depends_on が同じステップは並列実行可能と判定
func (p *Plan) GetParallelSteps() []int {
	// 実行可能なステップを収集
	var executableSteps []*PlanStep
	for i := range p.Steps {
		if p.CanExecute(p.Steps[i].ID) {
			executableSteps = append(executableSteps, &p.Steps[i])
		}
	}

	if len(executableSteps) <= 1 {
		// 1つ以下なら並列実行の必要なし
		return nil
	}

	// depends_on が同じステップをグループ化
	// 最初の実行可能ステップの depends_on と同じものを収集
	firstDeps := executableSteps[0].DependsOn
	var parallelIDs []int

	for _, step := range executableSteps {
		if sameDependencies(step.DependsOn, firstDeps) {
			parallelIDs = append(parallelIDs, step.ID)
		}
	}

	if len(parallelIDs) <= 1 {
		return nil
	}

	return parallelIDs
}

// sameDependencies は2つの依存リストが同じかチェック
func sameDependencies(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	aSet := make(map[int]bool)
	for _, id := range a {
		aSet[id] = true
	}
	for _, id := range b {
		if !aSet[id] {
			return false
		}
	}
	return true
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

// IncrementToolsExecuted は実行ツール数をインクリメント
func (p *Plan) IncrementToolsExecuted(stepID int) {
	step := p.GetStep(stepID)
	if step != nil {
		step.ToolsExecuted++
	}
}

// GetToolsExecuted は実行ツール数を取得
func (p *Plan) GetToolsExecuted(stepID int) int {
	step := p.GetStep(stepID)
	if step != nil {
		return step.ToolsExecuted
	}
	return 0
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
