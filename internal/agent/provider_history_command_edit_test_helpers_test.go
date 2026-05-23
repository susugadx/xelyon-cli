package agent

import (
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

const (
	providerHistorySuccessfulTestCommand  = "go test ./internal/agent"
	providerHistorySuccessfulBuildCommand = "go build ./cmd/xelyon"
	providerHistorySuccessfulLintCommand  = "npm run lint"

	providerHistorySuccessfulTestReplacementLabel  = "successful test command output"
	providerHistorySuccessfulBuildReplacementLabel = "successful build command output"
	providerHistorySuccessfulLintReplacementLabel  = "successful lint command output"
)

func providerHistoryUnsafeFormattedTestCommand() string {
	return providerHistorySuccessfulTestCommand + "\t\"quoted\" " + strings.Repeat("x", 150)
}

func providerHistoryLargeSuccessfulTestOutput() string {
	return strings.Repeat("ok\tgithub.com/susugadx/xelyon-cli/internal/agent\t0.001s\n", 260)
}

func providerHistoryLargeSuccessfulBuildOutput() string {
	return strings.Repeat("build completed successfully\n", 260)
}

func providerHistoryLargeSuccessfulLintOutput() string {
	return strings.Repeat("lint clean\n", 320)
}

func providerHistoryLargeInterruptedCommandOutput() string {
	return "Command interrupted.\nPartial output:\n" + strings.Repeat("ok\tgithub.com/susugadx/xelyon-cli/internal/agent\t0.001s\n", 260)
}

func providerHistoryLargeCommandOutput(line string) string {
	return strings.Repeat(line, 240)
}

func assertProviderHistoryCommandReplacement(t *testing.T, result providerHistoryProjectionResult, historyIndex int, original, wantLabel string) {
	t.Helper()
	assertProviderHistoryCommandContentReplacement(t, result.History[historyIndex].Content, original, wantLabel)
}

func assertProviderHistoryCommandContentReplacement(t *testing.T, got, original, wantLabel string) {
	t.Helper()
	if got == original || !strings.Contains(got, wantLabel) {
		t.Fatalf("command output content = %q, want replacement containing %q", got, wantLabel)
	}
	if strings.ContainsAny(got, "\n\t\"") {
		t.Fatalf("command output content = %q, want single-line quote-normalized placeholder", got)
	}
}

func assertProviderHistoryCommandProjectionUnchanged(t *testing.T, result providerHistoryProjectionResult, raw []api.Message) {
	t.Helper()
	if !reflect.DeepEqual(result.History, raw) {
		t.Fatalf("apply projection changed command output:\n got %#v\nwant %#v", result.History, raw)
	}
}

func assertProviderHistoryCommandReportNoReplacement(t *testing.T, report ProviderHistoryCommandEditDryRunReport) {
	t.Helper()
	if report.CommandReplacedCount != 0 || report.ReplacementStatus != providerHistoryCommandEditReplacementStatusNotImplemented {
		t.Fatalf("CommandEditDryRun = %#v, want no command replacement", report)
	}
}
