package planning

import (
	"encoding/json"
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/i18n"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// GetPlanTool は保存された計画を取得するツール
type GetPlanTool struct {
	storage *plan.PlanStorage
}

// NewGetPlanTool は GetPlanTool を作成
func NewGetPlanTool(storage *plan.PlanStorage) *GetPlanTool {
	return &GetPlanTool{storage: storage}
}

// Name はツール名を返す
func (t *GetPlanTool) Name() string {
	return "get_plan"
}

// Description はツールの説明を返す
func (t *GetPlanTool) Description() string {
	return tools.ToolDescriptions[t.Name()]
}

// Parameters はツールのパラメータ定義を返す
func (t *GetPlanTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type":        "string",
				"description": "計画の ID（UUID）",
			},
			"filename": map[string]interface{}{
				"type":        "string",
				"description": "計画のファイル名（例: 20260201_auth-feature.md）",
			},
		},
		"additionalProperties": false,
	}
}

// Run はツールを実行
func (t *GetPlanTool) Run(args map[string]string) (string, *tools.FileChange, error) {
	id := args["id"]
	filename := args["filename"]

	var p *plan.Plan
	var err error

	// ID を優先
	if id != "" {
		p, _, err = t.storage.LoadByID(id)
	} else if filename != "" {
		p, err = t.storage.Load(filename)
	} else {
		return "", nil, fmt.Errorf("id or filename is required")
	}

	if err != nil {
		return "", nil, fmt.Errorf("%s", i18n.T("plan.not_found", id+filename))
	}

	// JSON で返す
	data, _ := json.MarshalIndent(p, "", "  ")
	return string(data), nil, nil
}
