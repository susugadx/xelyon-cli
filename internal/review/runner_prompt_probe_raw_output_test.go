package review

import (
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
)

func TestReviewRunnerSaturationAbsorbsProbeCommandWithReviewRawOutputContext(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	absorbedOutput := strings.Repeat("ABSORBED_COMMAND_RAW_OUTPUT_REHYDRATED_IN_CONTEXT ", 120)
	keptOutput := strings.Repeat("UNREFERENCED_COMMAND_RAW_OUTPUT_STAYS_IN_PROBE_CONTEXT ", 120)
	probeResult := ReviewProbeResult{
		ID:     "probe-1",
		Mode:   ReviewProbeHostReadOnly,
		Status: ReviewProbePassed,
		CommandResults: []ReviewProbeCommandResult{
			{
				Command: "customtool",
				Args:    []string{"--first"},
				Status:  ReviewProbePassed,
				Output:  absorbedOutput,
			},
			{
				Command: "customtool",
				Args:    []string{"--second"},
				Status:  ReviewProbePassed,
				Output:  keptOutput,
			},
		},
	}
	probes := &runnerFakeProbeRunner{results: map[string]ReviewProbeResult{"probe-1": probeResult}}
	report := newRunnerCleanReportForTest(nil)
	commandRef := ReviewEvidenceRef{
		Kind:         ReviewEvidenceKindProbeCommand,
		ProbeID:      "probe-1",
		CommandIndex: ReviewCommandIndex(0),
	}
	report.ScopeCoverage.ReviewedImpactSurfaces[0].EvidenceRefs = []ReviewEvidenceRef{commandRef}
	report.ScopeCoverage.ReviewedCandidateRisks[0].EvidenceRefs = []ReviewEvidenceRef{commandRef}
	report.ProbeSummaries = newRedactedRunnerProbeSummariesForTest(t, evidence.bundle, []ReviewProbeResult{probeResult})
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, newRunnerProbePlanForTest("probe-1")))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, report))},
			saturatedRunnerModelResponseForTest(t),
		},
	}
	runner, err := NewReviewRunner(ReviewRunnerOptions{
		EvidenceBuilder:                   evidence,
		ProbeRunner:                       probes,
		Model:                             model,
		PromptReductionMode:               ReviewPromptReductionModeApply,
		RawOutputArtifactsMode:            ReviewRawOutputArtifactsModeApply,
		RawOutputArtifactStore:            newReviewPromptRawOutputStoreForTest(t),
		RawOutputSessionID:                "session-review-raw-output",
		ReviewRunID:                       "review-run-1",
		RawOutputRehydrateBudgetTokens:    4096,
		RawOutputRehydrateBudgetMaxTokens: 8192,
	})
	if err != nil {
		t.Fatalf("NewReviewRunner() error = %v, want nil", err)
	}

	if _, err := runner.Run(context.Background(), NewCurrentChangesRequest("")); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	saturationPrompt := model.requests[2].Prompt
	for _, want := range []string{
		`"absorbed": true`,
		`"raw_artifact_ref": "rawout_`,
		"## Review Probe Raw Output Context",
		"## Review Probe Raw Output Rehydrate Ledger",
		"ABSORBED_COMMAND_RAW_OUTPUT_REHYDRATED_IN_CONTEXT",
		"UNREFERENCED_COMMAND_RAW_OUTPUT_STAYS_IN_PROBE_CONTEXT",
	} {
		if !strings.Contains(saturationPrompt, want) {
			t.Fatalf("saturation prompt missing %q:\n%s", want, saturationPrompt)
		}
	}
	reductionReport := runner.PromptReductionReport()
	if reductionReport.ReplacedCount != 1 ||
		reductionReport.RawOutputLedgerCount != 1 ||
		reductionReport.RawOutputRequiredRefCount != 1 ||
		reductionReport.RawOutputRehydratedRefCount != 1 ||
		reductionReport.RawOutputMissingRefCount != 0 {
		t.Fatalf("PromptReductionReport() = %#v, want one applied/re-hydrated command ref", reductionReport)
	}
	item := findReviewPromptReductionItemForTest(runner, "probe_result:probe-1:command:0", ReviewModelPhaseSaturationCheck)
	if item == nil ||
		item.Status != ReviewPromptReductionItemAbsorbed ||
		!strings.HasPrefix(item.RawArtifactRef, "rawout_") {
		t.Fatalf("probe command prompt reduction item = %#v, want applied artifact-backed item", item)
	}
}

func TestReviewRunnerReviewRawOutputArtifactsDryRunDoesNotMutatePrompt(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	rawOutput := strings.Repeat("DRY_RUN_REVIEW_RAW_OUTPUT_MUST_STAY_IN_PROBE_CONTEXT ", 120)
	probeResult := ReviewProbeResult{
		ID:     "probe-1",
		Mode:   ReviewProbeHostReadOnly,
		Status: ReviewProbePassed,
		CommandResults: []ReviewProbeCommandResult{
			{Command: "customtool", Args: []string{"--inspect"}, Status: ReviewProbePassed, Output: rawOutput},
		},
	}
	probes := &runnerFakeProbeRunner{results: map[string]ReviewProbeResult{"probe-1": probeResult}}
	report := newRunnerCleanReportWithPassedProbeEvidenceForTest("probe-1")
	report.ProbeSummaries = newRedactedRunnerProbeSummariesForTest(t, evidence.bundle, []ReviewProbeResult{probeResult})
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, newRunnerProbePlanForTest("probe-1")))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, report))},
			saturatedRunnerModelResponseForTest(t),
		},
	}
	runner, err := NewReviewRunner(ReviewRunnerOptions{
		EvidenceBuilder:                   evidence,
		ProbeRunner:                       probes,
		Model:                             model,
		PromptReductionMode:               ReviewPromptReductionModeApply,
		RawOutputArtifactsMode:            ReviewRawOutputArtifactsModeDryRun,
		RawOutputArtifactStore:            newReviewPromptRawOutputStoreForTest(t),
		RawOutputSessionID:                "session-review-raw-output-dry-run",
		ReviewRunID:                       "review-run-dry-run",
		RawOutputRehydrateBudgetTokens:    4096,
		RawOutputRehydrateBudgetMaxTokens: 8192,
	})
	if err != nil {
		t.Fatalf("NewReviewRunner() error = %v, want nil", err)
	}

	if _, err := runner.Run(context.Background(), NewCurrentChangesRequest("")); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	saturationPrompt := model.requests[2].Prompt
	if !strings.Contains(saturationPrompt, rawOutput) {
		t.Fatalf("dry-run saturation prompt dropped raw probe output:\n%s", saturationPrompt)
	}
	for _, reject := range []string{`"absorbed": true`, "## Review Probe Raw Output Context", "## Review Probe Raw Output Rehydrate Ledger"} {
		if strings.Contains(saturationPrompt, reject) {
			t.Fatalf("dry-run saturation prompt changed with %q:\n%s", reject, saturationPrompt)
		}
	}
	reductionReport := runner.PromptReductionReport()
	if reductionReport.ReplacedCount != 0 ||
		reductionReport.RawOutputLedgerCount != 1 ||
		reductionReport.RawOutputRehydratedRefCount != 1 ||
		reductionReport.KeptReasonCounts[reviewProbeRawOutputReasonArtifactsDryRun] != 1 {
		t.Fatalf("PromptReductionReport() = %#v, want dry-run ledger without prompt mutation", reductionReport)
	}
}

func TestReviewRunnerKeepsProbeCommandRawWhenReviewRawOutputBudgetCannotFit(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	rawOutput := strings.Repeat("BUDGET_STARVED_REVIEW_RAW_OUTPUT_MUST_STAY_RAW ", 120)
	probeResult := ReviewProbeResult{
		ID:     "probe-1",
		Mode:   ReviewProbeHostReadOnly,
		Status: ReviewProbePassed,
		CommandResults: []ReviewProbeCommandResult{
			{Command: "customtool", Args: []string{"--budget"}, Status: ReviewProbePassed, Output: rawOutput},
		},
	}
	probes := &runnerFakeProbeRunner{results: map[string]ReviewProbeResult{"probe-1": probeResult}}
	commandRef := ReviewEvidenceRef{Kind: ReviewEvidenceKindProbeCommand, ProbeID: "probe-1", CommandIndex: ReviewCommandIndex(0)}
	report := newRunnerCleanReportForTest(nil)
	report.ScopeCoverage.ReviewedImpactSurfaces[0].EvidenceRefs = []ReviewEvidenceRef{commandRef}
	report.ScopeCoverage.ReviewedCandidateRisks[0].EvidenceRefs = []ReviewEvidenceRef{commandRef}
	report.ProbeSummaries = newRedactedRunnerProbeSummariesForTest(t, evidence.bundle, []ReviewProbeResult{probeResult})
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, newRunnerProbePlanForTest("probe-1")))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, report))},
			saturatedRunnerModelResponseForTest(t),
		},
	}
	runner, err := NewReviewRunner(ReviewRunnerOptions{
		EvidenceBuilder:                   evidence,
		ProbeRunner:                       probes,
		Model:                             model,
		PromptReductionMode:               ReviewPromptReductionModeApply,
		RawOutputArtifactsMode:            ReviewRawOutputArtifactsModeApply,
		RawOutputArtifactStore:            newReviewPromptRawOutputStoreForTest(t),
		RawOutputSessionID:                "session-review-raw-output-budget",
		ReviewRunID:                       "review-run-budget",
		RawOutputRehydrateBudgetTokens:    1,
		RawOutputRehydrateBudgetMaxTokens: 1,
	})
	if err != nil {
		t.Fatalf("NewReviewRunner() error = %v, want nil", err)
	}

	if _, err := runner.Run(context.Background(), NewCurrentChangesRequest("")); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	saturationPrompt := model.requests[2].Prompt
	if !strings.Contains(saturationPrompt, rawOutput) {
		t.Fatalf("budget-starved saturation prompt dropped raw probe output:\n%s", saturationPrompt)
	}
	for _, reject := range []string{`"absorbed": true`, "## Review Probe Raw Output Context", "## Review Probe Raw Output Rehydrate Ledger"} {
		if strings.Contains(saturationPrompt, reject) {
			t.Fatalf("budget-starved saturation prompt changed with %q:\n%s", reject, saturationPrompt)
		}
	}
	reductionReport := runner.PromptReductionReport()
	if reductionReport.ReplacedCount != 0 ||
		reductionReport.RawOutputLedgerCount != 1 ||
		reductionReport.RawOutputBudgetExhaustedRefCount != 1 ||
		reductionReport.KeptReasonCounts[reviewProbeRawOutputReasonRequiredRefBodyTooSmall] < 1 {
		t.Fatalf("PromptReductionReport() = %#v, want raw keep with budget-exhausted ledger", reductionReport)
	}
}

func TestReviewRunnerRevisionPromptRehydratesAbsorbedProbeCommandRef(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	rawOutput := strings.Repeat("REVISION_REHYDRATED_COMMAND_RAW_OUTPUT ", 120)
	probeResult := ReviewProbeResult{
		ID:     "probe-1",
		Mode:   ReviewProbeHostReadOnly,
		Status: ReviewProbePassed,
		CommandResults: []ReviewProbeCommandResult{
			{Command: "customtool", Args: []string{"--revision"}, Status: ReviewProbePassed, Output: rawOutput},
		},
	}
	probes := &runnerFakeProbeRunner{results: map[string]ReviewProbeResult{"probe-1": probeResult}}
	commandRef := ReviewEvidenceRef{Kind: ReviewEvidenceKindProbeCommand, ProbeID: "probe-1", CommandIndex: ReviewCommandIndex(0)}
	initialReport := newRunnerCleanReportForTest(nil)
	initialReport.ScopeCoverage.ReviewedImpactSurfaces[0].EvidenceRefs = []ReviewEvidenceRef{commandRef}
	initialReport.ScopeCoverage.ReviewedCandidateRisks[0].EvidenceRefs = []ReviewEvidenceRef{commandRef}
	initialReport.ProbeSummaries = newRedactedRunnerProbeSummariesForTest(t, evidence.bundle, []ReviewProbeResult{probeResult})
	revisedReport := initialReport
	needsRevision := needsRevisionAdditionalCandidateCheckForRunnerTest()
	needsRevision.AdditionalFindingCandidates[0].EvidenceRefs = []ReviewEvidenceRef{commandRef}
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, newRunnerProbePlanForTest("probe-1")))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, initialReport))},
			{content: string(mustMarshalReviewSaturationCheckForTest(t, needsRevision))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, revisedReport))},
			saturatedRunnerModelResponseForTest(t),
		},
	}
	runner, err := NewReviewRunner(ReviewRunnerOptions{
		EvidenceBuilder:                   evidence,
		ProbeRunner:                       probes,
		Model:                             model,
		PromptReductionMode:               ReviewPromptReductionModeApply,
		RawOutputArtifactsMode:            ReviewRawOutputArtifactsModeApply,
		RawOutputArtifactStore:            newReviewPromptRawOutputStoreForTest(t),
		RawOutputSessionID:                "session-review-raw-output-revision",
		ReviewRunID:                       "review-run-revision",
		RawOutputRehydrateBudgetTokens:    4096,
		RawOutputRehydrateBudgetMaxTokens: 8192,
	})
	if err != nil {
		t.Fatalf("NewReviewRunner() error = %v, want nil", err)
	}

	if _, err := runner.Run(context.Background(), NewCurrentChangesRequest("")); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	revisionPrompt := model.requests[3].Prompt
	for _, want := range []string{
		"## Review Probe Raw Output Context",
		"## Review Probe Raw Output Rehydrate Ledger",
		`"command_index": 0`,
		"REVISION_REHYDRATED_COMMAND_RAW_OUTPUT",
	} {
		if !strings.Contains(revisionPrompt, want) {
			t.Fatalf("revision prompt missing %q:\n%s", want, revisionPrompt)
		}
	}
}

func TestReviewProbeRawOutputCommandDisplayPreservesArgumentBoundaries(t *testing.T) {
	got := reviewProbeRawOutputCommandDisplay(ReviewProbeCommandResult{
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
		command:       ReviewProbeCommandResult{Command: "customtool", Args: []string{"foo bar"}, WorkDir: "/tmp/repo"},
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
		command:      ReviewProbeCommandResult{Command: "customtool", Args: []string{"foo bar"}, WorkDir: "/tmp/repo", Output: "ignored"},
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
	runner := &ReviewRunner{promptReductionStats: newReviewPromptReductionStats(ReviewPromptReductionModeApply)}
	check := newSaturatedReviewSaturationCheckForTest()
	ledger := &ReviewProbeRawOutputLedger{
		FailClosedReason:   reviewProbeRawOutputReasonRequiredRefMissing,
		CanAcceptSaturated: false,
	}

	got := runner.failClosedReviewSaturationByRawOutputLedger(check, ledger)
	if got.Status != ReviewSaturationStatusBlocked ||
		!strings.Contains(got.CheckedSummary, reviewProbeRawOutputReasonRequiredRefMissing) {
		t.Fatalf("failClosedReviewSaturationByRawOutputLedger() = %#v, want blocked with reason", got)
	}
	report := runner.PromptReductionReport()
	if report.KeptReasonCounts[reviewProbeRawOutputReasonSaturatedRejected] != 1 ||
		report.KeptReasonCounts[reviewProbeRawOutputReasonRequiredRefMissing] != 1 {
		t.Fatalf("PromptReductionReport() = %#v, want saturated rejection reasons", report)
	}
}

func newReviewPromptRawOutputStoreForTest(t *testing.T) ReviewRawOutputArtifactStore {
	t.Helper()
	store, err := rawoutputs.OpenStore(rawoutputs.Root(t.TempDir()), rawoutputs.StoreOptions{})
	if err != nil {
		t.Fatalf("rawoutputs.OpenStore() error = %v", err)
	}
	return store
}
