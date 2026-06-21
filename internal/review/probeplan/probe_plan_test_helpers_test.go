package probeplan

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
	"github.com/susugadx/xelyon-cli/internal/review/report"
)

func newValidReviewProbePlanForTest() ReviewProbePlan {
	return ReviewProbePlan{
		SchemaVersion: ReviewProbePlanSchemaVersionV2,
		TargetKind:    domain.TargetCurrentChanges,
		Summary:       "Probe current changes.",
		ImpactSurfaces: []ReviewProbeImpactSurface{
			{
				ID:              "surface-1",
				Summary:         "Probe plan validation changed.",
				Category:        ReviewProbeImpactSurfaceValidator,
				EvidenceSummary: "Diff touches internal/review/probe_plan_validate.go.",
				Status:          ReviewProbeImpactSurfaceNeedsProbe,
				Reason:          "Focused tests should verify the contract.",
			},
		},
		CandidateRisks: []ReviewProbeCandidateRisk{
			{
				ID:                   "risk-1",
				Summary:              "Validation could accept an invalid probe plan.",
				Severity:             report.ReviewGroupSeverityMedium,
				SurfaceIDs:           []string{"surface-1"},
				EvidenceSummary:      "Validation code owns the probe plan contract.",
				VerificationStrategy: "Run focused review tests.",
				Status:               ReviewProbeCandidateRiskNeedsProbe,
			},
		},
		Probes: []ReviewPlannedProbe{
			{
				ID:             "probe-1",
				SurfaceIDs:     []string{"surface-1"},
				RiskIDs:        []string{"risk-1"},
				Purpose:        "Confirm or falsify risk-1 for surface-1 by running focused review tests.",
				Mode:           domain.ReviewProbeRepoSandbox,
				TimeoutSeconds: 30,
				MaxOutputBytes: 4096,
				Commands: []ReviewPlannedProbeCommand{
					{
						Command: "go",
						Args:    []string{"test", "./internal/review"},
						WorkDir: ".",
					},
					{
						Command: "rg",
						Args:    []string{`path\like`, "ReviewProbePlan"},
						WorkDir: "internal/review",
					},
				},
				Files: []ReviewPlannedProbeFile{
					{
						Path:    "checks/check.txt",
						Content: "ok\n",
					},
				},
			},
		},
	}
}

func newNoProbeReviewProbePlanForTest() ReviewProbePlan {
	return markReviewProbePlanCheckedWithoutProbesForTest(newValidReviewProbePlanForTest())
}

func newNoProbeRisklessReviewProbePlanForTest() ReviewProbePlan {
	plan := markReviewProbePlanCheckedWithoutProbesForTest(newValidReviewProbePlanForTest())
	plan = markReviewProbePlanRisklessForTest(plan)
	plan.NoProbeReason = "surface-1 is checked by existing evidence."
	return plan
}

func markReviewProbePlanRisklessForTest(plan ReviewProbePlan) ReviewProbePlan {
	plan.CandidateRisks = nil
	for i := range plan.Probes {
		plan.Probes[i].RiskIDs = nil
	}
	plan.NoCandidateRiskReason = noCandidateRiskReasonForReviewProbePlanForTest(plan)
	return plan
}

func noCandidateRiskReasonForReviewProbePlanForTest(plan ReviewProbePlan) string {
	surfaceIDs := make([]string, 0, len(plan.ImpactSurfaces))
	for _, surface := range plan.ImpactSurfaces {
		surfaceIDs = append(surfaceIDs, surface.ID)
	}
	return "impact surfaces " + strings.Join(surfaceIDs, ", ") + " were reviewed from available evidence and leave no material candidate risk."
}

func markReviewProbePlanCheckedWithoutProbesForTest(plan ReviewProbePlan) ReviewProbePlan {
	plan.ImpactSurfaces[0].Status = ReviewProbeImpactSurfaceChecked
	plan.ImpactSurfaces[0].Reason = "Existing evidence covers surface-1."
	plan.CandidateRisks[0].Status = ReviewProbeCandidateRiskCheckedByEvidence
	plan.CandidateRisks[0].VerificationStrategy = "No probe is needed."
	plan.Probes = []ReviewPlannedProbe{}
	plan.NoProbeReason = "surface-1 and risk-1 are checked by existing evidence."
	return plan
}

func mustMarshalReviewProbePlanForTest(t *testing.T, plan ReviewProbePlan) []byte {
	t.Helper()

	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v, want nil", err)
	}
	return data
}

func mustMarshalMutatedReviewProbePlanJSONForTest(t *testing.T, mutate func(map[string]any)) string {
	t.Helper()

	var rawPlan map[string]any
	if err := json.Unmarshal(mustMarshalReviewProbePlanForTest(t, newValidReviewProbePlanForTest()), &rawPlan); err != nil {
		t.Fatalf("json.Unmarshal() error = %v, want nil", err)
	}
	mutate(rawPlan)

	data, err := json.Marshal(rawPlan)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v, want nil", err)
	}
	return string(data)
}

func mustFirstReviewProbePlanProbeJSONForTest(t *testing.T, plan map[string]any) map[string]any {
	t.Helper()

	probes, ok := plan["probes"].([]any)
	if !ok {
		t.Fatalf("probes expected array, got %T", plan["probes"])
	}
	if len(probes) == 0 {
		t.Fatal("probes expected at least one entry")
	}
	probe, ok := probes[0].(map[string]any)
	if !ok {
		t.Fatalf("probes[0] expected object, got %T", probes[0])
	}
	return probe
}

func mustFirstReviewProbePlanCommandJSONForTest(t *testing.T, plan map[string]any) map[string]any {
	t.Helper()

	probe := mustFirstReviewProbePlanProbeJSONForTest(t, plan)
	commands, ok := probe["commands"].([]any)
	if !ok {
		t.Fatalf("probes[0].commands expected array, got %T", probe["commands"])
	}
	if len(commands) == 0 {
		t.Fatal("probes[0].commands expected at least one entry")
	}
	command, ok := commands[0].(map[string]any)
	if !ok {
		t.Fatalf("probes[0].commands[0] expected object, got %T", commands[0])
	}
	return command
}

func newReviewPlannedProbeFilesForTest(count, contentBytes int) []ReviewPlannedProbeFile {
	files := make([]ReviewPlannedProbeFile, 0, count)
	for i := 0; i < count; i++ {
		files = append(files, ReviewPlannedProbeFile{
			Path:    "checks/file-" + string(rune('a'+i)) + ".txt",
			Content: strings.Repeat("x", contentBytes),
		})
	}
	return files
}
