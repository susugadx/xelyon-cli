package probe

import (
	"reflect"
	"testing"
	"time"
)

func TestBuildReviewProbeRequestsFromPlanConvertsValidPlan(t *testing.T) {
	plan := newValidReviewProbePlanForTest()

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
	if got, want := req.Mode, ReviewProbeRepoSandbox; got != want {
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
	plan := newNoProbeReviewProbePlanForTest()

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
	plan := newValidReviewProbePlanForTest()

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
	data := mustMarshalReviewProbePlanForTest(t, newValidReviewProbePlanForTest())

	firstPlan, err := DecodeReviewProbePlanJSON(data)
	if err != nil {
		t.Fatalf("first DecodeReviewProbePlanJSON() error = %v, want nil", err)
	}
	if err := ValidateReviewProbePlan(firstPlan); err != nil {
		t.Fatalf("first ValidateReviewProbePlan() error = %v, want nil", err)
	}
	firstRequests, err := BuildReviewProbeRequestsFromPlan(firstPlan)
	if err != nil {
		t.Fatalf("first BuildReviewProbeRequestsFromPlan() error = %v, want nil", err)
	}

	secondPlan, err := DecodeReviewProbePlanJSON(data)
	if err != nil {
		t.Fatalf("second DecodeReviewProbePlanJSON() error = %v, want nil", err)
	}
	if err := ValidateReviewProbePlan(secondPlan); err != nil {
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
