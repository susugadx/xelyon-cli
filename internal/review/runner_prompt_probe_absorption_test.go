package review

import (
	"context"
	"strings"
	"testing"

	reviewmodelinput "github.com/susugadx/xelyon-cli/internal/review/modelinput"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	reviewpromptreduction "github.com/susugadx/xelyon-cli/internal/review/promptreduction"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

func TestReviewRunnerSaturationKeepsProbeResultRawUntilReviewRehydrateLedgerExists(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	rawOutput := strings.Repeat("ABSORBED_PROBE_RAW_OUTPUT_SHOULD_NOT_REACH_SATURATION_PROMPT ", 100)
	probeResult := reviewprobe.ReviewProbeResult{
		ID:     "probe-1",
		Mode:   reviewprobe.ReviewProbeHostReadOnly,
		Status: reviewprobe.ReviewProbePassed,
		CommandResults: []reviewprobe.ReviewProbeCommandResult{
			{
				Command: "customtool",
				Args:    []string{"--inspect"},
				Status:  reviewprobe.ReviewProbePassed,
				Output:  rawOutput,
			},
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
		EvidenceBuilder:     evidence,
		ProbeRunner:         probes,
		Model:               model,
		PromptReductionMode: reviewpromptreduction.ReviewPromptReductionModeApply,
	})
	if err != nil {
		t.Fatalf("NewReviewRunner() error = %v, want nil", err)
	}

	if _, err := runner.Run(context.Background(), NewCurrentChangesRequest("")); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	reportPrompt := model.requests[1].Prompt
	if !strings.Contains(reportPrompt, rawOutput) {
		t.Fatalf("report prompt should keep current probe result context before absorption:\n%s", reportPrompt)
	}
	saturationPrompt := model.requests[2].Prompt
	if !strings.Contains(saturationPrompt, rawOutput) {
		t.Fatalf("saturation prompt dropped probe output without review rehydrate ledger:\n%s", saturationPrompt)
	}
	for _, reject := range []string{
		`"absorbed": true`,
		`"raw_artifact_ref": "probe_results.json"`,
	} {
		if strings.Contains(saturationPrompt, reject) {
			t.Fatalf("saturation prompt contains unsafe absorption marker %q:\n%s", reject, saturationPrompt)
		}
	}
	reductionReport := runner.PromptReductionReport()
	if reductionReport.ClassifierCounts["probe_result_absorption_candidate"] != 1 ||
		reductionReport.ReplacedCount != 0 ||
		reductionReport.KeptReasonCounts[reviewProbeRawOutputReasonArtifactMissing] != 1 {
		t.Fatalf("PromptReductionReport() = %#v, want probe result candidate with apply blocked", reductionReport)
	}
	if reductionReport.RawOutputLedgerCount != 0 {
		t.Fatalf("PromptReductionReport() raw output ledgers = %d, want none before review rehydrate ledger exists", reductionReport.RawOutputLedgerCount)
	}
	if reductionReport.FamilyCounts[string(reviewpromptreduction.ReviewPromptReductionFamilyProbeResult)] != 1 ||
		reductionReport.StatusCounts[string(reviewpromptreduction.ReviewPromptReductionItemCandidate)] != 1 {
		t.Fatalf("PromptReductionReport() = %#v, want candidate probe result family/status counts", reductionReport)
	}
	item := findReviewPromptReductionItemForTest(runner, "probe_result:probe-1", ReviewModelPhaseSaturationCheck)
	if item == nil ||
		item.Family != reviewpromptreduction.ReviewPromptReductionFamilyProbeResult ||
		item.Phase != reviewPromptReductionPhase(ReviewModelPhaseSaturationCheck) ||
		item.Status != reviewpromptreduction.ReviewPromptReductionItemCandidate ||
		item.RawArtifactRef != "" ||
		len(item.AbsorbedBy) != 2 ||
		len(item.EvidenceRefs) != 1 ||
		item.EvidenceRefs[0].ProbeID != "probe-1" ||
		item.OriginalBytes <= item.ReplacementBytes {
		t.Fatalf("probe prompt reduction item = %#v, want candidate item with refs and savings", item)
	}
}

func TestReviewRunnerSaturationKeepsProbeCommandRawUntilReviewRehydrateLedgerExists(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	absorbedOutput := strings.Repeat("ABSORBED_COMMAND_RAW_OUTPUT_SHOULD_NOT_REACH_SATURATION_PROMPT ", 120)
	keptOutput := strings.Repeat("UNREFERENCED_COMMAND_RAW_OUTPUT_MUST_STAY_IN_SATURATION_PROMPT ", 120)
	probeResult := reviewprobe.ReviewProbeResult{
		ID:     "probe-1",
		Mode:   reviewprobe.ReviewProbeHostReadOnly,
		Status: reviewprobe.ReviewProbePassed,
		CommandResults: []reviewprobe.ReviewProbeCommandResult{
			{
				Command: "customtool",
				Args:    []string{"--first"},
				Status:  reviewprobe.ReviewProbePassed,
				Output:  absorbedOutput,
			},
			{
				Command: "customtool",
				Args:    []string{"--second"},
				Status:  reviewprobe.ReviewProbePassed,
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
		EvidenceBuilder:     evidence,
		ProbeRunner:         probes,
		Model:               model,
		PromptReductionMode: reviewpromptreduction.ReviewPromptReductionModeApply,
	})
	if err != nil {
		t.Fatalf("NewReviewRunner() error = %v, want nil", err)
	}

	if _, err := runner.Run(context.Background(), NewCurrentChangesRequest("")); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	reportPrompt := model.requests[1].Prompt
	if !strings.Contains(reportPrompt, absorbedOutput) || !strings.Contains(reportPrompt, keptOutput) {
		t.Fatalf("report prompt should keep current multi-command probe result before absorption:\n%s", reportPrompt)
	}
	saturationPrompt := model.requests[2].Prompt
	if !strings.Contains(saturationPrompt, absorbedOutput) {
		t.Fatalf("saturation prompt dropped referenced command output without review rehydrate ledger:\n%s", saturationPrompt)
	}
	if !strings.Contains(saturationPrompt, keptOutput) {
		t.Fatalf("saturation prompt dropped unreferenced command output:\n%s", saturationPrompt)
	}
	for _, reject := range []string{
		`"absorbed": true`,
		`"raw_artifact_ref": "probe_results.json"`,
	} {
		if strings.Contains(saturationPrompt, reject) {
			t.Fatalf("saturation prompt contains unsafe command absorption marker %q:\n%s", reject, saturationPrompt)
		}
	}
	reductionReport := runner.PromptReductionReport()
	if reductionReport.ClassifierCounts["probe_command_result_absorption_candidate"] != 1 ||
		reductionReport.ReplacedCount != 0 ||
		reductionReport.KeptReasonCounts[reviewProbeRawOutputReasonArtifactMissing] != 1 {
		t.Fatalf("PromptReductionReport() = %#v, want one probe command candidate with apply blocked", reductionReport)
	}
	if reductionReport.RawOutputLedgerCount != 0 {
		t.Fatalf("PromptReductionReport() raw output ledgers = %d, want none before review rehydrate ledger exists", reductionReport.RawOutputLedgerCount)
	}
	item := findReviewPromptReductionItemForTest(runner, "probe_result:probe-1:command:0", ReviewModelPhaseSaturationCheck)
	if item == nil ||
		item.Family != reviewpromptreduction.ReviewPromptReductionFamilyProbeResult ||
		item.Status != reviewpromptreduction.ReviewPromptReductionItemCandidate ||
		item.RawArtifactRef != "" ||
		len(item.EvidenceRefs) != 1 ||
		item.EvidenceRefs[0].Kind != reviewreport.ReviewEvidenceKindProbeCommand ||
		item.EvidenceRefs[0].ProbeID != "probe-1" ||
		item.EvidenceRefs[0].CommandIndex == nil ||
		*item.EvidenceRefs[0].CommandIndex != 0 ||
		item.OriginalBytes <= item.ReplacementBytes {
		t.Fatalf("probe command prompt reduction item = %#v, want command-level candidate item with refs and savings", item)
	}
}

func TestReviewProbeResultAbsorptionKeepsFindingEvidenceProbe(t *testing.T) {
	rawOutput := strings.Repeat("FINDING_PROBE_RAW_OUTPUT_MUST_STAY ", 300)
	report := newRunnerCleanReportWithPassedProbeEvidenceForTest("probe-1")
	report.RootCauseGroups = []reviewreport.ReviewRootCauseGroup{
		{
			ID:                 "group-1",
			Title:              "Finding group",
			Severity:           reviewreport.ReviewGroupSeverityHigh,
			VerificationStatus: reviewreport.ReviewVerificationVerified,
			Findings: []reviewreport.ReviewFinding{
				{
					ID:    "finding-1",
					Title: "Finding uses probe evidence",
					EvidenceRefs: []reviewreport.ReviewEvidenceRef{
						{Kind: reviewreport.ReviewEvidenceKindProbe, ProbeID: "probe-1"},
					},
				},
			},
		},
	}
	result := reviewprobe.ReviewProbeResult{
		ID:     "probe-1",
		Mode:   reviewprobe.ReviewProbeHostReadOnly,
		Status: reviewprobe.ReviewProbePassed,
		CommandResults: []reviewprobe.ReviewProbeCommandResult{
			{Command: "customtool", Status: reviewprobe.ReviewProbePassed, Output: rawOutput},
		},
	}

	candidates := buildReviewProbeResultAbsorptionCandidates(report, []reviewprobe.ReviewProbeResult{result})
	if !candidates.empty() {
		t.Fatalf("buildReviewProbeResultAbsorptionCandidates() = %#v, want finding evidence probe kept", candidates)
	}
}

func TestReviewProbeResultAbsorptionKeepsFindingEvidenceCommandButAbsorbsSafeSibling(t *testing.T) {
	ref0 := reviewreport.ReviewEvidenceRef{
		Kind:         reviewreport.ReviewEvidenceKindProbeCommand,
		ProbeID:      "probe-1",
		CommandIndex: reviewreport.ReviewCommandIndex(0),
	}
	ref1 := reviewreport.ReviewEvidenceRef{
		Kind:         reviewreport.ReviewEvidenceKindProbeCommand,
		ProbeID:      "probe-1",
		CommandIndex: reviewreport.ReviewCommandIndex(1),
	}
	report := newRunnerCleanReportForTest(nil)
	report.ScopeCoverage.ReviewedImpactSurfaces[0].EvidenceRefs = []reviewreport.ReviewEvidenceRef{ref0, ref1}
	report.ScopeCoverage.ReviewedCandidateRisks[0].EvidenceRefs = []reviewreport.ReviewEvidenceRef{ref0, ref1}
	report.RootCauseGroups = []reviewreport.ReviewRootCauseGroup{
		{
			ID:                 "group-1",
			Title:              "Finding group",
			Severity:           reviewreport.ReviewGroupSeverityHigh,
			VerificationStatus: reviewreport.ReviewVerificationVerified,
			Findings: []reviewreport.ReviewFinding{
				{
					ID:           "finding-1",
					Title:        "Finding uses command[0] evidence",
					EvidenceRefs: []reviewreport.ReviewEvidenceRef{ref0},
				},
			},
		},
	}
	result := reviewprobe.ReviewProbeResult{
		ID:     "probe-1",
		Mode:   reviewprobe.ReviewProbeHostReadOnly,
		Status: reviewprobe.ReviewProbePassed,
		CommandResults: []reviewprobe.ReviewProbeCommandResult{
			{Command: "customtool", Args: []string{"--first"}, Status: reviewprobe.ReviewProbePassed, Output: strings.Repeat("finding command output ", 120)},
			{Command: "customtool", Args: []string{"--second"}, Status: reviewprobe.ReviewProbePassed, Output: strings.Repeat("safe sibling command output ", 120)},
		},
	}

	candidates := buildReviewProbeResultAbsorptionCandidates(report, []reviewprobe.ReviewProbeResult{result})
	if len(candidates.probes) != 0 {
		t.Fatalf("probe candidates = %#v, want no full-probe absorption when one command is finding evidence", candidates.probes)
	}
	if _, ok := candidates.commands[reviewmodelinput.ProbeCommandResultKey{ProbeID: "probe-1", CommandIndex: 0}]; ok {
		t.Fatalf("command[0] candidate exists, want finding evidence command kept: %#v", candidates.commands)
	}
	if _, ok := candidates.commands[reviewmodelinput.ProbeCommandResultKey{ProbeID: "probe-1", CommandIndex: 1}]; !ok {
		t.Fatalf("command[1] candidate missing, want safe sibling absorbed: %#v", candidates.commands)
	}
}
