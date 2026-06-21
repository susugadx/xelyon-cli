package analysis

import (
	"strings"
	"testing"
)

func TestValidateProbePlanAgainstEvidenceRequiresGenericImpactCoverage(t *testing.T) {
	bundle := newProbePlanEvidenceInputForValidationTest("")
	bundle.GenericImpactCandidates = ReviewGenericImpactCandidates{
		Tokens: []string{"--generic-impact"},
		Candidates: []ReviewGenericImpactCandidate{
			{
				Path:  "docs/commands.md",
				Role:  ReviewGenericImpactRoleDocsReference,
				Token: "--generic-impact",
				Line:  12,
			},
		},
	}
	plan := newValidReviewProbePlanForTest()

	err := validateProbePlanAgainstEvidenceForTest(plan, bundle)
	if err == nil {
		t.Fatal("validateProbePlanAgainstEvidenceForTest() error = nil, want generic impact coverage error")
	}
	if !strings.Contains(err.Error(), "generic impact candidates role") {
		t.Fatalf("validateProbePlanAgainstEvidenceForTest() error = %q, want generic impact role error", err.Error())
	}
}

func TestValidateProbePlanAgainstEvidenceAllowsGenericImpactRoleGroupedCoverage(t *testing.T) {
	bundle := newProbePlanEvidenceInputForValidationTest("")
	bundle.GenericImpactCandidates = ReviewGenericImpactCandidates{
		Tokens: []string{"--generic-impact"},
		Candidates: []ReviewGenericImpactCandidate{
			{
				Path:  "docs/commands.md",
				Role:  ReviewGenericImpactRoleDocsReference,
				Token: "--generic-impact",
				Line:  12,
			},
			{
				Path:  "README.md",
				Role:  ReviewGenericImpactRoleDocsReference,
				Token: "--generic-impact",
				Line:  5,
			},
		},
	}
	plan := newValidReviewProbePlanForTest()
	plan.ImpactSurfaces[0].EvidenceSummary = "Diff touches internal/review/probe_plan_validate.go and covers docs_reference generic impact candidates as docs leads."

	if err := validateProbePlanAgainstEvidenceForTest(plan, bundle); err != nil {
		t.Fatalf("validateProbePlanAgainstEvidenceForTest() error = %v, want nil", err)
	}
}

func TestValidateProbePlanAgainstEvidenceAllowsGenericImpactCoverageInSurfaceText(t *testing.T) {
	bundle := newProbePlanEvidenceInputForValidationTest("")
	bundle.GenericImpactCandidates = ReviewGenericImpactCandidates{
		Tokens: []string{"--generic-impact"},
		Candidates: []ReviewGenericImpactCandidate{
			{
				Path:  "docs/commands.md",
				Role:  ReviewGenericImpactRoleDocsReference,
				Token: "--generic-impact",
				Line:  12,
			},
		},
	}
	plan := newValidReviewProbePlanForTest()
	plan.ImpactSurfaces[0].Summary = "Covers docs_reference generic impact candidates."
	plan.ImpactSurfaces[0].EvidenceSummary = "Diff touches internal/review/probe_plan_validate.go."

	if err := validateProbePlanAgainstEvidenceForTest(plan, bundle); err != nil {
		t.Fatalf("validateProbePlanAgainstEvidenceForTest() error = %v, want nil", err)
	}
}

func TestValidateProbePlanAgainstEvidenceRejectsGenericImpactTokenSubstringCoverage(t *testing.T) {
	bundle := newProbePlanEvidenceInputForValidationTest("")
	bundle.GenericImpactCandidates = ReviewGenericImpactCandidates{
		Tokens: []string{"app"},
		Candidates: []ReviewGenericImpactCandidate{
			{
				Path:  "docs/app.md",
				Role:  ReviewGenericImpactRoleTextualReference,
				Token: "app",
				Line:  3,
			},
		},
	}
	plan := newValidReviewProbePlanForTest()
	plan.ImpactSurfaces[0].Summary = "Application command behavior remains in scope."
	plan.ImpactSurfaces[0].Reason = "Existing evidence covers the changed validator path."

	err := validateProbePlanAgainstEvidenceForTest(plan, bundle)
	if err == nil {
		t.Fatal("validateProbePlanAgainstEvidenceForTest() error = nil, want generic impact token substring coverage error")
	}
	if !strings.Contains(err.Error(), "generic impact candidates role") {
		t.Fatalf("validateProbePlanAgainstEvidenceForTest() error = %q, want generic impact role error", err.Error())
	}
}

func TestValidateProbePlanAgainstEvidenceAllowsGenericImpactExactTokenCoverage(t *testing.T) {
	bundle := newProbePlanEvidenceInputForValidationTest("")
	bundle.GenericImpactCandidates = ReviewGenericImpactCandidates{
		Tokens: []string{"app"},
		Candidates: []ReviewGenericImpactCandidate{
			{
				Path:  "docs/app.md",
				Role:  ReviewGenericImpactRoleTextualReference,
				Token: "app",
				Line:  3,
			},
		},
	}
	plan := newValidReviewProbePlanForTest()
	plan.ImpactSurfaces[0].Summary = "The app token from generic impact candidates remains in scope."

	if err := validateProbePlanAgainstEvidenceForTest(plan, bundle); err != nil {
		t.Fatalf("validateProbePlanAgainstEvidenceForTest() error = %v, want nil", err)
	}
}

func TestValidateProbePlanAgainstEvidenceAllowsGenericImpactPathCoverageInSurfaceReason(t *testing.T) {
	bundle := newProbePlanEvidenceInputForValidationTest("")
	bundle.GenericImpactCandidates = ReviewGenericImpactCandidates{
		Tokens: []string{"--generic-impact"},
		Candidates: []ReviewGenericImpactCandidate{
			{
				Path:  "docs/commands.md",
				Role:  ReviewGenericImpactRoleDocsReference,
				Token: "--generic-impact",
				Line:  12,
			},
		},
	}
	plan := newValidReviewProbePlanForTest()
	plan.ImpactSurfaces[0].Reason = "The generic impact candidate path docs/commands.md is covered as a docs lead."

	if err := validateProbePlanAgainstEvidenceForTest(plan, bundle); err != nil {
		t.Fatalf("validateProbePlanAgainstEvidenceForTest() error = %v, want nil", err)
	}
}

func TestValidateProbePlanAgainstEvidenceRejectsEmptyGenericImpactTokenCoverage(t *testing.T) {
	bundle := newProbePlanEvidenceInputForValidationTest("")
	bundle.GenericImpactCandidates = ReviewGenericImpactCandidates{
		Candidates: []ReviewGenericImpactCandidate{
			{
				Path:  "tests/.gitkeep",
				Role:  ReviewGenericImpactRoleNearbyTestOrTestsDir,
				Token: "",
			},
		},
	}
	plan := newValidReviewProbePlanForTest()

	err := validateProbePlanAgainstEvidenceForTest(plan, bundle)
	if err == nil {
		t.Fatal("validateProbePlanAgainstEvidenceForTest() error = nil, want empty-token generic impact coverage error")
	}
	if !strings.Contains(err.Error(), "generic impact candidates role") {
		t.Fatalf("validateProbePlanAgainstEvidenceForTest() error = %q, want generic impact role error", err.Error())
	}
}

func TestValidateProbePlanAgainstEvidenceAllowsEmptyGenericImpactTokenPathCoverage(t *testing.T) {
	bundle := newProbePlanEvidenceInputForValidationTest("")
	bundle.GenericImpactCandidates = ReviewGenericImpactCandidates{
		Candidates: []ReviewGenericImpactCandidate{
			{
				Path:  "tests/.gitkeep",
				Role:  ReviewGenericImpactRoleNearbyTestOrTestsDir,
				Token: "",
			},
		},
	}
	plan := newValidReviewProbePlanForTest()
	plan.ImpactSurfaces[0].Reason = "The nearby test-dir candidate tests/.gitkeep remains in scope."

	if err := validateProbePlanAgainstEvidenceForTest(plan, bundle); err != nil {
		t.Fatalf("validateProbePlanAgainstEvidenceForTest() error = %v, want nil", err)
	}
}

func TestValidateProbePlanAgainstEvidenceRejectsAllCheckedNoProbeWhenGenericImpactTruncated(t *testing.T) {
	bundle := newProbePlanEvidenceInputForValidationTest("")
	bundle.GenericImpactCandidates = ReviewGenericImpactCandidates{
		Tokens:    []string{"--generic-impact"},
		Truncated: true,
		Candidates: []ReviewGenericImpactCandidate{
			{
				Path:  "docs/commands.md",
				Role:  ReviewGenericImpactRoleDocsReference,
				Token: "--generic-impact",
				Line:  12,
			},
		},
	}
	plan := newNoProbeReviewProbePlanForTest()
	plan.ImpactSurfaces[0].EvidenceSummary = "Diff touches internal/review/probe_plan_validate.go and covers docs_reference generic impact candidates."

	err := validateProbePlanAgainstEvidenceForTest(plan, bundle)
	if err == nil {
		t.Fatal("validateProbePlanAgainstEvidenceForTest() error = nil, want generic truncation pressure error")
	}
	if !strings.Contains(err.Error(), "generic impact candidates were truncated") {
		t.Fatalf("validateProbePlanAgainstEvidenceForTest() error = %q, want generic truncation error", err.Error())
	}
}
