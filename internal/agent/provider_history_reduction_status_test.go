package agent

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

const providerHistoryStatusSummaryFixture = "mode=apply; candidates=3; replaced=2; kept=1; original=1,000 B; projected=250 B; saved=750 B; approx_saved_tokens=42; kept_reasons=dry_run:1, missing_evidence_pointer:2; responses_chain_disabled=true"
const providerHistoryCommandEditStatusSummaryFixture = "command/edit: replacement=partial_apply; command_candidates=2; command_replaced=1; edit_arg_candidates=1; command_original_bytes=4,096 B; edit_arg_original_bytes=2,048 B; command_replacement_saved=1,500 B; approx_command_saved_tokens=120; approx_command_replacement_saved_tokens=90; approx_edit_arg_saved_tokens=60; candidate_reasons=command_exit_nonzero:1, git_diff_output:1, write_file_content:1; kept_reasons=latest_tool_result:1, trailing_tool_suffix:2"

func TestProviderHistoryProjectionReportStatusSummaryFormat(t *testing.T) {
	report := providerHistoryStatusTestReport()

	got := formatProviderHistoryProjectionReportSummary(report)
	if got != providerHistoryStatusSummaryFixture {
		t.Fatalf("formatProviderHistoryProjectionReportSummary() = %q, want %q", got, providerHistoryStatusSummaryFixture)
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

func TestProviderHistoryCommandEditDryRunStatusSummaryFormat(t *testing.T) {
	report := providerHistoryStatusTestCommandEditReport()

	got := formatProviderHistoryCommandEditDryRunReportSummary(report)
	if got != providerHistoryCommandEditStatusSummaryFixture {
		t.Fatalf("formatProviderHistoryCommandEditDryRunReportSummary() = %q, want %q", got, providerHistoryCommandEditStatusSummaryFixture)
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
		"mode=apply; no report yet",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("status output missing %q:\n%s", want, output)
		}
	}
}

func TestHandleStatusCommandShowsProviderHistoryReductionDryRunWithoutReport(t *testing.T) {
	var out bytes.Buffer
	agent := newProviderHistoryStatusTestAgent(t, &out)
	agent.Runtime.Options.ProviderHistoryReductionMode = ProviderHistoryReductionDryRun
	agent.Runtime.Options.ProviderHistoryReductionModeSet = true

	output := renderProviderHistoryStatusCommand(t, agent, &out)
	for _, want := range []string{
		"Provider history reduction",
		"mode=dry_run; no report yet",
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
	for _, want := range []string{
		"Provider history reduction",
		providerHistoryStatusSummaryFixture,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("status output missing %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, "command/edit:") {
		t.Fatalf("status output should hide empty command/edit dry-run diagnostics:\n%s", output)
	}
}

func TestHandleStatusCommandShowsProviderHistoryReductionCommandEditDryRunSummary(t *testing.T) {
	var out bytes.Buffer
	agent := newProviderHistoryStatusTestAgent(t, &out)
	report := providerHistoryStatusTestReport()
	report.CommandEditDryRun = providerHistoryStatusTestCommandEditReport()
	agent.Runtime.LastProviderHistoryProjectionReport = report

	output := renderProviderHistoryStatusCommand(t, agent, &out)
	for _, want := range []string{
		"Provider history reduction",
		providerHistoryStatusSummaryFixture,
		providerHistoryCommandEditStatusSummaryFixture,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("status output missing %q:\n%s", want, output)
		}
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
		"mode=auto; effective=dry_run",
		"report: mode=dry_run; candidates=3; replaced=2; kept=1",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("status output missing %q:\n%s", want, output)
		}
	}
}

func TestHandleStatusCommandDoesNotMutateHistoryOrSessionForProviderHistoryReductionDiagnostics(t *testing.T) {
	var out bytes.Buffer
	agent := newProviderHistoryStatusTestAgent(t, &out)
	agent.Runtime.Options.EnableProviderHistoryReduction = true
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
	agent.Runtime.LastProviderHistoryProjectionReport = providerHistoryStatusTestReport()

	if !handleTokensCommand(agent) {
		t.Fatal("handleTokensCommand() = false, want true")
	}

	output := out.String()
	for _, reject := range []string{"Provider history reduction", "command/edit", "candidates=", "approx_saved_tokens", "kept_reasons", "responses_chain_disabled", "command_candidates", "command_replaced", "edit_arg_candidates", "command_replacement_saved", "approx_command_saved_tokens", "approx_command_replacement_saved_tokens", "approx_edit_arg_saved_tokens", "replacement=not_implemented", "replacement=partial_apply"} {
		if strings.Contains(output, reject) {
			t.Fatalf("/tokens output should not contain %q:\n%s", reject, output)
		}
	}
}

func providerHistoryStatusTestReport() ProviderHistoryProjectionReport {
	return ProviderHistoryProjectionReport{
		Mode:                   ProviderHistoryReductionApply,
		CandidateCount:         3,
		ReplacedCount:          2,
		KeptCount:              1,
		OriginalBytes:          1000,
		ProjectedBytes:         250,
		EstimatedSavedBytes:    750,
		ApproxSavedTokens:      42,
		KeptReasonCounts:       map[string]int{"missing_evidence_pointer": 2, "dry_run": 1},
		ResponsesChainDisabled: true,
	}
}

func providerHistoryStatusTestCommandEditReport() ProviderHistoryCommandEditDryRunReport {
	return ProviderHistoryCommandEditDryRunReport{
		ReplacementStatus:                   providerHistoryCommandEditReplacementStatusPartialApply,
		CommandCandidates:                   2,
		EditArgCandidates:                   1,
		CommandOriginalBytes:                4096,
		EditArgOriginalBytes:                2048,
		CommandReplacedCount:                1,
		CommandReplacementSavedBytes:        1500,
		ApproxCommandSavedTokens:            120,
		ApproxCommandReplacementSavedTokens: 90,
		ApproxEditArgSavedTokens:            60,
		CandidateReasonCounts:               map[string]int{"write_file_content": 1, "git_diff_output": 1, "command_exit_nonzero": 1},
		KeptReasonCounts:                    map[string]int{"trailing_tool_suffix": 2, "latest_tool_result": 1},
	}
}

func newProviderHistoryStatusTestAgent(t *testing.T, out *bytes.Buffer) *Agent {
	t.Helper()

	runtime := newIsolatedRuntime()
	runtime.UI = ui.NewRuntime(strings.NewReader(""), out, out)
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
