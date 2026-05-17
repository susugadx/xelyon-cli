package agent

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
)

func TestBuildPlanReviewDisplay_IncludesParsedPlanFiles(t *testing.T) {
	display := buildPlanReviewDisplay(&plan.Plan{
		Summary: "Ship a reviewable plan",
		Steps: []plan.PlanStep{{
			ID:          1,
			Description: "Update plan review UI",
			Tools:       []string{"apply_patch", "go test ./internal/agent"},
			Files:       []string{" internal/agent/plan_request.go ", "internal/agent/plan_request_test.go"},
		}},
	})

	rendered := display.Render()
	for _, want := range []string{
		"Implementation Plan Review",
		"Ship a reviewable plan",
		"Update plan review UI",
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
