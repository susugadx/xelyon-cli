package analysis

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateProbePlanAgainstEvidenceRequiresMaterialPathCoverage(t *testing.T) {
	bundle := newProbePlanEvidenceInputForValidationTest("")
	plan := newValidReviewProbePlanForTest()
	plan.ImpactSurfaces[0].EvidenceSummary = "Diff touches probe plan validation."

	err := validateProbePlanAgainstEvidenceForTest(plan, bundle)
	if err == nil {
		t.Fatal("validateProbePlanAgainstEvidenceForTest() error = nil, want material path coverage error")
	}
	if !strings.Contains(err.Error(), `internal/review/probe_plan_validate.go`) {
		t.Fatalf("validateProbePlanAgainstEvidenceForTest() error = %q, want changed path", err.Error())
	}
}

func TestValidateProbePlanAgainstEvidenceRequiresInventoryCategoryCoverage(t *testing.T) {
	repoRoot := ""
	bundle := newProbePlanEvidenceInputForValidationTest(repoRoot)
	bundle.Inventory.Config = []string{filepath.Join(repoRoot, "config/review.yaml")}
	plan := newValidReviewProbePlanForTest()

	err := validateProbePlanAgainstEvidenceForTest(plan, bundle)
	if err == nil {
		t.Fatal("validateProbePlanAgainstEvidenceForTest() error = nil, want config inventory coverage error")
	}
	if !strings.Contains(err.Error(), `config/review.yaml`) {
		t.Fatalf("validateProbePlanAgainstEvidenceForTest() error = %q, want config path", err.Error())
	}
}

func TestValidateProbePlanAgainstEvidenceRequiresUntrackedCoverage(t *testing.T) {
	repoRoot := ""
	bundle := newProbePlanEvidenceInputForValidationTest(repoRoot)
	bundle.UntrackedFiles = []ReviewUntrackedFile{{Path: filepath.Join(repoRoot, "notes/new-case.txt")}}
	bundle.Inventory.Untracked = []string{filepath.Join(repoRoot, "notes/new-case.txt")}
	plan := newValidReviewProbePlanForTest()

	err := validateProbePlanAgainstEvidenceForTest(plan, bundle)
	if err == nil {
		t.Fatal("validateProbePlanAgainstEvidenceForTest() error = nil, want untracked coverage error")
	}
	if !strings.Contains(err.Error(), "untracked path") {
		t.Fatalf("validateProbePlanAgainstEvidenceForTest() error = %q, want untracked path error", err.Error())
	}
}

func TestValidateProbePlanAgainstEvidenceAllowsUntrackedInventoryPathsCoveredByToken(t *testing.T) {
	repoRoot := ""
	bundle := newProbePlanEvidenceInputForValidationTest(repoRoot)
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

	if err := validateProbePlanAgainstEvidenceForTest(plan, bundle); err != nil {
		t.Fatalf("validateProbePlanAgainstEvidenceForTest() error = %v, want nil", err)
	}
}

func TestValidateProbePlanAgainstEvidenceRequiresRenameAndDeletedCoverage(t *testing.T) {
	repoRoot := ""
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
			bundle := newProbePlanEvidenceInputForValidationTest(repoRoot)
			tt.mutate(&bundle)
			plan := newValidReviewProbePlanForTest()

			err := validateProbePlanAgainstEvidenceForTest(plan, bundle)
			if err == nil {
				t.Fatal("validateProbePlanAgainstEvidenceForTest() error = nil, want path coverage error")
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("validateProbePlanAgainstEvidenceForTest() error = %q, want %q", err.Error(), tt.errContains)
			}
		})
	}
}

func TestValidateProbePlanAgainstEvidenceRejectsAllCheckedWhenTruncated(t *testing.T) {
	bundle := newProbePlanEvidenceInputForValidationTest("")
	bundle.Diffs = []ReviewDiffEvidence{{Source: "unstaged", DiffTruncated: true}}
	plan := newNoProbeReviewProbePlanForTest()

	err := validateProbePlanAgainstEvidenceForTest(plan, bundle)
	if err == nil {
		t.Fatal("validateProbePlanAgainstEvidenceForTest() error = nil, want truncation pressure error")
	}
	if !strings.Contains(err.Error(), "all be checked") {
		t.Fatalf("validateProbePlanAgainstEvidenceForTest() error = %q, want all checked error", err.Error())
	}
}

func TestValidateProbePlanAgainstEvidenceRejectsNoProbeWithoutRelatedEvidence(t *testing.T) {
	bundle := newProbePlanEvidenceInputForValidationTest("")
	bundle.RelatedSearchHits = nil
	plan := newNoProbeReviewProbePlanForTest()

	err := validateProbePlanAgainstEvidenceForTest(plan, bundle)
	if err == nil {
		t.Fatal("validateProbePlanAgainstEvidenceForTest() error = nil, want related evidence error")
	}
	if !strings.Contains(err.Error(), "requires related context files or related search hits") {
		t.Fatalf("validateProbePlanAgainstEvidenceForTest() error = %q, want related evidence error", err.Error())
	}
}

func TestValidateProbePlanAgainstEvidenceAcceptsValidFixture(t *testing.T) {
	bundle := newProbePlanEvidenceInputForValidationTest("")
	plan := newValidReviewProbePlanForTest()

	if err := validateProbePlanAgainstEvidenceForTest(plan, bundle); err != nil {
		t.Fatalf("validateProbePlanAgainstEvidenceForTest() error = %v, want nil", err)
	}
}
