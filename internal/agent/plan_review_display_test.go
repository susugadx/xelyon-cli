package agent

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
)

func TestBuildPlanReviewDisplay_IncludesParsedPlanFiles(t *testing.T) {
	display := buildPlanReviewDisplay(&plan.Plan{
		Summary:     "Ship a reviewable plan",
		Findings:    []string{"plan_request.go owns approval handoff"},
		Evidence:    []string{"internal/agent/plan_request.go: handleInvestigationResult"},
		Constraints: []string{"Do not carry raw investigation history"},
		Steps: []plan.PlanStep{{
			ID:          1,
			Description: "Update plan review UI",
			Purpose:     "Make the approval screen readable",
			Tools:       []string{"apply_patch", "go test ./internal/agent"},
			Files:       []string{" internal/agent/plan_request.go ", "internal/agent/plan_request_test.go"},
			Verification: []string{
				"go test ./internal/agent",
			},
		}},
	})

	rendered := display.Render()
	for _, want := range []string{
		"Implementation Plan Review",
		"Ship a reviewable plan",
		"調査結果",
		"plan_request.go owns approval handoff",
		"根拠",
		"internal/agent/plan_request.go: handleInvestigationResult",
		"制約",
		"Do not carry raw investigation history",
		"Update plan review UI",
		"目的: Make the approval screen readable",
		"触るファイル: internal/agent/plan_request.go, internal/agent/plan_request_test.go",
		"検証: go test ./internal/agent",
		"Tools: apply_patch, go test ./internal/agent",
		"関連ファイル",
		"internal/agent/plan_request.go",
		"internal/agent/plan_request_test.go",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered plan review = %q, want fragment %q", rendered, want)
		}
	}
}

func TestBuildPlanNoImplementationDisplay_IncludesHandoffDetails(t *testing.T) {
	display := buildPlanNoImplementationDisplay(&plan.Plan{
		Summary:     "No implementation is required",
		Findings:    []string{"The existing command already handles this"},
		Evidence:    []string{"internal/agent/plan_request.go: handleInvestigationResult"},
		Constraints: []string{"Do not change CLI output"},
		Steps:       []plan.PlanStep{},
	})
	if display == nil {
		t.Fatal("display = nil, want no-implementation details")
	}

	rendered := display.Render()
	for _, want := range []string{
		"Investigation Result",
		"No implementation is required",
		"調査結果",
		"The existing command already handles this",
		"根拠",
		"internal/agent/plan_request.go: handleInvestigationResult",
		"制約",
		"Do not change CLI output",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered no-implementation display = %q, want fragment %q", rendered, want)
		}
	}
	if strings.Contains(rendered, "ステップ") {
		t.Fatalf("rendered no-implementation display = %q, should not render step sections", rendered)
	}
}

func TestPlanReviewStepVerification_TrimsAndDedupes(t *testing.T) {
	verification := planReviewStepVerification(plan.PlanStep{
		Verification: []string{" go test ./internal/agent ", "go test ./internal/agent", "", "make ci-check"},
	})

	want := []string{"go test ./internal/agent", "make ci-check"}
	if len(verification) != len(want) {
		t.Fatalf("verification = %#v, want %#v", verification, want)
	}
	for i := range want {
		if verification[i] != want[i] {
			t.Fatalf("verification = %#v, want %#v", verification, want)
		}
	}
}

func TestPlanReviewStepFiles_CombinesStructuredAndLegacyFiles(t *testing.T) {
	files := planReviewStepFiles(plan.PlanStep{
		TargetFiles: []string{"foo.go", " foo.go "},
		WriteFiles:  []string{"foo_test.go"},
		ReadFiles:   []string{"README.md"},
		Files:       []string{"foo.go", "docs/foo.md", ""},
	})

	want := []string{"foo.go", "foo_test.go", "README.md", "docs/foo.md"}
	if len(files) != len(want) {
		t.Fatalf("files = %#v, want %#v", files, want)
	}
	for i := range want {
		if files[i] != want[i] {
			t.Fatalf("files = %#v, want %#v", files, want)
		}
	}
}
