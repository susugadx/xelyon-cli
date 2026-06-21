package uiplanview

import "github.com/susugadx/xelyon-cli/internal/uiprompt"

const planReviewActionHint = "Approve starts implementation from this plan. Request changes sends feedback and regenerates the plan. Cancel exits Plan Mode without implementation."

// NewPlanReviewDisplay は承認前レビュー用の PlanDisplay を作成する。
func NewPlanReviewDisplay() *PlanDisplay {
	return NewPlanDisplay("Implementation Plan Review").
		SetFilesTitle("関連ファイル").
		SetFooter(PlanReviewActionHint())
}

// PlanReviewActionHint は Plan review 表示の操作説明を返す。
func PlanReviewActionHint() string {
	return planReviewActionHint
}

// NewPlanApprovalPromptRequest は Plan 承認ダイアログ用の PromptRequest を返す。
func NewPlanApprovalPromptRequest() uiprompt.PromptRequest {
	return uiprompt.PromptRequest{
		Kind:                uiprompt.PromptKindConfirm,
		Title:               "Review implementation plan",
		Message:             "Approve the plan, request changes, or cancel Plan Mode.",
		AllowComment:        true,
		ConfirmSubmitPolicy: uiprompt.PromptConfirmSubmitExplicit,
		Placeholder:         "Describe what should change before implementation...",
		Options: []uiprompt.PromptOption{
			{Label: "Approve", Description: "Start implementation", Value: string(uiprompt.PromptActionYes)},
			{Label: "Request changes", Description: "Send feedback and re-plan", Value: string(uiprompt.PromptActionComment)},
			{Label: "Cancel", Description: "Exit Plan Mode", Value: string(uiprompt.PromptActionNo)},
		},
	}
}
