package review

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateReviewProbePlanAgainstEvidenceRequiresMaterialPathCoverage(t *testing.T) {
	bundle := newProbePlanEvidenceBundleForValidationTest("/tmp/review-evidence")
	plan := newValidReviewProbePlanForTest()
	plan.ImpactSurfaces[0].EvidenceSummary = "Diff touches probe plan validation."

	err := ValidateReviewProbePlanAgainstEvidence(plan, bundle)
	if err == nil {
		t.Fatal("ValidateReviewProbePlanAgainstEvidence() error = nil, want material path coverage error")
	}
	if !strings.Contains(err.Error(), `internal/review/probe_plan_validate.go`) {
		t.Fatalf("ValidateReviewProbePlanAgainstEvidence() error = %q, want changed path", err.Error())
	}
}

func TestValidateReviewProbePlanAgainstEvidenceRequiresInventoryCategoryCoverage(t *testing.T) {
	repoRoot := "/tmp/review-evidence"
	bundle := newProbePlanEvidenceBundleForValidationTest(repoRoot)
	bundle.Inventory.Config = []string{filepath.Join(repoRoot, "config/review.yaml")}
	plan := newValidReviewProbePlanForTest()

	err := ValidateReviewProbePlanAgainstEvidence(plan, bundle)
	if err == nil {
		t.Fatal("ValidateReviewProbePlanAgainstEvidence() error = nil, want config inventory coverage error")
	}
	if !strings.Contains(err.Error(), `config/review.yaml`) {
		t.Fatalf("ValidateReviewProbePlanAgainstEvidence() error = %q, want config path", err.Error())
	}
}

func TestValidateReviewProbePlanAgainstEvidenceRequiresUntrackedCoverage(t *testing.T) {
	repoRoot := "/tmp/review-evidence"
	bundle := newProbePlanEvidenceBundleForValidationTest(repoRoot)
	bundle.UntrackedFiles = []ReviewUntrackedFile{{Path: filepath.Join(repoRoot, "notes/new-case.txt")}}
	bundle.Inventory.Untracked = []string{filepath.Join(repoRoot, "notes/new-case.txt")}
	plan := newValidReviewProbePlanForTest()

	err := ValidateReviewProbePlanAgainstEvidence(plan, bundle)
	if err == nil {
		t.Fatal("ValidateReviewProbePlanAgainstEvidence() error = nil, want untracked coverage error")
	}
	if !strings.Contains(err.Error(), "untracked path") {
		t.Fatalf("ValidateReviewProbePlanAgainstEvidence() error = %q, want untracked path error", err.Error())
	}
}

func TestValidateReviewProbePlanAgainstEvidenceAllowsUntrackedInventoryPathsCoveredByToken(t *testing.T) {
	repoRoot := "/tmp/review-evidence"
	bundle := newProbePlanEvidenceBundleForValidationTest(repoRoot)
	scratchPath := filepath.Join(repoRoot, "scratch.txt")
	configPath := filepath.Join(repoRoot, "config/local.yaml")
	bundle.UntrackedFiles = []ReviewUntrackedFile{
		{Path: scratchPath},
		{Path: configPath},
	}
	bundle.Inventory.Untracked = []string{scratchPath, configPath}
	bundle.Inventory.Production = append(bundle.Inventory.Production, scratchPath)
	bundle.Inventory.Config = []string{configPath}
	plan := newValidReviewProbePlanForTest()
	plan.ImpactSurfaces[0].EvidenceSummary = "Diff touches internal/review/probe_plan_validate.go. Untracked files are present."

	if err := ValidateReviewProbePlanAgainstEvidence(plan, bundle); err != nil {
		t.Fatalf("ValidateReviewProbePlanAgainstEvidence() error = %v, want nil", err)
	}
}

func TestValidateReviewProbePlanAgainstEvidenceRequiresRenameAndDeletedCoverage(t *testing.T) {
	repoRoot := "/tmp/review-evidence"
	tests := []struct {
		name        string
		mutate      func(*ReviewEvidenceBundle)
		errContains string
	}{
		{
			name: "rename old path",
			mutate: func(bundle *ReviewEvidenceBundle) {
				bundle.ChangedFiles[0].OldPath = filepath.Join(repoRoot, "internal/review/old_probe_plan.go")
				bundle.ChangedFiles[0].Status = "R"
			},
			errContains: "internal/review/old_probe_plan.go",
		},
		{
			name: "deleted inventory path",
			mutate: func(bundle *ReviewEvidenceBundle) {
				bundle.Inventory.DeletedFiles = []string{filepath.Join(repoRoot, "internal/review/removed_probe_plan.go")}
			},
			errContains: "internal/review/removed_probe_plan.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bundle := newProbePlanEvidenceBundleForValidationTest(repoRoot)
			tt.mutate(&bundle)
			plan := newValidReviewProbePlanForTest()

			err := ValidateReviewProbePlanAgainstEvidence(plan, bundle)
			if err == nil {
				t.Fatal("ValidateReviewProbePlanAgainstEvidence() error = nil, want path coverage error")
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("ValidateReviewProbePlanAgainstEvidence() error = %q, want %q", err.Error(), tt.errContains)
			}
		})
	}
}

func TestValidateReviewProbePlanAgainstEvidenceRejectsAllCheckedWhenTruncated(t *testing.T) {
	bundle := newProbePlanEvidenceBundleForValidationTest("/tmp/review-evidence")
	bundle.Diffs = []ReviewDiffEvidence{{Source: "unstaged", DiffTruncated: true}}
	plan := newNoProbeReviewProbePlanForTest()

	err := ValidateReviewProbePlanAgainstEvidence(plan, bundle)
	if err == nil {
		t.Fatal("ValidateReviewProbePlanAgainstEvidence() error = nil, want truncation pressure error")
	}
	if !strings.Contains(err.Error(), "all be checked") {
		t.Fatalf("ValidateReviewProbePlanAgainstEvidence() error = %q, want all checked error", err.Error())
	}
}

func TestValidateReviewProbePlanAgainstEvidenceRequiresGenericImpactCoverage(t *testing.T) {
	bundle := newProbePlanEvidenceBundleForValidationTest("/tmp/review-evidence")
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

	err := ValidateReviewProbePlanAgainstEvidence(plan, bundle)
	if err == nil {
		t.Fatal("ValidateReviewProbePlanAgainstEvidence() error = nil, want generic impact coverage error")
	}
	if !strings.Contains(err.Error(), "generic impact candidates role") {
		t.Fatalf("ValidateReviewProbePlanAgainstEvidence() error = %q, want generic impact role error", err.Error())
	}
}

func TestValidateReviewProbePlanAgainstEvidenceAllowsGenericImpactRoleGroupedCoverage(t *testing.T) {
	bundle := newProbePlanEvidenceBundleForValidationTest("/tmp/review-evidence")
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

	if err := ValidateReviewProbePlanAgainstEvidence(plan, bundle); err != nil {
		t.Fatalf("ValidateReviewProbePlanAgainstEvidence() error = %v, want nil", err)
	}
}

func TestValidateReviewProbePlanAgainstEvidenceAllowsGenericImpactCoverageInSurfaceText(t *testing.T) {
	bundle := newProbePlanEvidenceBundleForValidationTest("/tmp/review-evidence")
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

	if err := ValidateReviewProbePlanAgainstEvidence(plan, bundle); err != nil {
		t.Fatalf("ValidateReviewProbePlanAgainstEvidence() error = %v, want nil", err)
	}
}

func TestValidateReviewProbePlanAgainstEvidenceRejectsGenericImpactTokenSubstringCoverage(t *testing.T) {
	bundle := newProbePlanEvidenceBundleForValidationTest("/tmp/review-evidence")
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

	err := ValidateReviewProbePlanAgainstEvidence(plan, bundle)
	if err == nil {
		t.Fatal("ValidateReviewProbePlanAgainstEvidence() error = nil, want generic impact token substring coverage error")
	}
	if !strings.Contains(err.Error(), "generic impact candidates role") {
		t.Fatalf("ValidateReviewProbePlanAgainstEvidence() error = %q, want generic impact role error", err.Error())
	}
}

func TestValidateReviewProbePlanAgainstEvidenceAllowsGenericImpactExactTokenCoverage(t *testing.T) {
	bundle := newProbePlanEvidenceBundleForValidationTest("/tmp/review-evidence")
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

	if err := ValidateReviewProbePlanAgainstEvidence(plan, bundle); err != nil {
		t.Fatalf("ValidateReviewProbePlanAgainstEvidence() error = %v, want nil", err)
	}
}

func TestValidateReviewProbePlanAgainstEvidenceAllowsGenericImpactPathCoverageInSurfaceReason(t *testing.T) {
	bundle := newProbePlanEvidenceBundleForValidationTest("/tmp/review-evidence")
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

	if err := ValidateReviewProbePlanAgainstEvidence(plan, bundle); err != nil {
		t.Fatalf("ValidateReviewProbePlanAgainstEvidence() error = %v, want nil", err)
	}
}

func TestValidateReviewProbePlanAgainstEvidenceRejectsEmptyGenericImpactTokenCoverage(t *testing.T) {
	bundle := newProbePlanEvidenceBundleForValidationTest("/tmp/review-evidence")
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

	err := ValidateReviewProbePlanAgainstEvidence(plan, bundle)
	if err == nil {
		t.Fatal("ValidateReviewProbePlanAgainstEvidence() error = nil, want empty-token generic impact coverage error")
	}
	if !strings.Contains(err.Error(), "generic impact candidates role") {
		t.Fatalf("ValidateReviewProbePlanAgainstEvidence() error = %q, want generic impact role error", err.Error())
	}
}

func TestValidateReviewProbePlanAgainstEvidenceAllowsEmptyGenericImpactTokenPathCoverage(t *testing.T) {
	bundle := newProbePlanEvidenceBundleForValidationTest("/tmp/review-evidence")
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

	if err := ValidateReviewProbePlanAgainstEvidence(plan, bundle); err != nil {
		t.Fatalf("ValidateReviewProbePlanAgainstEvidence() error = %v, want nil", err)
	}
}

func TestValidateReviewProbePlanAgainstEvidenceRejectsAllCheckedNoProbeWhenGenericImpactTruncated(t *testing.T) {
	bundle := newProbePlanEvidenceBundleForValidationTest("/tmp/review-evidence")
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

	err := ValidateReviewProbePlanAgainstEvidence(plan, bundle)
	if err == nil {
		t.Fatal("ValidateReviewProbePlanAgainstEvidence() error = nil, want generic truncation pressure error")
	}
	if !strings.Contains(err.Error(), "generic impact candidates were truncated") {
		t.Fatalf("ValidateReviewProbePlanAgainstEvidence() error = %q, want generic truncation error", err.Error())
	}
}

func TestValidateReviewProbePlanAgainstEvidenceRejectsNoProbeWithoutRelatedEvidence(t *testing.T) {
	bundle := newProbePlanEvidenceBundleForValidationTest("/tmp/review-evidence")
	bundle.RelatedSearchHits = nil
	plan := newNoProbeReviewProbePlanForTest()

	err := ValidateReviewProbePlanAgainstEvidence(plan, bundle)
	if err == nil {
		t.Fatal("ValidateReviewProbePlanAgainstEvidence() error = nil, want related evidence error")
	}
	if !strings.Contains(err.Error(), "requires related context files or related search hits") {
		t.Fatalf("ValidateReviewProbePlanAgainstEvidence() error = %q, want related evidence error", err.Error())
	}
}

func TestValidateReviewProbePlanAgainstEvidenceAcceptsValidFixture(t *testing.T) {
	bundle := newProbePlanEvidenceBundleForValidationTest("/tmp/review-evidence")
	plan := newValidReviewProbePlanForTest()

	if err := ValidateReviewProbePlanAgainstEvidence(plan, bundle); err != nil {
		t.Fatalf("ValidateReviewProbePlanAgainstEvidence() error = %v, want nil", err)
	}
}

func newProbePlanEvidenceBundleForValidationTest(repoRoot string) ReviewEvidenceBundle {
	changedPath := filepath.Join(repoRoot, "internal/review/probe_plan_validate.go")
	return ReviewEvidenceBundle{
		TargetKind: TargetCurrentChanges,
		RepoRoot:   repoRoot,
		CWD:        repoRoot,
		ChangedFiles: []ReviewChangedFile{
			{
				Path:     changedPath,
				Status:   "M",
				Unstaged: true,
			},
		},
		RelatedSearchHits: []ReviewRelatedSearchHit{
			{
				Path:    filepath.Join(repoRoot, "internal/review/probe_plan_test.go"),
				Line:    1,
				Snippet: "ValidateReviewProbePlan",
				Reason:  "focused validation coverage",
			},
		},
		Inventory: ReviewChangeInventory{
			Production: []string{changedPath},
		},
		Limits: DefaultReviewEvidenceLimits(),
	}
}
