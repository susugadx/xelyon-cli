package probe

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
	"github.com/susugadx/xelyon-cli/internal/review/probeplan"
	"github.com/susugadx/xelyon-cli/internal/review/report"
)

func TestBuildReviewProbeRequestsFromPlanConvertsValidPlan(t *testing.T) {
	plan := newReviewProbePlanForRequestConversionTest()

	requests, err := BuildReviewProbeRequestsFromPlan(plan)
	if err != nil {
		t.Fatalf("BuildReviewProbeRequestsFromPlan() error = %v, want nil", err)
	}
	if got, want := len(requests), 1; got != want {
		t.Fatalf("len(requests) = %d, want %d", got, want)
	}

	req := requests[0]
	if got, want := req.ID, "probe-1"; got != want {
		t.Fatalf("ID = %q, want %q", got, want)
	}
	if got, want := req.Mode, domain.ReviewProbeRepoSandbox; got != want {
		t.Fatalf("Mode = %q, want %q", got, want)
	}
	if got, want := req.Timeout, 30*time.Second; got != want {
		t.Fatalf("Timeout = %v, want %v", got, want)
	}
	if got, want := req.MaxOutputBytes, int64(4096); got != want {
		t.Fatalf("MaxOutputBytes = %d, want %d", got, want)
	}
	if got, want := len(req.Commands), 2; got != want {
		t.Fatalf("len(Commands) = %d, want %d", got, want)
	}
	if got := req.Commands[0].WorkDir; got != "" {
		t.Fatalf("Commands[0].WorkDir = %q, want default empty workdir", got)
	}
	if got, want := req.Commands[1].WorkDir, "internal/review"; got != want {
		t.Fatalf("Commands[1].WorkDir = %q, want %q", got, want)
	}
	if got, want := req.Commands[1].Args[0], `path\like`; got != want {
		t.Fatalf("Commands[1].Args[0] = %q, want %q", got, want)
	}
	if got, want := req.Files[0].Path, "checks/check.txt"; got != want {
		t.Fatalf("Files[0].Path = %q, want %q", got, want)
	}
}

func TestBuildReviewProbeRequestsFromPlanNoProbePlanReturnsEmptySlice(t *testing.T) {
	plan := newNoProbeReviewProbePlanForRequestConversionTest()

	requests, err := BuildReviewProbeRequestsFromPlan(plan)
	if err != nil {
		t.Fatalf("BuildReviewProbeRequestsFromPlan() error = %v, want nil", err)
	}
	if requests == nil {
		t.Fatal("BuildReviewProbeRequestsFromPlan() requests = nil, want non-nil empty slice")
	}
	if got := len(requests); got != 0 {
		t.Fatalf("len(requests) = %d, want 0", got)
	}
}

func TestBuildReviewProbeRequestsFromPlanCopiesSlices(t *testing.T) {
	plan := newReviewProbePlanForRequestConversionTest()

	requests, err := BuildReviewProbeRequestsFromPlan(plan)
	if err != nil {
		t.Fatalf("BuildReviewProbeRequestsFromPlan() error = %v, want nil", err)
	}

	plan.Probes[0].Commands[0].Args[0] = "changed"
	plan.Probes[0].Files[0].Path = "changed.txt"

	if got, want := requests[0].Commands[0].Args[0], "test"; got != want {
		t.Fatalf("request command args share DTO slice: got %q, want %q", got, want)
	}
	if got, want := requests[0].Files[0].Path, "checks/check.txt"; got != want {
		t.Fatalf("request files share DTO slice: got %q, want %q", got, want)
	}
}

func TestReviewProbePlanDecodeValidateConvertIsDeterministic(t *testing.T) {
	data := mustMarshalReviewProbePlanForRequestConversionTest(t, newReviewProbePlanForRequestConversionTest())

	firstPlan, err := probeplan.DecodeReviewProbePlanJSON(data)
	if err != nil {
		t.Fatalf("first DecodeReviewProbePlanJSON() error = %v, want nil", err)
	}
	if err := probeplan.ValidateReviewProbePlan(firstPlan); err != nil {
		t.Fatalf("first ValidateReviewProbePlan() error = %v, want nil", err)
	}
	firstRequests, err := BuildReviewProbeRequestsFromPlan(firstPlan)
	if err != nil {
		t.Fatalf("first BuildReviewProbeRequestsFromPlan() error = %v, want nil", err)
	}

	secondPlan, err := probeplan.DecodeReviewProbePlanJSON(data)
	if err != nil {
		t.Fatalf("second DecodeReviewProbePlanJSON() error = %v, want nil", err)
	}
	if err := probeplan.ValidateReviewProbePlan(secondPlan); err != nil {
		t.Fatalf("second ValidateReviewProbePlan() error = %v, want nil", err)
	}
	secondRequests, err := BuildReviewProbeRequestsFromPlan(secondPlan)
	if err != nil {
		t.Fatalf("second BuildReviewProbeRequestsFromPlan() error = %v, want nil", err)
	}

	if !reflect.DeepEqual(firstPlan, secondPlan) {
		t.Fatalf("decoded plans differ:\nfirst: %#v\nsecond: %#v", firstPlan, secondPlan)
	}
	if !reflect.DeepEqual(firstRequests, secondRequests) {
		t.Fatalf("converted requests differ:\nfirst: %#v\nsecond: %#v", firstRequests, secondRequests)
	}
}

func newReviewProbePlanForRequestConversionTest() probeplan.ReviewProbePlan {
	return probeplan.ReviewProbePlan{
		SchemaVersion: probeplan.ReviewProbePlanSchemaVersionV2,
		TargetKind:    domain.TargetCurrentChanges,
		Summary:       "Probe plan covers validator behavior.",
		ImpactSurfaces: []probeplan.ReviewProbeImpactSurface{
			{
				ID:              "surface-1",
				Summary:         "validator changed",
				Category:        probeplan.ReviewProbeImpactSurfaceValidator,
				EvidenceSummary: "diff shows validator update",
				Status:          probeplan.ReviewProbeImpactSurfaceNeedsProbe,
				Reason:          "surface needs probe",
			},
		},
		CandidateRisks: []probeplan.ReviewProbeCandidateRisk{
			{
				ID:                   "risk-1",
				Summary:              "edge case may regress",
				Severity:             report.ReviewGroupSeverityMedium,
				SurfaceIDs:           []string{"surface-1"},
				EvidenceSummary:      "related tests mention validator",
				VerificationStrategy: "run focused test",
				Status:               probeplan.ReviewProbeCandidateRiskNeedsProbe,
			},
		},
		Probes: []probeplan.ReviewPlannedProbe{
			{
				ID:             "probe-1",
				SurfaceIDs:     []string{"surface-1"},
				RiskIDs:        []string{"risk-1"},
				Purpose:        "run focused validation coverage",
				Mode:           domain.ReviewProbeRepoSandbox,
				TimeoutSeconds: 30,
				MaxOutputBytes: 4096,
				Commands: []probeplan.ReviewPlannedProbeCommand{
					{Command: "go", Args: []string{"test", "./internal/review/probe"}},
					{Command: "rg", Args: []string{`path\like`, "ReviewProbePlan"}, WorkDir: "internal/review"},
				},
				Files: []probeplan.ReviewPlannedProbeFile{
					{Path: "checks/check.txt", Content: "check"},
				},
			},
		},
	}
}

func newNoProbeReviewProbePlanForRequestConversionTest() probeplan.ReviewProbePlan {
	plan := newReviewProbePlanForRequestConversionTest()
	plan.ImpactSurfaces[0].Status = probeplan.ReviewProbeImpactSurfaceChecked
	plan.CandidateRisks[0].Status = probeplan.ReviewProbeCandidateRiskCheckedByEvidence
	plan.Probes = []probeplan.ReviewPlannedProbe{}
	plan.NoProbeReason = "surface-1 and risk-1 are checked by existing evidence"
	return plan
}

func mustMarshalReviewProbePlanForRequestConversionTest(t *testing.T, plan probeplan.ReviewProbePlan) []byte {
	t.Helper()
	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("json.Marshal(plan) error = %v", err)
	}
	return data
}
