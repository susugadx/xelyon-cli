package agent

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
	reviewpromptreduction "github.com/susugadx/xelyon-cli/internal/review/promptreduction"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

const providerHistoryStatusSummaryFixture = `provider history reduction: apply
replacement_status=partial_apply
content_replacements=2; content_saved=750 B; approx_content_saved_tokens=42
command_output_replacements=0; command_output_saved=0 B; approx_command_output_saved_tokens=0
edit_arg_replacements=0; edit_arg_saved=0 B; approx_edit_arg_saved_tokens=0
total_provider_facing_saved=750 B; approx_total_provider_facing_saved_tokens=42
responses_chain_disabled=true`

const providerHistoryStatusSummaryWithCommandEditFixture = `provider history reduction: apply
replacement_status=partial_apply
content_replacements=2; content_saved=750 B; approx_content_saved_tokens=42
command_output_replacements=1; command_output_saved=1,500 B; approx_command_output_saved_tokens=90
command_output_tools=validation:1
edit_arg_replacements=1; edit_arg_saved=700 B; approx_edit_arg_saved_tokens=55
total_provider_facing_saved=2,950 B; approx_total_provider_facing_saved_tokens=187
responses_chain_disabled=true`

func TestProviderHistoryProjectionReportStatusSummaryFormat(t *testing.T) {
	report := providerHistoryStatusTestReport()

	got := formatProviderHistoryProjectionReportSummary(report)
	if got != providerHistoryStatusSummaryFixture {
		t.Fatalf("formatProviderHistoryProjectionReportSummary() = %q, want %q", got, providerHistoryStatusSummaryFixture)
	}
}

func TestProviderHistoryProjectionReportStatusSummaryIncludesContentToolBreakdown(t *testing.T) {
	report := providerHistoryStatusTestReport()
	report.ContentReplacementToolCounts = map[string]int{"list_dir": 1, "read_file": 2}

	got := formatProviderHistoryProjectionReportSummary(report)
	if !strings.Contains(got, "content_replacement_tools=list_dir:1, read_file:2") {
		t.Fatalf("formatProviderHistoryProjectionReportSummary() = %q, want content tool breakdown", got)
	}
}

func TestProviderHistoryProjectionReportStatusSummaryIncludesRawOutputArtifacts(t *testing.T) {
	report := providerHistoryStatusTestReport()
	report.RawOutputRefCount = 2
	report.RawOutputArtifactCount = 1
	report.DataBearingCandidateCount = 2
	report.ArtifactBackedEstimatedSavedBytes = 8192
	report.ApproxArtifactBackedEstimatedSavedTokens = 2048
	report.ArtifactBackedActualSavedBytes = 4096
	report.ApproxArtifactBackedActualSavedTokens = 1024
	report.CommandEditDryRun.ArtifactBackedCommandCandidates = 2
	report.CommandEditDryRun.ArtifactBackedCommandApplyEligible = 1
	report.CommandEditDryRun.ArtifactBackedCommandReplacedCount = 1
	report.CommandEditDryRun.ArtifactBackedKeptReasonCounts = map[string]int{"raw_output_rehydrate_unsupported": 1}

	got := formatProviderHistoryProjectionReportSummary(report)
	for _, want := range []string{
		"raw_output_refs=2; raw_output_artifacts=1; data_bearing_candidates=2",
		"artifact_backed_command_candidates=2; artifact_backed_command_apply_eligible=1; artifact_backed_command_replacements=1",
		"artifact_backed_estimated_saved=8,192 B; approx_artifact_backed_estimated_saved_tokens=2,048",
		"artifact_backed_saved=4,096 B; approx_artifact_backed_saved_tokens=1,024",
		"artifact_backed_command_kept_reasons=raw_output_rehydrate_unsupported:1",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("formatProviderHistoryProjectionReportSummary() missing %q:\n%s", want, got)
		}
	}
}

func TestProviderHistoryProjectionReportStatusSummaryKeepsFutureFamilyReasonsScoped(t *testing.T) {
	report := providerHistoryStatusTestReport()
	report.FutureFamilyCandidateCounts = map[string]int{"wait_agent": 1}
	report.FutureFamilyKeptReasonCounts = map[string]int{"wait_agent_freeform_output_keep": 1}
	report.KeptReasonCounts = map[string]int{
		"latest_tool_result":              1,
		"wait_agent_freeform_output_keep": 1,
	}

	got := formatProviderHistoryProjectionReportSummary(report)
	if !strings.Contains(got, "future_family_kept_reasons=wait_agent_freeform_output_keep:1") {
		t.Fatalf("formatProviderHistoryProjectionReportSummary() = %q, want scoped future kept reasons", got)
	}
	if strings.Contains(got, "future_family_kept_reasons=latest_tool_result") ||
		strings.Contains(got, "latest_tool_result:1") {
		t.Fatalf("formatProviderHistoryProjectionReportSummary() leaked generic kept reason into future family line:\n%s", got)
	}
}

func TestReviewPromptReductionStatusSummaryFormat(t *testing.T) {
	report := reviewpromptreduction.ReviewPromptReductionReport{
		Mode:                             reviewpromptreduction.ReviewPromptReductionModeApply,
		CandidateCount:                   3,
		ReplacedCount:                    2,
		StateSummaryCount:                2,
		AbsorbedItemCount:                1,
		RawOutputLedgerCount:             1,
		RawOutputRequiredRefCount:        2,
		RawOutputRehydratedRefCount:      1,
		RawOutputMissingRefCount:         1,
		RawOutputBudgetExhaustedRefCount: 1,
		EstimatedSavedBytes:              4096,
		ApproxEstimatedSavedTokens:       512,
		ReplacementSavedBytes:            2048,
		ApproxReplacementSavedTokens:     256,
		ClassifierCounts:                 map[string]int{"git_diff": 1, "validation": 2},
		FamilyCounts:                     map[string]int{"probe_result": 1, "external_doc": 1},
		StatusCounts:                     map[string]int{"absorbed": 2},
		KeptReasonCounts:                 map[string]int{"review_state_summary_current_only": 1},
		QualityFloorPreserved:            true,
	}

	got, ok := reviewPromptReductionStatusSummary(&AgentRuntime{LastReviewPromptReductionReport: report})
	if !ok {
		t.Fatal("reviewPromptReductionStatusSummary() ok = false, want true")
	}
	for _, want := range []string{
		"review prompt reduction: apply",
		"review_history_candidates=3; review_history_replacements=2",
		"review_state_summaries=2; absorbed_intermediate=1; quality_floor=preserved",
		"review_history_estimated_saved=4,096 B; approx_review_history_estimated_saved_tokens=512",
		"review_history_saved=2,048 B; approx_review_history_saved_tokens=256",
		"review_raw_output_ledgers=1; required_refs=2; rehydrated_refs=1; missing_refs=1; budget_exhausted_refs=1",
		"review_history_tools=git_diff:1, validation:2",
		"review_history_families=external_doc:1, probe_result:1",
		"review_history_statuses=absorbed:2",
		"review_history_kept_reasons=review_state_summary_current_only:1",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("summary missing %q:\n%s", want, got)
		}
	}
}

func TestFormatProviderHistoryReasonCountsSortsReasons(t *testing.T) {
	got := formatProviderHistoryReasonCounts(map[string]int{
		"missing_evidence_pointer": 2,
		"dry_run":                  1,
	})
	want := "dry_run:1, missing_evidence_pointer:2"
	if got != want {
		t.Fatalf("formatProviderHistoryReasonCounts() = %q, want %q", got, want)
	}
}

func TestProviderHistoryProjectionReportIsEmptyForDisabledZeroReport(t *testing.T) {
	tests := []struct {
		name   string
		report ProviderHistoryProjectionReport
		want   bool
	}{
		{name: "zero value", report: ProviderHistoryProjectionReport{}, want: true},
		{name: "disabled zero", report: ProviderHistoryProjectionReport{Mode: ProviderHistoryReductionDisabled}, want: true},
		{name: "empty candidate slices", report: ProviderHistoryProjectionReport{
			Candidates: []ProviderHistoryReductionCandidate{},
			Kept:       []ProviderHistoryReductionCandidate{},
		}, want: true},
		{name: "command edit default replacement only", report: ProviderHistoryProjectionReport{
			CommandEditDryRun: newProviderHistoryCommandEditDryRunReport(),
		}, want: true},
		{name: "disabled with bytes", report: ProviderHistoryProjectionReport{Mode: ProviderHistoryReductionDisabled, OriginalBytes: 1}},
		{name: "apply empty report", report: ProviderHistoryProjectionReport{Mode: ProviderHistoryReductionApply}},
		{name: "candidate slice entry", report: ProviderHistoryProjectionReport{
			Candidates: []ProviderHistoryReductionCandidate{{ToolName: "read_file"}},
		}},
		{name: "command edit candidate slice entry", report: ProviderHistoryProjectionReport{
			CommandEditDryRun: ProviderHistoryCommandEditDryRunReport{
				Candidates: []ProviderHistoryCommandEditDryRunCandidate{{ToolName: "bash"}},
			},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := providerHistoryProjectionReportIsEmpty(tt.report); got != tt.want {
				t.Fatalf("providerHistoryProjectionReportIsEmpty(%#v) = %v, want %v", tt.report, got, tt.want)
			}
		})
	}
}

func TestProviderHistoryProjectionModeLabel(t *testing.T) {
	tests := []struct {
		mode ProviderHistoryReductionMode
		want string
	}{
		{ProviderHistoryReductionDisabled, "off"},
		{ProviderHistoryReductionDryRun, "dry_run"},
		{ProviderHistoryReductionApply, "apply"},
		{ProviderHistoryReductionAuto, "auto"},
		{ProviderHistoryReductionMode(99), "unknown"},
	}

	for _, tt := range tests {
		if got := providerHistoryProjectionModeLabel(tt.mode); got != tt.want {
			t.Fatalf("providerHistoryProjectionModeLabel(%d) = %q, want %q", tt.mode, got, tt.want)
		}
	}
}

func TestHandleStatusCommandHidesProviderHistoryReductionWhenDisabledAndNoReport(t *testing.T) {
	var out bytes.Buffer
	agent := newProviderHistoryStatusTestAgent(t, &out)

	output := renderProviderHistoryStatusCommand(t, agent, &out)
	if strings.Contains(output, "Provider history reduction") || strings.Contains(output, "candidates=") {
		t.Fatalf("status output should not contain provider history reduction diagnostics:\n%s", output)
	}
}

func TestHandleStatusCommandShowsProviderHistoryReductionEnabledWithoutReport(t *testing.T) {
	var out bytes.Buffer
	agent := newProviderHistoryStatusTestAgent(t, &out)
	agent.Runtime.Options.EnableProviderHistoryReduction = true

	output := renderProviderHistoryStatusCommand(t, agent, &out)
	for _, want := range []string{
		"Provider history reduction",
		"provider history reduction: apply; no report yet",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("status output missing %q:\n%s", want, output)
		}
	}
}

func TestHandleStatusCommandShowsProviderHistoryReductionDryRunWithoutReport(t *testing.T) {
	var out bytes.Buffer
	agent := newProviderHistoryStatusTestAgent(t, &out)
	applyProviderHistoryStatusActiveContextFixture(agent, activeContextDeepSeek)
	agent.Runtime.Options.ProviderHistoryReductionMode = ProviderHistoryReductionDryRun
	agent.Runtime.Options.ProviderHistoryReductionModeSet = true

	output := renderProviderHistoryStatusCommand(t, agent, &out)
	for _, want := range []string{
		"Provider history reduction",
		"provider history reduction: dry_run; no report yet",
		"rehydrate_context=off; active_context_transport=ephemeral_system_message; active_context_rehydrated_evidence=false; count=0",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("status output missing %q:\n%s", want, output)
		}
	}
}

func TestHandleStatusCommandShowsProviderHistoryRehydrateContextEnabledWithoutReport(t *testing.T) {
	var out bytes.Buffer
	agent := newProviderHistoryStatusTestAgent(t, &out)
	applyProviderHistoryStatusActiveContextFixture(agent, activeContextDeepSeek)
	agent.Runtime.Options.EnableProviderHistoryRehydrateContext = true

	output := renderProviderHistoryStatusCommand(t, agent, &out)
	for _, want := range []string{
		"Provider history reduction",
		"rehydrate_context=on; active_context_transport=ephemeral_system_message; active_context_rehydrated_evidence=false; count=0",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("status output missing %q:\n%s", want, output)
		}
	}
}

func TestHandleStatusCommandShowsProviderHistoryRehydrateContextUnsupportedTransport(t *testing.T) {
	var out bytes.Buffer
	agent := newProviderHistoryStatusTestAgent(t, &out)
	applyProviderHistoryStatusActiveContextFixture(agent, activeContextUnsupported)
	agent.Runtime.Options.EnableProviderHistoryRehydrateContext = true

	output := renderProviderHistoryStatusCommand(t, agent, &out)
	for _, want := range []string{
		"Provider history reduction",
		"rehydrate_context=on; active_context_transport=none; active_context_rehydrated_evidence=false; count=0",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("status output missing %q:\n%s", want, output)
		}
	}
}

func TestHandleStatusCommandShowsProviderHistoryReductionReportSummary(t *testing.T) {
	var out bytes.Buffer
	agent := newProviderHistoryStatusTestAgent(t, &out)
	agent.Runtime.Options.EnableProviderHistoryReduction = false
	report := providerHistoryStatusTestReport()
	report.CommandEditDryRun = newProviderHistoryCommandEditDryRunReport()
	agent.Runtime.LastProviderHistoryProjectionReport = report

	output := renderProviderHistoryStatusCommand(t, agent, &out)
	wants := append([]string{"Provider history reduction"}, providerHistoryStatusSummaryLines(providerHistoryStatusSummaryFixture)...)
	for _, want := range wants {
		if !strings.Contains(output, want) {
			t.Fatalf("status output missing %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, "command/edit:") {
		t.Fatalf("status output should hide empty command/edit dry-run diagnostics:\n%s", output)
	}
}

func TestHandleStatusCommandShowsReviewPromptReductionSummary(t *testing.T) {
	var out bytes.Buffer
	agent := newProviderHistoryStatusTestAgent(t, &out)
	agent.Runtime.LastReviewPromptReductionReport = reviewpromptreduction.ReviewPromptReductionReport{
		Mode:                       reviewpromptreduction.ReviewPromptReductionModeDryRun,
		CandidateCount:             1,
		EstimatedSavedBytes:        1500,
		ApproxEstimatedSavedTokens: 90,
		ClassifierCounts:           map[string]int{"validation": 1},
	}

	output := renderProviderHistoryStatusCommand(t, agent, &out)
	for _, want := range []string{
		"Provider history reduction",
		"review prompt reduction: dry_run",
		"review_history_candidates=1; review_history_replacements=0",
		"review_history_estimated_saved=1,500 B; approx_review_history_estimated_saved_tokens=90",
		"review_history_saved=0 B; approx_review_history_saved_tokens=0",
		"review_history_tools=validation:1",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("status output missing %q:\n%s", want, output)
		}
	}
}

func TestHandleStatusCommandShowsProviderHistoryReductionCommandEditDryRunSummary(t *testing.T) {
	var out bytes.Buffer
	agent := newProviderHistoryStatusTestAgent(t, &out)
	report := providerHistoryStatusTestReportWithCommandEdit()
	agent.Runtime.LastProviderHistoryProjectionReport = report

	output := renderProviderHistoryStatusCommand(t, agent, &out)
	wants := append([]string{"Provider history reduction"}, providerHistoryStatusSummaryLines(providerHistoryStatusSummaryWithCommandEditFixture)...)
	for _, want := range wants {
		if !strings.Contains(output, want) {
			t.Fatalf("status output missing %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, "command/edit:") {
		t.Fatalf("status output should use component summary instead of old command/edit diagnostics:\n%s", output)
	}
}

func TestHandleStatusCommandShowsProviderHistoryReductionAutoAndEffectiveReport(t *testing.T) {
	var out bytes.Buffer
	agent := newProviderHistoryStatusTestAgent(t, &out)
	agent.Runtime.Options.ProviderHistoryReductionMode = ProviderHistoryReductionAuto
	agent.Runtime.Options.ProviderHistoryReductionModeSet = true
	report := providerHistoryStatusTestReport()
	report.Mode = ProviderHistoryReductionDryRun
	agent.Runtime.LastProviderHistoryProjectionReport = report

	output := renderProviderHistoryStatusCommand(t, agent, &out)
	for _, want := range []string{
		"Provider history reduction",
		"provider history reduction: auto; effective=dry_run",
		"content_replacements=2; content_saved=750 B",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("status output missing %q:\n%s", want, output)
		}
	}
}

func TestHandleStatusCommandOmitsProviderHistoryRehydrateCandidates(t *testing.T) {
	var out bytes.Buffer
	agent := newProviderHistoryStatusTestAgent(t, &out)
	installProviderHistoryRehydratePlanFixture(t, agent, "src/main.go", 1, 2)

	output := renderProviderHistoryStatusCommand(t, agent, &out)
	if strings.Contains(output, "Rehydrate candidates") {
		t.Fatalf("status output should not contain rehydrate diagnostics:\n%s", output)
	}
}

func TestHandleStatusCommandDoesNotMutateHistoryOrSessionForProviderHistoryReductionDiagnostics(t *testing.T) {
	var out bytes.Buffer
	agent := newProviderHistoryStatusTestAgent(t, &out)
	agent.Runtime.Options.EnableProviderHistoryReduction = true
	agent.Runtime.Options.EnableProviderHistoryRehydrateContext = true
	agent.Runtime.LastProviderHistoryProjectionReport = providerHistoryStatusTestReport()
	agent.History = []api.Message{
		{Role: "user", Content: "inspect the repo"},
		{Role: "assistant", Content: "done"},
	}
	agent.session.AddMessage("user", "inspect the repo", agent.CurrentModel)
	agent.session.AddMessage("assistant", "done", agent.CurrentModel)
	beforeHistory := api.CloneMessages(agent.History)
	beforeSession := append(agent.session.Messages[:0:0], agent.session.Messages...)

	_ = renderProviderHistoryStatusCommand(t, agent, &out)

	if len(agent.History) != len(beforeHistory) || !reflect.DeepEqual(agent.History, beforeHistory) {
		t.Fatalf("Agent.History changed after /status:\n got %#v\nwant %#v", agent.History, beforeHistory)
	}
	if len(agent.session.Messages) != len(beforeSession) || !reflect.DeepEqual(agent.session.Messages, beforeSession) {
		t.Fatalf("session.Messages changed after /status:\n got %#v\nwant %#v", agent.session.Messages, beforeSession)
	}
}

func TestHandleTokensCommandOmitsProviderHistoryReductionDiagnostics(t *testing.T) {
	var out bytes.Buffer
	agent := newProviderHistoryStatusTestAgent(t, &out)
	agent.Runtime.Options.EnableProviderHistoryReduction = true
	installProviderHistoryRehydratePlanFixture(t, agent, "src/main.go", 1, 2)

	if !handleTokensCommand(agent) {
		t.Fatal("handleTokensCommand() = false, want true")
	}

	output := out.String()
	for _, reject := range []string{"Provider history reduction", "Rehydrate candidates", "rehydrate_context", "active_context_transport", "active_context_rehydrated_evidence", "command/edit", "content_replacements", "command_output_replacements", "edit_arg_replacements", "content_saved", "command_output_saved", "edit_arg_saved", "total_provider_facing_saved", "approx_content_saved_tokens", "approx_command_output_saved_tokens", "approx_edit_arg_saved_tokens", "approx_total_provider_facing_saved_tokens", "responses_chain_disabled", "command_candidates", "command_replaced", "edit_arg_candidates", "edit_arg_replaced", "command_replacement_saved", "edit_arg_replacement_saved", "approx_command_saved_tokens", "approx_command_replacement_saved_tokens", "approx_edit_arg_replacement_saved_tokens", "replacement_status=not_implemented", "replacement_status=partial_apply"} {
		if strings.Contains(output, reject) {
			t.Fatalf("/tokens output should not contain %q:\n%s", reject, output)
		}
	}
}

func providerHistoryStatusTestReport() ProviderHistoryProjectionReport {
	return ProviderHistoryProjectionReport{
		Mode:                                ProviderHistoryReductionApply,
		CandidateCount:                      3,
		ReplacedCount:                       2,
		KeptCount:                           1,
		OriginalBytes:                       1000,
		ProjectedBytes:                      250,
		EstimatedSavedBytes:                 750,
		ApproxSavedTokens:                   42,
		ContentReplacementSavedBytes:        750,
		ApproxContentReplacementSavedTokens: 42,
		ReplacementStatus:                   providerHistoryCommandEditReplacementStatusPartialApply,
		KeptReasonCounts:                    map[string]int{"missing_evidence_pointer": 2, "dry_run": 1},
		ResponsesChainDisabled:              true,
	}
}

func providerHistoryStatusTestReportWithCommandEdit() ProviderHistoryProjectionReport {
	report := providerHistoryStatusTestReport()
	report.CommandEditDryRun = providerHistoryStatusTestCommandEditReport()
	report.EstimatedSavedBytes = 2950
	report.ApproxSavedTokens = 187
	return report
}

func providerHistoryStatusTestCommandEditReport() ProviderHistoryCommandEditDryRunReport {
	return ProviderHistoryCommandEditDryRunReport{
		ReplacementStatus:                   providerHistoryCommandEditReplacementStatusPartialApply,
		CommandCandidates:                   2,
		EditArgCandidates:                   1,
		CommandOriginalBytes:                4096,
		EditArgOriginalBytes:                2048,
		CommandReplacedCount:                1,
		EditArgReplacedCount:                1,
		CommandReplacementSavedBytes:        1500,
		EditArgReplacementSavedBytes:        700,
		ApproxCommandSavedTokens:            120,
		ApproxCommandReplacementSavedTokens: 90,
		ApproxEditArgSavedTokens:            60,
		ApproxEditArgReplacementSavedTokens: 55,
		CandidateReasonCounts:               map[string]int{"write_file_content": 1, "git_diff": 1, "unknown_failure": 1},
		CommandReplacementClassifierCounts:  map[string]int{"validation": 1},
		KeptReasonCounts:                    map[string]int{"trailing_tool_suffix": 2, "latest_tool_result": 1},
	}
}

func newProviderHistoryStatusTestAgent(t *testing.T, out *bytes.Buffer) *Agent {
	t.Helper()

	runtime := newIsolatedRuntime()
	runtime.UI = uiruntime.NewRuntime(strings.NewReader(""), out, out)
	agent := NewAgentWithRuntime("gpt-5.4", &mockProvider{name: "openai"}, false, runtime)
	t.Cleanup(agent.Cleanup)
	return agent
}

func renderProviderHistoryStatusCommand(t *testing.T, agent *Agent, out *bytes.Buffer) string {
	t.Helper()

	if !handleStatusCommandForSurface(agent, commandcatalog.CommandSurfaceTUI) {
		t.Fatal("handleStatusCommandForSurface() = false, want true")
	}
	return out.String()
}

func applyProviderHistoryStatusActiveContextFixture(agent *Agent, fixture activeContextProviderFixture) {
	applyActiveContextProviderFixture(agent, fixture)
	agent.CurrentProvider = &mockProvider{name: fixture.providerName}
}
