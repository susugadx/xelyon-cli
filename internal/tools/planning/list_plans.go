package planning

import (
	"encoding/json"
	"strconv"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/i18n"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// ListPlansTool は保存された計画の一覧を取得するツール
type ListPlansTool struct {
	storage *plan.PlanStorage
}

// NewListPlansTool は ListPlansTool を作成
func NewListPlansTool(storage *plan.PlanStorage) *ListPlansTool {
	return &ListPlansTool{storage: storage}
}

// Name はツール名を返す
func (t *ListPlansTool) Name() string {
	return "list_plans"
}

// Description はツールの説明を返す
func (t *ListPlansTool) Description() string {
	return `保存された計画の一覧を取得します。

オプションでステータスによるフィルタリングが可能です。
新しい順に返します。`
}

// Parameters はツールのパラメータ定義を返す
func (t *ListPlansTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"status": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"draft", "pending", "approved", "running", "completed", "failed", "cancelled"},
				"description": "フィルタするステータス（省略時は全て）",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "取得する最大件数（デフォルト: 10）",
			},
		},
		"additionalProperties": false,
	}
}

// Run はツールを実行
func (t *ListPlansTool) Run(args map[string]string) (string, *tools.FileChange, error) {
	statusFilter := args["status"]
	limitStr := args["limit"]

	limit := 10
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
			limit = n
		}
	}

	plans, err := t.storage.List()
	if err != nil {
		return "", nil, err
	}

	// ステータスフィルタ
	if statusFilter != "" {
		var filtered []plan.PlanMetadata
		for _, p := range plans {
			if string(p.Status) == statusFilter {
				filtered = append(filtered, p)
			}
		}
		plans = filtered
	}

	if len(plans) == 0 {
		return i18n.T("plan.list_empty"), nil, nil
	}

	// 件数制限
	if len(plans) > limit {
		plans = plans[:limit]
	}

	data, _ := json.MarshalIndent(plans, "", "  ")
	return string(data), nil, nil
}
