package uiplanview

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/uiprompt"
)

func TestNewPlanReviewDisplay_RendersReviewActions(t *testing.T) {
	p := NewPlanReviewDisplay().
		SetSummary("Update Plan Mode review flow").
		AddStep(1, "Update review display", []string{"apply_patch"}, []string{"internal/uiplanview/plan_review.go"})

	result := p.Render()
	for _, want := range []string{
		"Implementation Plan Review",
		"関連ファイル",
		"internal/uiplanview/plan_review.go",
		"確認",
		"Approve starts implementation",
		"Request changes sends feedback",
		"Cancel exits Plan Mode",
	} {
		if !strings.Contains(result, want) {
			t.Fatalf("Plan review display = %q, want fragment %q", result, want)
		}
	}
}

func TestNewPlanApprovalPromptRequest_UsesPlanReviewActions(t *testing.T) {
	req := NewPlanApprovalPromptRequest()

	if req.Kind != uiprompt.PromptKindConfirm {
		t.Fatalf("Kind = %q, want confirm", req.Kind)
	}
	if !req.AllowComment {
		t.Fatal("AllowComment = false, want true")
	}
	if req.ConfirmSubmitPolicy != uiprompt.PromptConfirmSubmitExplicit {
		t.Fatalf("ConfirmSubmitPolicy = %q, want explicit", req.ConfirmSubmitPolicy)
	}
	if len(req.Options) != 3 {
		t.Fatalf("len(Options) = %d, want 3", len(req.Options))
	}
	assertPlanReviewPromptOption(t, req.Options[0], "Approve", string(uiprompt.PromptActionYes))
	assertPlanReviewPromptOption(t, req.Options[1], "Request changes", string(uiprompt.PromptActionComment))
	assertPlanReviewPromptOption(t, req.Options[2], "Cancel", string(uiprompt.PromptActionNo))
	if !strings.Contains(req.Placeholder, "change") {
		t.Fatalf("Placeholder = %q, want feedback guidance", req.Placeholder)
	}
}

func assertPlanReviewPromptOption(t *testing.T, option uiprompt.PromptOption, label string, value string) {
	t.Helper()
	if option.Label != label || option.Value != value {
		t.Fatalf("PromptOption = %#v, want label %q value %q", option, label, value)
	}
}
