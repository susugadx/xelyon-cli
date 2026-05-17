package plan

import (
	"encoding/json"
	"fmt"
)

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
