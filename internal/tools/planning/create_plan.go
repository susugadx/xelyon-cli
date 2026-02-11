package planning

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/i18n"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// CreatePlanTool は計画を作成・保存するツール
type CreatePlanTool struct {
	storage  *plan.PlanStorage
	lastPlan *plan.Plan
}

// NewCreatePlanTool は CreatePlanTool を作成
func NewCreatePlanTool(storage *plan.PlanStorage) *CreatePlanTool {
	return &CreatePlanTool{storage: storage}
}

// Name はツール名を返す
func (t *CreatePlanTool) Name() string {
	return "create_plan"
}

// Description はツールの説明を返す
func (t *CreatePlanTool) Description() string {
	return tools.ToolDescriptions[t.Name()]
}

// Parameters はツールのパラメータ定義を返す
func (t *CreatePlanTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"title": map[string]interface{}{
				"type":        "string",
				"description": "計画のタイトル",
			},
			"summary": map[string]interface{}{
				"type":        "string",
				"description": "計画の概要",
			},
			"user_request": map[string]interface{}{
				"type":        "string",
				"description": "ユーザーの元のリクエスト（オプション）",
			},
			"clarifications": map[string]interface{}{
				"type":        "string",
				"description": `質問と回答のJSON配列文字列（オプション）。例: [{"question":"Q","answer":"A"}]`,
			},
			"steps": map[string]interface{}{
				"type":        "string",
				"description": `実行ステップのJSON配列文字列。例: [{"id":1,"description":"Step 1","tools":["read_file"],"depends_on":[]}]`,
			},
		},
		"required":             []string{"title", "summary", "steps"},
		"additionalProperties": false,
	}
}

// Run はツールを実行
func (t *CreatePlanTool) Run(args map[string]string) (string, *tools.FileChange, error) {
	title := args["title"]
	summary := args["summary"]
	userRequest := args["user_request"]
	clarificationsJSON := args["clarifications"]
	stepsJSON := args["steps"]

	// clarifications のパース
	var clarifications []plan.Clarification
	if clarificationsJSON != "" {
		if err := json.Unmarshal([]byte(clarificationsJSON), &clarifications); err != nil {
			return "", nil, fmt.Errorf("invalid clarifications format: %w", err)
		}
	}

	// steps のパース
	var steps []plan.PlanStep
	if err := json.Unmarshal([]byte(stepsJSON), &steps); err != nil {
		return "", nil, fmt.Errorf("invalid steps format: %w", err)
	}

	// ステップのステータスを初期化
	for i := range steps {
		steps[i].Status = "pending"
	}

	// Plan を作成
	now := time.Now()
	p := &plan.Plan{
		ID:             uuid.New().String(),
		Title:          title,
		Summary:        summary,
		UserRequest:    userRequest,
		Clarifications: clarifications,
		Steps:          steps,
		Status:         plan.PlanStatusPending,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// 保存
	filename, err := t.storage.Save(p)
	if err != nil {
		return "", nil, fmt.Errorf("failed to save plan: %w", err)
	}

	// 作成した Plan を保持
	t.lastPlan = p

	return fmt.Sprintf("%s\nPlan ID: %s\nFilename: %s", i18n.T("plan.created", title), p.ID, filename), nil, nil
}

// LastPlan は最後に作成した Plan を返す
func (t *CreatePlanTool) LastPlan() *plan.Plan {
	return t.lastPlan
}

// ClearLastPlan は lastPlan をクリアする
func (t *CreatePlanTool) ClearLastPlan() {
	t.lastPlan = nil
}
