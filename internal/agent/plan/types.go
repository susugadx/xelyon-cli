package plan

import (
	"encoding/json"
	"strconv"
	"time"
)

// PlanStatus は計画の全体ステータス
type PlanStatus string

const (
	PlanStatusDraft     PlanStatus = "draft"
	PlanStatusPending   PlanStatus = "pending"
	PlanStatusApproved  PlanStatus = "approved"
	PlanStatusRunning   PlanStatus = "running"
	PlanStatusCompleted PlanStatus = "completed"
	PlanStatusFailed    PlanStatus = "failed"
	PlanStatusCancelled PlanStatus = "cancelled"
)

// Clarification はユーザーへの質問と回答
type Clarification struct {
	Question     string    `json:"question"`
	QuestionType string    `json:"question_type"` // single_choice, multi_choice, free_text
	Options      []string  `json:"options,omitempty"`
	Answer       string    `json:"answer,omitempty"`
	Answers      []string  `json:"answers,omitempty"`
	AskedAt      time.Time `json:"asked_at"`
}

// StepLog はステップ実行の詳細ログ
type StepLog struct {
	StepID     int               `json:"step_id"`
	ToolName   string            `json:"tool_name"`
	ToolArgs   map[string]string `json:"tool_args,omitempty"`
	Result     string            `json:"result,omitempty"`
	Error      string            `json:"error,omitempty"`
	ExecutedAt time.Time         `json:"executed_at"`
	DurationMs int64             `json:"duration_ms"`
}

// Plan は実行計画を表す
type Plan struct {
	// 既存フィールド
	Summary            string     `json:"summary"`
	AcceptanceCriteria []string   `json:"acceptance_criteria,omitempty"` // 完了条件
	Findings           []string   `json:"findings,omitempty"`            // 調査で分かった重要事実
	Evidence           []string   `json:"evidence,omitempty"`            // 関連ファイル、関数、テスト、根拠
	Constraints        []string   `json:"constraints,omitempty"`         // 守るべき制約や避けるべき変更
	Steps              []PlanStep `json:"steps"`
	OpenQuestions      []string   `json:"open_questions,omitempty"` // 未解決質問

	// メタデータ（新規）
	ID             string          `json:"id,omitempty"`
	Title          string          `json:"title,omitempty"`
	CreatedAt      time.Time       `json:"created_at,omitempty"`
	UpdatedAt      time.Time       `json:"updated_at,omitempty"`
	Status         PlanStatus      `json:"status,omitempty"`
	Model          string          `json:"model,omitempty"`
	Provider       string          `json:"provider,omitempty"`
	UserRequest    string          `json:"user_request,omitempty"`
	Clarifications []Clarification `json:"clarifications,omitempty"`
	ExecutionLog   []StepLog       `json:"execution_log,omitempty"`
}

// AddLog は実行ログを追加
func (p *Plan) AddLog(log StepLog) {
	p.ExecutionLog = append(p.ExecutionLog, log)
	p.UpdatedAt = time.Now()
}

// PlanStep は計画の1ステップを表す
type PlanStep struct {
	ID            int      `json:"id"`
	Description   string   `json:"description"`
	Purpose       string   `json:"purpose,omitempty"`
	Tools         []string `json:"tools"`      // 使用予定ツール
	DependsOn     []int    `json:"depends_on"` // 依存するステップID
	Status        string   `json:"status"`     // "pending", "running", "completed", "failed"
	Result        string   `json:"result"`     // 実行結果
	ToolsExecuted int      `json:"-"`          // 実際に実行されたツール数

	// ファイルアクセス情報（依存関係解析で使用）
	TargetFiles []string `json:"-"` // 操作対象ファイル（推論結果）
	ReadFiles   []string `json:"-"` // 読み取りファイル
	WriteFiles  []string `json:"-"` // 書き込みファイル

	// 追加フィールド（Phase 1）
	Files        []string   `json:"files,omitempty"`        // 関連ファイル
	Verification []string   `json:"verification,omitempty"` // 検証コマンド・確認観点
	StartedAt    *time.Time `json:"started_at,omitempty"`   // 開始時刻
	CompletedAt  *time.Time `json:"completed_at,omitempty"` // 完了時刻
}

// UnmarshalJSON は PlanStep のカスタムデシリアライズ
// Gemini が depends_on を []string（例: ["1","2"]）で返すケースに対応
func (ps *PlanStep) UnmarshalJSON(data []byte) error {
	type Alias PlanStep
	type flex struct {
		Alias
		DependsOnRaw json.RawMessage `json:"depends_on"`
	}
	var f flex
	if err := json.Unmarshal(data, &f); err != nil {
		return err
	}
	*ps = PlanStep(f.Alias)
	if len(f.DependsOnRaw) == 0 || string(f.DependsOnRaw) == "null" {
		return nil
	}
	// []int → OK
	if err := json.Unmarshal(f.DependsOnRaw, &ps.DependsOn); err == nil {
		return nil
	}
	// []string → int に変換
	var strs []string
	if err := json.Unmarshal(f.DependsOnRaw, &strs); err == nil {
		ps.DependsOn = make([]int, 0, len(strs))
		for _, s := range strs {
			if v, e := strconv.Atoi(s); e == nil {
				ps.DependsOn = append(ps.DependsOn, v)
			}
		}
		return nil
	}
	// 単一 int
	var single int
	if err := json.Unmarshal(f.DependsOnRaw, &single); err == nil {
		ps.DependsOn = []int{single}
	}
	return nil
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
