package review

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	reviewevidence "github.com/susugadx/xelyon-cli/internal/review/evidence"
	reviewmodeloutput "github.com/susugadx/xelyon-cli/internal/review/modeloutput"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	reviewprobeplan "github.com/susugadx/xelyon-cli/internal/review/probeplan"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

type runnerFakeEvidenceBuilder struct {
	bundle reviewevidence.ReviewEvidenceBundle
	err    error
	calls  int
	events *[]string
}

func (b *runnerFakeEvidenceBuilder) BuildCurrentChanges(context.Context) (reviewevidence.ReviewEvidenceBundle, error) {
	b.calls++
	if b.events != nil {
		*b.events = append(*b.events, "evidence")
	}
	if b.err != nil {
		return reviewevidence.ReviewEvidenceBundle{}, b.err
	}
	return b.bundle, nil
}

type runnerFakeProbeRunner struct {
	results map[string]reviewprobe.ReviewProbeResult
	errors  map[string]error
	calls   []reviewprobe.ReviewProbeRequest
	events  *[]string
}

func (r *runnerFakeProbeRunner) Run(_ context.Context, req reviewprobe.ReviewProbeRequest) (reviewprobe.ReviewProbeResult, error) {
	r.calls = append(r.calls, req)
	if r.events != nil {
		*r.events = append(*r.events, "probe:"+req.ID)
	}
	if err := r.errors[req.ID]; err != nil {
		return reviewprobe.ReviewProbeResult{}, err
	}
	result, ok := r.results[req.ID]
	if !ok {
		return reviewprobe.ReviewProbeResult{
			ID:     req.ID,
			Mode:   req.Mode,
			Status: reviewprobe.ReviewProbePassed,
		}, nil
	}
	if result.ID == "" {
		result.ID = req.ID
	}
	if result.Mode == "" {
		result.Mode = req.Mode
	}
	if result.Status == "" {
		result.Status = reviewprobe.ReviewProbePassed
	}
	return result, nil
}

type runnerFakeModel struct {
	responses []runnerFakeModelResponse
	requests  []ReviewModelRequest
	events    *[]string
}

type runnerFakeModelResponse struct {
	content string
	err     error
}

func (m *runnerFakeModel) CompleteReview(_ context.Context, req ReviewModelRequest) (ReviewModelResponse, error) {
	m.requests = append(m.requests, req)
	if m.events != nil {
		*m.events = append(*m.events, "model:"+string(req.Phase))
	}
	index := len(m.requests) - 1
	if index >= len(m.responses) {
		return ReviewModelResponse{}, errors.New("unexpected review model call")
	}
	response := m.responses[index]
	if response.err != nil {
		return ReviewModelResponse{}, response.err
	}
	return ReviewModelResponse{Content: response.content}, nil
}

func newReviewRunnerForTest(t *testing.T, evidence ReviewEvidenceProvider, probes ReviewProbeExecutor, model ReviewModel) *ReviewRunner {
	t.Helper()

	runner, err := NewReviewRunner(ReviewRunnerOptions{
		EvidenceBuilder: evidence,
		ProbeRunner:     probes,
		Model:           model,
	})
	if err != nil {
		t.Fatalf("NewReviewRunner() error = %v, want nil", err)
	}
	return runner
}

func newRunnerNonNilDependenciesForTest() ReviewRunnerOptions {
	return ReviewRunnerOptions{
		EvidenceBuilder: &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")},
		ProbeRunner:     &runnerFakeProbeRunner{},
		Model:           &runnerFakeModel{},
	}
}

func mustMarshalReviewProbePlanForRunnerTest(t *testing.T, plan reviewprobeplan.ReviewProbePlan) []byte {
	t.Helper()

	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v, want nil", err)
	}
	return data
}

func mustMarshalReviewProbePlanWithMissingNoProbeReasonForRunnerTest(t *testing.T) []byte {
	t.Helper()

	plan := newRunnerNoProbePlanForTest()
	plan.NoProbeReason = ""
	return mustMarshalReviewProbePlanForRunnerTest(t, plan)
}

func mustMarshalReviewReportForRunnerTest(t *testing.T, report reviewreport.ReviewReport) []byte {
	t.Helper()

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v, want nil", err)
	}
	return data
}

func newRedactedRunnerProbeSummariesForTest(t *testing.T, bundle reviewevidence.ReviewEvidenceBundle, results []reviewprobe.ReviewProbeResult) []reviewreport.ReviewProbeSummary {
	t.Helper()

	summaries := reviewprobe.BuildReviewProbeSummaries(results)
	probeIDs := make([]string, 0, len(summaries))
	for _, summary := range summaries {
		probeIDs = append(probeIDs, summary.ProbeID)
	}
	finalized, err := reviewmodeloutput.FinalizeReport(reviewmodeloutput.ReportFinalizationInput{
		Report:                newRunnerBlockedReportForTest(nil),
		Plan:                  newRunnerProbePlanForTest(probeIDs...),
		TrustedProbeSummaries: summaries,
		Redactor:              newReviewRunnerPromptRedactor(bundle, results),
	})
	if err != nil {
		t.Fatalf("FinalizeReport() error = %v, want nil", err)
	}
	return finalized.ProbeSummaries
}

func assertReviewReportDoesNotContainForRunnerTest(t *testing.T, report reviewreport.ReviewReport, leakedValues ...string) {
	t.Helper()

	reportJSON := string(mustMarshalReviewReportForRunnerTest(t, report))
	for _, leaked := range leakedValues {
		if leaked == "" {
			continue
		}
		if strings.Contains(reportJSON, leaked) {
			t.Fatalf("review report leaked %q:\n%s", leaked, reportJSON)
		}
	}
}

func assertStringSliceEqualForRunnerTest(t *testing.T, got, want []string) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("slice = %#v, want %#v", got, want)
	}
}

func assertReviewRunnerRequestPhasesForTest(t *testing.T, requests []ReviewModelRequest, want []ReviewModelPhase) {
	t.Helper()

	got := make([]ReviewModelPhase, 0, len(requests))
	for _, req := range requests {
		got = append(got, req.Phase)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("model request phases = %#v, want %#v", got, want)
	}
}
