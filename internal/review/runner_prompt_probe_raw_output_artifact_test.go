package review

import (
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	reviewpromptreduction "github.com/susugadx/xelyon-cli/internal/review/promptreduction"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

func TestReviewProbeRawOutputCommandDisplayPreservesArgumentBoundaries(t *testing.T) {
	got := reviewProbeRawOutputCommandDisplay(reviewprobe.ReviewProbeCommandResult{
		Command: "customtool",
		Args:    []string{"foo bar", "--flag=a;b", "$(printf token)"},
	})
	want := `customtool "foo bar" "--flag=a;b" "$(printf token)"`
	if got != want {
		t.Fatalf("reviewProbeRawOutputCommandDisplay() = %q, want %q", got, want)
	}
}

func TestReviewProbeRawOutputCommandHashUsesStableCommandIndexValue(t *testing.T) {
	firstIndex := 0
	secondIndex := 0
	otherIndex := 1
	base := reviewProbeRawOutputSource{
		probeID:       "probe-1",
		command:       reviewprobe.ReviewProbeCommandResult{Command: "customtool", Args: []string{"foo bar"}, WorkDir: "/tmp/repo"},
		body:          "stable body",
		originalBytes: 11,
	}
	first := base
	first.commandIndex = &firstIndex
	second := base
	second.commandIndex = &secondIndex
	other := base
	other.commandIndex = &otherIndex
	probeLevel := base

	if got, want := reviewProbeRawOutputCommandHash(first), reviewProbeRawOutputCommandHash(second); got != want {
		t.Fatalf("command hash differs for equal command index values: %s vs %s", got, want)
	}
	if got, reject := reviewProbeRawOutputCommandHash(first), reviewProbeRawOutputCommandHash(other); got == reject {
		t.Fatalf("command hash for command[0] matched command[1]: %s", got)
	}
	if got, reject := reviewProbeRawOutputCommandHash(first), reviewProbeRawOutputCommandHash(probeLevel); got == reject {
		t.Fatalf("command-level hash matched probe-level hash: %s", got)
	}
}

func TestReviewProbeRawOutputCommandArtifactRefIsStableAcrossCommandIndexPointers(t *testing.T) {
	store, err := rawoutputs.OpenStore(rawoutputs.Root(t.TempDir()), rawoutputs.StoreOptions{})
	if err != nil {
		t.Fatalf("rawoutputs.OpenStore() error = %v", err)
	}
	runner := &ReviewRunner{
		rawOutputArtifactStore: store,
		rawOutputSessionID:     "session-stable-probe-command-ref",
		reviewRunID:            "review-run-stable",
	}
	firstIndex := 0
	secondIndex := 0
	source := reviewProbeRawOutputSource{
		probeID:      "probe-1",
		commandIndex: &firstIndex,
		command:      reviewprobe.ReviewProbeCommandResult{Command: "customtool", Args: []string{"foo bar"}, WorkDir: "/tmp/repo", Output: "ignored"},
		body:         strings.Repeat("stable command raw output ", 40),
	}
	secondSource := source
	secondSource.commandIndex = &secondIndex

	firstRef, reason, ok := runner.createReviewProbeRawOutputArtifact(context.Background(), ReviewModelPhaseSaturationCheck, source)
	if !ok {
		t.Fatalf("first createReviewProbeRawOutputArtifact() reason=%q, want ref", reason)
	}
	secondRef, reason, ok := runner.createReviewProbeRawOutputArtifact(context.Background(), ReviewModelPhaseSaturationCheck, secondSource)
	if !ok {
		t.Fatalf("second createReviewProbeRawOutputArtifact() reason=%q, want ref", reason)
	}
	if firstRef.RefID != secondRef.RefID || firstRef.CommandHash != secondRef.CommandHash {
		t.Fatalf("refs differ for equal command index values:\n first=%#v\nsecond=%#v", firstRef, secondRef)
	}
}

func TestReviewRunnerRejectsSaturatedWhenReviewRawOutputLedgerFailsClosed(t *testing.T) {
	runner := &ReviewRunner{promptReductionStats: reviewpromptreduction.NewStats(reviewpromptreduction.ReviewPromptReductionModeApply)}
	check := newSaturatedReviewSaturationCheckForTest()
	ledger := &reviewpromptreduction.ReviewProbeRawOutputLedger{
		FailClosedReason:   reviewpromptreduction.ReviewProbeRawOutputReasonRequiredRefMissing,
		CanAcceptSaturated: false,
	}

	got := runner.failClosedReviewSaturationByRawOutputLedger(check, ledger)
	if got.Status != reviewreport.ReviewSaturationStatusBlocked ||
		!strings.Contains(got.CheckedSummary, reviewpromptreduction.ReviewProbeRawOutputReasonRequiredRefMissing) {
		t.Fatalf("failClosedReviewSaturationByRawOutputLedger() = %#v, want blocked with reason", got)
	}
	report := runner.PromptReductionReport()
	if report.KeptReasonCounts[reviewpromptreduction.ReviewProbeRawOutputReasonSaturatedRejected] != 1 ||
		report.KeptReasonCounts[reviewpromptreduction.ReviewProbeRawOutputReasonRequiredRefMissing] != 1 {
		t.Fatalf("PromptReductionReport() = %#v, want saturated rejection reasons", report)
	}
}

func newReviewPromptRawOutputStoreForTest(t *testing.T) reviewpromptreduction.ReviewRawOutputArtifactStore {
	t.Helper()
	store, err := rawoutputs.OpenStore(rawoutputs.Root(t.TempDir()), rawoutputs.StoreOptions{})
	if err != nil {
		t.Fatalf("rawoutputs.OpenStore() error = %v", err)
	}
	return store
}
