package review

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestReviewRunnerRunEmitsProgressEvents(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{
		bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo"),
	}
	probeResult := ReviewProbeResult{
		ID:     "probe-1",
		Mode:   ReviewProbeHostReadOnly,
		Status: ReviewProbePassed,
		CommandResults: []ReviewProbeCommandResult{
			{
				Command:  "go",
				Args:     []string{"test", "./internal/review"},
				Status:   ReviewProbePassed,
				Output:   "PASS runner",
				Duration: 1500 * time.Millisecond,
			},
		},
	}
	probes := &runnerFakeProbeRunner{
		results: map[string]ReviewProbeResult{"probe-1": probeResult},
	}
	plan := newRunnerProbePlanForTest("probe-1")
	probeResults := []ReviewProbeResult{probeResult}
	modelReport := newRunnerCleanReportWithPassedProbeEvidenceForTest("probe-1")
	modelReport.ProbeSummaries = newRedactedRunnerProbeSummariesForTest(t, evidence.bundle, probeResults)
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, plan))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, modelReport))},
			saturatedRunnerModelResponseForTest(t),
		},
	}

	var events []ReviewProgressEvent
	runner, err := NewReviewRunner(ReviewRunnerOptions{
		EvidenceBuilder: evidence,
		ProbeRunner:     probes,
		Model:           model,
		ProgressSink: func(event ReviewProgressEvent) {
			events = append(events, event)
		},
	})
	if err != nil {
		t.Fatalf("NewReviewRunner() error = %v, want nil", err)
	}

	if _, err := runner.Run(context.Background(), NewCurrentChangesRequest("")); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	assertReviewProgressSubsequence(t, events, []ReviewProgressEvent{
		{ID: "evidence", Phase: ReviewProgressPhaseEvidence, Status: ReviewProgressRunning, Label: "collecting current changes"},
		{ID: "evidence", Phase: ReviewProgressPhaseEvidence, Status: ReviewProgressOK, Label: "evidence collected"},
		{ID: "probe_plan", Phase: ReviewProgressPhaseProbePlan, Status: ReviewProgressRunning, Label: "planning probes"},
		{ID: "probe_plan", Phase: ReviewProgressPhaseProbePlan, Status: ReviewProgressOK, Label: "planned probes"},
		{ID: "probe:probe-1:0", Phase: ReviewProgressPhaseProbe, Status: ReviewProgressRunning, Label: "probe host_readonly"},
		{ID: "probe:probe-1:0", Phase: ReviewProgressPhaseProbe, Status: ReviewProgressOK, Label: "probe host_readonly"},
		{ID: "report", Phase: ReviewProgressPhaseReport, Status: ReviewProgressRunning, Label: "writing report"},
		{ID: "report", Phase: ReviewProgressPhaseReport, Status: ReviewProgressOK, Label: "report drafted"},
		{ID: "saturation_check", Phase: ReviewProgressPhaseSaturationCheck, Status: ReviewProgressRunning, Label: "checking review coverage"},
		{ID: "saturation_check", Phase: ReviewProgressPhaseSaturationCheck, Status: ReviewProgressOK, Label: "coverage checked"},
	})
	if !reviewProgressDetailContains(events, "evidence", ReviewProgressOK, "staged") {
		t.Fatalf("evidence ok progress missing file-count detail: %#v", events)
	}
	if !reviewProgressDetailContains(events, "probe_plan", ReviewProgressOK, "1 checks") {
		t.Fatalf("probe plan ok progress missing check count: %#v", events)
	}
	if !reviewProgressDetailContains(events, "probe:probe-1:0", ReviewProgressOK, "go test ./internal/review") {
		t.Fatalf("probe ok progress missing command detail: %#v", events)
	}
}

func TestReviewRunnerProbeProgressFinalizesBlockedBeforeExecutionUnderStartedID(t *testing.T) {
	req := ReviewProbeRequest{
		ID:   "probe-blocked",
		Mode: ReviewProbeHostReadOnly,
		Commands: []ReviewProbeCommand{
			{Command: "go", Args: []string{"test", "./internal/review"}},
		},
	}
	probes := &runnerFakeProbeRunner{
		results: map[string]ReviewProbeResult{
			"probe-blocked": {
				ID:     "probe-blocked",
				Mode:   ReviewProbeHostReadOnly,
				Status: ReviewProbeBlocked,
				Error:  "command policy blocked before execution",
			},
		},
	}

	var events []ReviewProgressEvent
	runner := &ReviewRunner{
		probeRunner: probes,
		progressSink: func(event ReviewProgressEvent) {
			events = append(events, event)
		},
	}

	results, err := runner.runReviewProbesSequentially(context.Background(), []ReviewProbeRequest{req})
	if err != nil {
		t.Fatalf("runReviewProbesSequentially() error = %v, want nil", err)
	}
	if len(results) != 1 || results[0].Status != ReviewProbeBlocked {
		t.Fatalf("probe results = %#v, want one blocked result", results)
	}

	assertReviewProgressSubsequence(t, events, []ReviewProgressEvent{
		{ID: "probe:probe-blocked:0", Phase: ReviewProgressPhaseProbe, Status: ReviewProgressRunning, Label: "probe host_readonly"},
		{ID: "probe:probe-blocked:0", Phase: ReviewProgressPhaseProbe, Status: ReviewProgressError, Label: "probe host_readonly"},
	})
	if reviewProgressDetailContains(events, "probe:probe-blocked", ReviewProgressError, "command policy blocked") {
		t.Fatalf("blocked zero-command probe finalized under unsuffixed ID; events = %#v", events)
	}
	if !reviewProgressDetailContains(events, "probe:probe-blocked:0", ReviewProgressError, "command policy blocked") {
		t.Fatalf("blocked progress missing error detail under started ID: %#v", events)
	}
}

func assertReviewProgressSubsequence(t *testing.T, events []ReviewProgressEvent, want []ReviewProgressEvent) {
	t.Helper()

	index := 0
	for _, event := range events {
		if index >= len(want) {
			break
		}
		if reviewProgressEventMatches(event, want[index]) {
			index++
		}
	}
	if index != len(want) {
		t.Fatalf("progress events did not contain wanted subsequence\nwant next %#v\ngot %#v", want[index], events)
	}
}

func reviewProgressEventMatches(got ReviewProgressEvent, want ReviewProgressEvent) bool {
	return got.ID == want.ID &&
		got.Phase == want.Phase &&
		got.Status == want.Status &&
		got.Label == want.Label
}

func reviewProgressDetailContains(events []ReviewProgressEvent, id string, status ReviewProgressStatus, fragment string) bool {
	for _, event := range events {
		if event.ID == id && event.Status == status && strings.Contains(event.Detail, fragment) {
			return true
		}
	}
	return false
}
