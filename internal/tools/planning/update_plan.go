package planning

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/i18n"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// UpdatePlanTool は計画を更新するツール
type UpdatePlanTool struct {
	storage *plan.PlanStorage
}

// NewUpdatePlanTool は UpdatePlanTool を作成
func NewUpdatePlanTool(storage *plan.PlanStorage) *UpdatePlanTool {
	return &UpdatePlanTool{storage: storage}
}

// Name はツール名を返す
func (t *UpdatePlanTool) Name() string {
	return "update_plan"
}

// Description はツールの説明を返す
func (t *UpdatePlanTool) Description() string {
	return tools.ToolDescriptions[t.Name()]
}

// Parameters はツールのパラメータ定義を返す
func (t *UpdatePlanTool) Parameters() map[string]interface{} {
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
			"action": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"set_status", "add_step", "remove_step", "update_step", "set_title", "set_summary"},
				"description": "実行するアクション。set_status→statusが必須, add_step→stepが必須, remove_step→step_idが必須, update_step→step_id+stepが必須, set_title→titleが必須, set_summary→summaryが必須",
			},
			"status": map[string]interface{}{
				"type":        "string",
				"description": "set_status 用: 新しいステータス",
			},
			"step": map[string]interface{}{
				"type":        "string",
				"description": `add_step/update_step 用: ステップ情報のJSON文字列。例: {"id":5,"description":"New step","tools":["bash"],"depends_on":[]}`,
			},
			"step_id": map[string]interface{}{
				"type":        "integer",
				"description": "remove_step/update_step 用: 対象ステップID",
			},
			"title": map[string]interface{}{
				"type":        "string",
				"description": "set_title 用: 新しいタイトル",
			},
			"summary": map[string]interface{}{
				"type":        "string",
				"description": "set_summary 用: 新しいサマリー",
			},
		},
		"required":             []string{"action"},
		"additionalProperties": false,
	}
}

// Run はツールを実行
func (t *UpdatePlanTool) Run(args map[string]string) (string, *tools.FileChange, error) {
	id := args["id"]
	filename := args["filename"]
	action := args["action"]

	// 計画を取得
	var p *plan.Plan
	var err error

	if id != "" {
		p, filename, err = t.storage.LoadByID(id)
	} else if filename != "" {
		p, err = t.storage.Load(filename)
	} else if lastID := t.storage.LastPlanID(); lastID != "" {
		id = lastID
		p, filename, err = t.storage.LoadByID(lastID)
	} else {
		return "", nil, fmt.Errorf("id or filename is required (no recent plan available)")
	}

	if err != nil {
		return "", nil, fmt.Errorf("%s", i18n.T("plan.not_found", id+filename))
	}

	// アクション実行
	switch action {
	case "set_status":
		p.Status = plan.PlanStatus(args["status"])
	case "set_title":
		p.Title = args["title"]
	case "set_summary":
		p.Summary = args["summary"]
	case "add_step":
		var step plan.PlanStep
		if err := json.Unmarshal([]byte(args["step"]), &step); err != nil {
			return "", nil, fmt.Errorf("invalid step JSON: %w", err)
		}
		step.Status = "pending"
		p.Steps = append(p.Steps, step)
	case "remove_step":
		stepID, _ := strconv.Atoi(args["step_id"])
		var newSteps []plan.PlanStep
		for _, s := range p.Steps {
			if s.ID != stepID {
				newSteps = append(newSteps, s)
			}
		}
		p.Steps = newSteps
	case "update_step":
		stepID, _ := strconv.Atoi(args["step_id"])
		var updates plan.PlanStep
		if err := json.Unmarshal([]byte(args["step"]), &updates); err != nil {
			return "", nil, fmt.Errorf("invalid step JSON: %w", err)
		}
		for i := range p.Steps {
			if p.Steps[i].ID == stepID {
				if updates.Description != "" {
					p.Steps[i].Description = updates.Description
				}
				if updates.Status != "" {
					p.Steps[i].Status = updates.Status
				}
				if len(updates.Tools) > 0 {
					p.Steps[i].Tools = updates.Tools
				}
				if len(updates.Files) > 0 {
					p.Steps[i].Files = updates.Files
				}
				break
			}
		}
	default:
		return "", nil, fmt.Errorf("unknown action: %s", action)
	}

	// 保存（タイトル変更時は古いファイルを削除）
	oldFilename := filename
	newFilename, err := t.storage.Save(p)
	if err != nil {
		return "", nil, err
	}

	// ファイル名が変わった場合、古いファイルを削除
	if newFilename != oldFilename && oldFilename != "" {
		_ = t.storage.Delete(oldFilename) // エラーは無視
	}

	return i18n.T("plan.updated", p.Title), nil, nil
}
