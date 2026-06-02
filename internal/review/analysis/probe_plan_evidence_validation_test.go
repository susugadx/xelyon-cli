package analysis

import (
	"path/filepath"
	"strings"
	"testing"

	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
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

func newProbePlanEvidenceInputForValidationTest(repoRoot string) ReviewEvidenceBundle {
	changedPath := filepath.Join(repoRoot, "internal/review/probe_plan_validate.go")
	return ReviewEvidenceBundle{
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
	}
}

func validateProbePlanAgainstEvidenceForTest(plan reviewprobe.ReviewProbePlan, bundle ReviewEvidenceBundle) error {
	return ValidateProbePlanAgainstEvidence(plan, bundle.toEvidenceInput())
}

type ReviewEvidenceBundle struct {
	ChangedFiles            []ReviewChangedFile
	RelatedSearchHits       []ReviewRelatedSearchHit
	Inventory               ReviewChangeInventory
	UntrackedFiles          []ReviewUntrackedFile
	Diffs                   []ReviewDiffEvidence
	GenericImpactCandidates ReviewGenericImpactCandidates
}

func (b ReviewEvidenceBundle) toEvidenceInput() EvidenceInput {
	return EvidenceInput{
		ChangedFiles:      reviewChangedFilesForAnalysisTest(b.ChangedFiles),
		RelatedSearchHits: reviewRelatedSearchHitsForAnalysisTest(b.RelatedSearchHits),
		ChangeInventory: ChangeInventory{
			Generated:    append([]string(nil), b.Inventory.Generated...),
			Tests:        append([]string(nil), b.Inventory.Tests...),
			Docs:         append([]string(nil), b.Inventory.Docs...),
			Config:       append([]string(nil), b.Inventory.Config...),
			Production:   append([]string(nil), b.Inventory.Production...),
			NewFiles:     append([]string(nil), b.Inventory.NewFiles...),
			DeletedFiles: append([]string(nil), b.Inventory.DeletedFiles...),
			RenamedFiles: append([]string(nil), b.Inventory.RenamedFiles...),
			Untracked:    append([]string(nil), b.Inventory.Untracked...),
		},
		UntrackedFiles: reviewUntrackedFilesForAnalysisTest(b.UntrackedFiles),
		GenericImpact: GenericImpact{
			Tokens:     append([]string(nil), b.GenericImpactCandidates.Tokens...),
			Candidates: reviewGenericImpactCandidatesForAnalysisTest(b.GenericImpactCandidates.Candidates),
			Truncated:  b.GenericImpactCandidates.Truncated,
		},
		TruncationFlags: TruncationFlags{
			Diffs: reviewDiffTruncationsForAnalysisTest(b.Diffs),
		},
	}
}

type ReviewChangedFile struct {
	Path     string
	OldPath  string
	Status   string
	Unstaged bool
}

func reviewChangedFilesForAnalysisTest(files []ReviewChangedFile) []ChangedFile {
	result := make([]ChangedFile, 0, len(files))
	for _, file := range files {
		result = append(result, ChangedFile{
			Path:    file.Path,
			OldPath: file.OldPath,
		})
	}
	return result
}

type ReviewRelatedSearchHit struct {
	Path    string
	Line    int
	Snippet string
	Reason  string
}

func reviewRelatedSearchHitsForAnalysisTest(hits []ReviewRelatedSearchHit) []RelatedSearchHit {
	result := make([]RelatedSearchHit, 0, len(hits))
	for _, hit := range hits {
		result = append(result, RelatedSearchHit{Path: hit.Path})
	}
	return result
}

type ReviewChangeInventory struct {
	Generated    []string
	Tests        []string
	Docs         []string
	Config       []string
	Production   []string
	NewFiles     []string
	DeletedFiles []string
	RenamedFiles []string
	Untracked    []string
}

type ReviewUntrackedFile struct {
	Path string
}

func reviewUntrackedFilesForAnalysisTest(files []ReviewUntrackedFile) []UntrackedFile {
	result := make([]UntrackedFile, 0, len(files))
	for _, file := range files {
		result = append(result, UntrackedFile(file))
	}
	return result
}

type ReviewDiffEvidence struct {
	Source        string
	DiffTruncated bool
}

func reviewDiffTruncationsForAnalysisTest(diffs []ReviewDiffEvidence) []DiffTruncation {
	result := make([]DiffTruncation, 0, len(diffs))
	for _, diff := range diffs {
		result = append(result, DiffTruncation{
			Source: diff.Source,
			Diff:   diff.DiffTruncated,
		})
	}
	return result
}

type ReviewGenericImpactCandidates struct {
	Tokens     []string
	Candidates []ReviewGenericImpactCandidate
	Truncated  bool
}

type ReviewGenericImpactCandidate struct {
	Path  string
	Role  string
	Token string
	Line  int
}

func reviewGenericImpactCandidatesForAnalysisTest(candidates []ReviewGenericImpactCandidate) []GenericImpactCandidate {
	result := make([]GenericImpactCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		result = append(result, GenericImpactCandidate{
			Path:  candidate.Path,
			Role:  candidate.Role,
			Token: candidate.Token,
		})
	}
	return result
}

const (
	ReviewGenericImpactRoleSameStemTestOrSpec   = "same_stem_test_or_spec"
	ReviewGenericImpactRoleNearbyTestOrTestsDir = "nearby_test_or_tests_dir"
	ReviewGenericImpactRoleDocsReference        = "docs_reference"
	ReviewGenericImpactRoleTextualReference     = "textual_reference"
)

func newValidReviewProbePlanForTest() reviewprobe.ReviewProbePlan {
	return reviewprobe.ReviewProbePlan{
		SchemaVersion: reviewprobe.ReviewProbePlanSchemaVersionV2,
		TargetKind:    reviewprobe.TargetCurrentChanges,
		Summary:       "Probe current changes.",
		ImpactSurfaces: []reviewprobe.ReviewProbeImpactSurface{
			{
				ID:              "surface-1",
				Summary:         "Probe plan validation changed.",
				Category:        reviewprobe.ReviewProbeImpactSurfaceValidator,
				EvidenceSummary: "Diff touches internal/review/probe_plan_validate.go.",
				Status:          reviewprobe.ReviewProbeImpactSurfaceNeedsProbe,
				Reason:          "Focused tests should verify the contract.",
			},
		},
		CandidateRisks: []reviewprobe.ReviewProbeCandidateRisk{
			{
				ID:                   "risk-1",
				Summary:              "Validation could accept an invalid probe plan.",
				Severity:             reviewprobe.ReviewGroupSeverityMedium,
				SurfaceIDs:           []string{"surface-1"},
				EvidenceSummary:      "Validation code owns the probe plan contract.",
				VerificationStrategy: "Run focused review tests.",
				Status:               reviewprobe.ReviewProbeCandidateRiskNeedsProbe,
			},
		},
		Probes: []reviewprobe.ReviewPlannedProbe{
			{
				ID:             "probe-1",
				SurfaceIDs:     []string{"surface-1"},
				RiskIDs:        []string{"risk-1"},
				Purpose:        "Confirm or falsify risk-1 for surface-1 by running focused review tests.",
				Mode:           reviewprobe.ReviewProbeRepoSandbox,
				TimeoutSeconds: 30,
				MaxOutputBytes: 4096,
				Commands: []reviewprobe.ReviewPlannedProbeCommand{
					{
						Command: "go",
						Args:    []string{"test", "./internal/review"},
						WorkDir: ".",
					},
				},
			},
		},
	}
}

func newNoProbeReviewProbePlanForTest() reviewprobe.ReviewProbePlan {
	plan := newValidReviewProbePlanForTest()
	plan.ImpactSurfaces[0].Status = reviewprobe.ReviewProbeImpactSurfaceChecked
	plan.ImpactSurfaces[0].Reason = "Existing evidence covers surface-1."
	plan.CandidateRisks[0].Status = reviewprobe.ReviewProbeCandidateRiskCheckedByEvidence
	plan.CandidateRisks[0].VerificationStrategy = "No probe is needed."
	plan.Probes = nil
	plan.NoProbeReason = "surface-1 and risk-1 are checked by existing evidence."
	return plan
}
