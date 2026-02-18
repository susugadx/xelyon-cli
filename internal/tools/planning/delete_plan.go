package planning

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/i18n"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// DeletePlanTool は計画を削除するツール
type DeletePlanTool struct {
	storage *plan.PlanStorage
}

// NewDeletePlanTool は DeletePlanTool を作成
func NewDeletePlanTool(storage *plan.PlanStorage) *DeletePlanTool {
	return &DeletePlanTool{storage: storage}
}

// Name はツール名を返す
func (t *DeletePlanTool) Name() string {
	return "delete_plan"
}

// Description はツールの説明を返す
func (t *DeletePlanTool) Description() string {
	return tools.ToolDescriptions[t.Name()]
}

// Parameters はツールのパラメータ定義を返す
func (t *DeletePlanTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type":        "string",
				"description": "計画の ID（UUID）。create_plan が返す Plan ID を指定。省略時は直前のプランを使用",
			},
			"filename": map[string]interface{}{
				"type":        "string",
				"description": "計画のファイル名。id の代わりに使用可能",
			},
		},
		"additionalProperties": false,
	}
}

// Run はツールを実行
func (t *DeletePlanTool) Run(args map[string]string) (string, *tools.FileChange, error) {
	id := args["id"]
	filename := args["filename"]

	// ID から filename を取得（id → filename → lastPlanID の順でフォールバック）
	if id != "" {
		_, fn, err := t.storage.LoadByID(id)
		if err != nil {
			return "", nil, fmt.Errorf("%s", i18n.T("plan.not_found", id))
		}
		filename = fn
	} else if filename == "" {
		if lastID := t.storage.LastPlanID(); lastID != "" {
			_, fn, err := t.storage.LoadByID(lastID)
			if err != nil {
				return "", nil, fmt.Errorf("%s", i18n.T("plan.not_found", lastID))
			}
			filename = fn
		} else {
			return "", nil, fmt.Errorf("id or filename is required (no recent plan available)")
		}
	}

	if err := t.storage.Delete(filename); err != nil {
		return "", nil, err
	}

	t.storage.ClearLastPlanID()

	return i18n.T("plan.deleted", filename), nil, nil
}
