package analysis

import (
	"path/filepath"

	reviewprobeplan "github.com/susugadx/xelyon-cli/internal/review/probeplan"
)

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

func validateProbePlanAgainstEvidenceForTest(plan reviewprobeplan.ReviewProbePlan, bundle ReviewEvidenceBundle) error {
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

func newValidReviewProbePlanForTest() reviewprobeplan.ReviewProbePlan {
	return reviewprobeplan.ReviewProbePlan{
		SchemaVersion: reviewprobeplan.ReviewProbePlanSchemaVersionV2,
		TargetKind:    reviewprobeplan.TargetCurrentChanges,
		Summary:       "Probe current changes.",
		ImpactSurfaces: []reviewprobeplan.ReviewProbeImpactSurface{
			{
				ID:              "surface-1",
				Summary:         "Probe plan validation changed.",
				Category:        reviewprobeplan.ReviewProbeImpactSurfaceValidator,
				EvidenceSummary: "Diff touches internal/review/probe_plan_validate.go.",
				Status:          reviewprobeplan.ReviewProbeImpactSurfaceNeedsProbe,
				Reason:          "Focused tests should verify the contract.",
			},
		},
		CandidateRisks: []reviewprobeplan.ReviewProbeCandidateRisk{
			{
				ID:                   "risk-1",
				Summary:              "Validation could accept an invalid probe plan.",
				Severity:             reviewprobeplan.ReviewGroupSeverityMedium,
				SurfaceIDs:           []string{"surface-1"},
				EvidenceSummary:      "Validation code owns the probe plan contract.",
				VerificationStrategy: "Run focused review tests.",
				Status:               reviewprobeplan.ReviewProbeCandidateRiskNeedsProbe,
			},
		},
		Probes: []reviewprobeplan.ReviewPlannedProbe{
			{
				ID:             "probe-1",
				SurfaceIDs:     []string{"surface-1"},
				RiskIDs:        []string{"risk-1"},
				Purpose:        "Confirm or falsify risk-1 for surface-1 by running focused review tests.",
				Mode:           reviewprobeplan.ReviewProbeRepoSandbox,
				TimeoutSeconds: 30,
				MaxOutputBytes: 4096,
				Commands: []reviewprobeplan.ReviewPlannedProbeCommand{
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

func newNoProbeReviewProbePlanForTest() reviewprobeplan.ReviewProbePlan {
	plan := newValidReviewProbePlanForTest()
	plan.ImpactSurfaces[0].Status = reviewprobeplan.ReviewProbeImpactSurfaceChecked
	plan.ImpactSurfaces[0].Reason = "Existing evidence covers surface-1."
	plan.CandidateRisks[0].Status = reviewprobeplan.ReviewProbeCandidateRiskCheckedByEvidence
	plan.CandidateRisks[0].VerificationStrategy = "No probe is needed."
	plan.Probes = nil
	plan.NoProbeReason = "surface-1 and risk-1 are checked by existing evidence."
	return plan
}
