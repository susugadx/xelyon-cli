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

const providerHistoryStatusSummaryFixture = "mode=apply; candidates=3; replaced=2; kept=1; original=1,000 B; projected=250 B; saved=750 B"

func TestProviderHistoryProjectionReportStatusSummaryFormat(t *testing.T) {
	report := providerHistoryStatusTestReport()

	got := formatProviderHistoryProjectionReportSummary(report)
	if got != providerHistoryStatusSummaryFixture {
		t.Fatalf("formatProviderHistoryProjectionReportSummary() = %q, want %q", got, providerHistoryStatusSummaryFixture)
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
		{name: "disabled with bytes", report: ProviderHistoryProjectionReport{Mode: ProviderHistoryReductionDisabled, OriginalBytes: 1}},
		{name: "apply empty report", report: ProviderHistoryProjectionReport{Mode: ProviderHistoryReductionApply}},
		{name: "candidate slice entry", report: ProviderHistoryProjectionReport{
			Candidates: []ProviderHistoryReductionCandidate{{ToolName: "read_file"}},
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
	agent.Runtime.LastProviderHistoryProjectionReport = providerHistoryStatusTestReport()

	output := renderProviderHistoryStatusCommand(t, agent, &out)
	for _, want := range []string{
		"Provider history reduction",
		providerHistoryStatusSummaryFixture,
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
	for _, reject := range []string{"Provider history reduction", "candidates="} {
		if strings.Contains(output, reject) {
			t.Fatalf("/tokens output should not contain %q:\n%s", reject, output)
		}
	}
}

func providerHistoryStatusTestReport() ProviderHistoryProjectionReport {
	return ProviderHistoryProjectionReport{
		Mode:                ProviderHistoryReductionApply,
		CandidateCount:      3,
		ReplacedCount:       2,
		KeptCount:           1,
		OriginalBytes:       1000,
		ProjectedBytes:      250,
		EstimatedSavedBytes: 750,
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
