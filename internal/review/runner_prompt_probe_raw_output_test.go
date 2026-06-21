package review

import (
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	reviewpromptreduction "github.com/susugadx/xelyon-cli/internal/review/promptreduction"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

func TestReviewRunnerSaturationAbsorbsProbeCommandWithReviewRawOutputContext(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	absorbedOutput := strings.Repeat("ABSORBED_COMMAND_RAW_OUTPUT_REHYDRATED_IN_CONTEXT ", 120)
	keptOutput := strings.Repeat("UNREFERENCED_COMMAND_RAW_OUTPUT_STAYS_IN_PROBE_CONTEXT ", 120)
	probeResult := reviewprobe.ReviewProbeResult{
		ID:     "probe-1",
		Mode:   domain.ReviewProbeHostReadOnly,
		Status: domain.ReviewProbePassed,
		CommandResults: []reviewprobe.ReviewProbeCommandResult{
			{
				Command: "customtool",
				Args:    []string{"--first"},
				Status:  domain.ReviewProbePassed,
				Output:  absorbedOutput,
			},
			{
				Command: "customtool",
				Args:    []string{"--second"},
				Status:  domain.ReviewProbePassed,
				Output:  keptOutput,
			},
		},
	}
	probes := &runnerFakeProbeRunner{results: map[string]reviewprobe.ReviewProbeResult{"probe-1": probeResult}}
	report := newRunnerCleanReportForTest(nil)
	commandRef := reviewreport.ReviewEvidenceRef{
		Kind:         reviewreport.ReviewEvidenceKindProbeCommand,
		ProbeID:      "probe-1",
		CommandIndex: reviewreport.ReviewCommandIndex(0),
	}
	report.ScopeCoverage.ReviewedImpactSurfaces[0].EvidenceRefs = []reviewreport.ReviewEvidenceRef{commandRef}
	report.ScopeCoverage.ReviewedCandidateRisks[0].EvidenceRefs = []reviewreport.ReviewEvidenceRef{commandRef}
	report.ProbeSummaries = newRedactedRunnerProbeSummariesForTest(t, evidence.bundle, []reviewprobe.ReviewProbeResult{probeResult})
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
		PromptReductionMode:               reviewpromptreduction.ReviewPromptReductionModeApply,
		RawOutputArtifactsMode:            reviewpromptreduction.ReviewRawOutputArtifactsModeApply,
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
		item.Status != reviewpromptreduction.ReviewPromptReductionItemAbsorbed ||
		!strings.HasPrefix(item.RawArtifactRef, "rawout_") {
		t.Fatalf("probe command prompt reduction item = %#v, want applied artifact-backed item", item)
	}
}

func TestReviewRunnerReviewRawOutputArtifactsDryRunDoesNotMutatePrompt(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	rawOutput := strings.Repeat("DRY_RUN_REVIEW_RAW_OUTPUT_MUST_STAY_IN_PROBE_CONTEXT ", 120)
	probeResult := reviewprobe.ReviewProbeResult{
		ID:     "probe-1",
		Mode:   domain.ReviewProbeHostReadOnly,
		Status: domain.ReviewProbePassed,
		CommandResults: []reviewprobe.ReviewProbeCommandResult{
			{Command: "customtool", Args: []string{"--inspect"}, Status: domain.ReviewProbePassed, Output: rawOutput},
		},
	}
	probes := &runnerFakeProbeRunner{results: map[string]reviewprobe.ReviewProbeResult{"probe-1": probeResult}}
	report := newRunnerCleanReportWithPassedProbeEvidenceForTest("probe-1")
	report.ProbeSummaries = newRedactedRunnerProbeSummariesForTest(t, evidence.bundle, []reviewprobe.ReviewProbeResult{probeResult})
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
		PromptReductionMode:               reviewpromptreduction.ReviewPromptReductionModeApply,
		RawOutputArtifactsMode:            reviewpromptreduction.ReviewRawOutputArtifactsModeDryRun,
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
		reductionReport.KeptReasonCounts[reviewpromptreduction.ReviewProbeRawOutputReasonArtifactsDryRun] != 1 {
		t.Fatalf("PromptReductionReport() = %#v, want dry-run ledger without prompt mutation", reductionReport)
	}
}

func TestReviewRunnerKeepsProbeCommandRawWhenReviewRawOutputBudgetCannotFit(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	rawOutput := strings.Repeat("BUDGET_STARVED_REVIEW_RAW_OUTPUT_MUST_STAY_RAW ", 120)
	probeResult := reviewprobe.ReviewProbeResult{
		ID:     "probe-1",
		Mode:   domain.ReviewProbeHostReadOnly,
		Status: domain.ReviewProbePassed,
		CommandResults: []reviewprobe.ReviewProbeCommandResult{
			{Command: "customtool", Args: []string{"--budget"}, Status: domain.ReviewProbePassed, Output: rawOutput},
		},
	}
	probes := &runnerFakeProbeRunner{results: map[string]reviewprobe.ReviewProbeResult{"probe-1": probeResult}}
	commandRef := reviewreport.ReviewEvidenceRef{Kind: reviewreport.ReviewEvidenceKindProbeCommand, ProbeID: "probe-1", CommandIndex: reviewreport.ReviewCommandIndex(0)}
	report := newRunnerCleanReportForTest(nil)
	report.ScopeCoverage.ReviewedImpactSurfaces[0].EvidenceRefs = []reviewreport.ReviewEvidenceRef{commandRef}
	report.ScopeCoverage.ReviewedCandidateRisks[0].EvidenceRefs = []reviewreport.ReviewEvidenceRef{commandRef}
	report.ProbeSummaries = newRedactedRunnerProbeSummariesForTest(t, evidence.bundle, []reviewprobe.ReviewProbeResult{probeResult})
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
		PromptReductionMode:               reviewpromptreduction.ReviewPromptReductionModeApply,
		RawOutputArtifactsMode:            reviewpromptreduction.ReviewRawOutputArtifactsModeApply,
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
		reductionReport.KeptReasonCounts[reviewpromptreduction.ReviewProbeRawOutputReasonRequiredRefBodyTooSmall] < 1 {
		t.Fatalf("PromptReductionReport() = %#v, want raw keep with budget-exhausted ledger", reductionReport)
	}
}

func TestReviewRunnerRevisionPromptRehydratesAbsorbedProbeCommandRef(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	rawOutput := strings.Repeat("REVISION_REHYDRATED_COMMAND_RAW_OUTPUT ", 120)
	probeResult := reviewprobe.ReviewProbeResult{
		ID:     "probe-1",
		Mode:   domain.ReviewProbeHostReadOnly,
		Status: domain.ReviewProbePassed,
		CommandResults: []reviewprobe.ReviewProbeCommandResult{
			{Command: "customtool", Args: []string{"--revision"}, Status: domain.ReviewProbePassed, Output: rawOutput},
		},
	}
	probes := &runnerFakeProbeRunner{results: map[string]reviewprobe.ReviewProbeResult{"probe-1": probeResult}}
	commandRef := reviewreport.ReviewEvidenceRef{Kind: reviewreport.ReviewEvidenceKindProbeCommand, ProbeID: "probe-1", CommandIndex: reviewreport.ReviewCommandIndex(0)}
	initialReport := newRunnerCleanReportForTest(nil)
	initialReport.ScopeCoverage.ReviewedImpactSurfaces[0].EvidenceRefs = []reviewreport.ReviewEvidenceRef{commandRef}
	initialReport.ScopeCoverage.ReviewedCandidateRisks[0].EvidenceRefs = []reviewreport.ReviewEvidenceRef{commandRef}
	initialReport.ProbeSummaries = newRedactedRunnerProbeSummariesForTest(t, evidence.bundle, []reviewprobe.ReviewProbeResult{probeResult})
	revisedReport := initialReport
	needsRevision := needsRevisionAdditionalCandidateCheckForRunnerTest()
	needsRevision.AdditionalFindingCandidates[0].EvidenceRefs = []reviewreport.ReviewEvidenceRef{commandRef}
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
		PromptReductionMode:               reviewpromptreduction.ReviewPromptReductionModeApply,
		RawOutputArtifactsMode:            reviewpromptreduction.ReviewRawOutputArtifactsModeApply,
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
