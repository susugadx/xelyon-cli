package providerhistory

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestProjectApplyCompactsDatabaseRowsWithFailureTextAsArtifactBackedData(t *testing.T) {
	commandOutput := strings.Repeat("id | status | note\n1 | failed | Error: stored application error row\n2 | ok | expected fixture value\n", 1600)
	history := []api.Message{
		providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, "call_db", "bash", map[string]string{"command": `sqlite3 app.db "select * from audit_log"`})),
		providerHistoryTestToolResult("call_db", "bash", commandOutput),
		{Role: "assistant", Content: "database rows reviewed"},
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest"),
		{Role: "assistant", Content: "done"},
	}

	result := Project(ProjectionInput{
		Messages: history,
		Policy: Policy{
			Mode:                             Apply,
			RawOutputArtifactsMode:           RawOutputArtifactsApply,
			RawOutputArtifactStore:           providerHistoryTestRawOutputStore(t),
			SessionID:                        "session-database-row-artifact",
			RawOutputRehydrateContextEnabled: true,
			ActiveContextTransportAvailable:  true,
		},
	})

	projected := result.History[1].Content
	if projected == commandOutput ||
		!strings.Contains(projected, "[compacted old data-bearing command output;") ||
		!strings.Contains(projected, "raw_output_ref=") {
		t.Fatalf("projected database output = %q, want artifact-backed data-bearing placeholder", projected)
	}
	for _, reject := range []string{"[compacted old failed command output", "classifier=database_failure", "database_connection_error"} {
		if strings.Contains(projected, reject) {
			t.Fatalf("database row output was mislabeled as failure with %q:\n%s", reject, projected)
		}
	}
	commandReport := result.Report.CommandEditDryRun
	if commandReport.ArtifactBackedCommandCandidates != 1 ||
		commandReport.ArtifactBackedCommandApplyEligible != 1 ||
		commandReport.ArtifactBackedCommandReplacedCount != 1 ||
		commandReport.CommandReplacedCount != 0 ||
		!result.Report.ResponsesChainDisabled {
		t.Fatalf("command report = %#v / top-level %#v, want artifact-backed database data replacement only", commandReport, result.Report)
	}
	candidate := providerHistoryTestCommandCandidateByToolCallID(commandReport, "call_db")
	if candidate == nil ||
		candidate.Classifier != "database_query_result" ||
		!candidate.ArtifactBackedCandidate ||
		candidate.RawOutputRefID == "" ||
		candidate.RehydrateGateStatus != "available" ||
		!candidate.ArtifactBackedApplyEligible ||
		!candidate.ReplacementApplied {
		t.Fatalf("candidate = %#v, want applied artifact-backed database query data candidate", candidate)
	}
	if result.Report.DataBearingCandidateCount != 1 ||
		result.Report.RawOutputRefCount != 1 ||
		result.Report.ArtifactBackedActualSavedBytes != commandReport.ArtifactBackedCommandReplacementSavedBytes {
		t.Fatalf("top-level artifact metrics = %#v, command report = %#v, want one data-bearing database ref and actual savings", result.Report, commandReport)
	}
}

func TestProjectApplyCompactsSideEffectSuccessesWithFinalReport(t *testing.T) {
	tests := []struct {
		name       string
		callID     string
		command    string
		output     string
		wantText   string
		reason     string
		classifier string
	}{
		{
			name:       "database migration success",
			callID:     "call_migrate",
			command:    "npx prisma migrate deploy",
			output:     strings.Repeat("Migration 20260101000000_init applied\nProcess exited with code 0\n", 90),
			wantText:   "successful database operation command output",
			reason:     "database_operation_success",
			classifier: "database_operation",
		},
		{
			name:       "package install success",
			callID:     "call_install",
			command:    "npm install",
			output:     strings.Repeat("added 120 packages\nProcess exited with code 0\n", 90),
			wantText:   "successful side-effect command output",
			reason:     "package_success",
			classifier: "package_install",
		},
		{
			name:       "deploy success",
			callID:     "call_deploy",
			command:    "vercel deploy",
			output:     strings.Repeat("deployment completed successfully\nProcess exited with code 0\n", 90),
			wantText:   "successful side-effect command output",
			reason:     "deploy_success",
			classifier: "deploy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			history := []api.Message{
				providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, tt.callID, "bash", map[string]string{"command": tt.command})),
				providerHistoryTestToolResult(tt.callID, "bash", tt.output),
				{Role: "assistant", Content: "side effect completed"},
				providerHistoryTestAssistantToolCall("call_latest", "read_file"),
				providerHistoryTestToolResult("call_latest", "read_file", "latest"),
				{Role: "assistant", Content: "done"},
			}

			result := Project(ProjectionInput{Messages: history, Policy: Policy{Mode: Apply}})

			projected := result.History[1].Content
			if projected == tt.output || !strings.Contains(projected, tt.wantText) {
				t.Fatalf("projected command output = %q, want %q placeholder", projected, tt.wantText)
			}
			for _, reject := range []string{"[compacted old failed command output", "validation_failure", "lint_failure"} {
				if strings.Contains(projected, reject) {
					t.Fatalf("projected side-effect success mislabeled with %q:\n%s", reject, projected)
				}
			}
			commandReport := result.Report.CommandEditDryRun
			if commandReport.CommandReplacedCount != 1 ||
				commandReport.ArtifactBackedCommandReplacedCount != 0 ||
				!result.Report.ResponsesChainDisabled {
				t.Fatalf("command report = %#v / top-level %#v, want one inline side-effect replacement", commandReport, result.Report)
			}
			candidate := providerHistoryTestCommandCandidateByToolCallID(commandReport, tt.callID)
			if candidate == nil ||
				candidate.Reason != tt.reason ||
				candidate.Classifier != tt.classifier ||
				!candidate.ReplacementEligible ||
				!candidate.ReplacementApplied {
				t.Fatalf("candidate = %#v, want applied %s/%s side-effect candidate", candidate, tt.reason, tt.classifier)
			}
			if got := commandReport.CommandReplacementClassifierCounts[tt.classifier]; got != 1 {
				t.Fatalf("CommandReplacementClassifierCounts = %#v, want %s:1", commandReport.CommandReplacementClassifierCounts, tt.classifier)
			}
		})
	}
}

func TestProjectApplyKeepsAmbiguousSideEffectOutputsRaw(t *testing.T) {
	tests := []struct {
		name       string
		callID     string
		command    string
		output     string
		wantReason string
	}{
		{
			name:       "database migration partial failure",
			callID:     "call_migrate",
			command:    "prisma migrate deploy",
			output:     "migration started\npartial output: command interrupted before completion\n",
			wantReason: "database_failure_not_large",
		},
		{
			name:       "package interrupted",
			callID:     "call_install",
			command:    "npm install",
			output:     "resolving packages\ncommand interrupted before completion\n",
			wantReason: "package_failure_not_large",
		},
		{
			name:       "deploy partial",
			callID:     "call_deploy",
			command:    "vercel deploy",
			output:     "uploading deployment\npartial output: connection closed\n",
			wantReason: "deploy_failure_not_large",
		},
		{
			name:       "package short success stays raw below threshold",
			callID:     "call_install_ambiguous",
			command:    "npm install",
			output:     strings.Repeat("resolving packages\nfetching metadata\n", 4),
			wantReason: "command_replacement_below_min_saved_tokens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			history := []api.Message{
				providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, tt.callID, "bash", map[string]string{"command": tt.command})),
				providerHistoryTestToolResult(tt.callID, "bash", tt.output),
				{Role: "assistant", Content: "side effect not confirmed"},
				providerHistoryTestAssistantToolCall("call_latest", "read_file"),
				providerHistoryTestToolResult("call_latest", "read_file", "latest"),
				{Role: "assistant", Content: "done"},
			}

			result := Project(ProjectionInput{Messages: history, Policy: Policy{Mode: Apply}})

			if result.History[1].Content != tt.output {
				t.Fatalf("projected ambiguous side-effect output changed:\n got %q\nwant %q", result.History[1].Content, tt.output)
			}
			commandReport := result.Report.CommandEditDryRun
			if commandReport.CommandReplacedCount != 0 ||
				commandReport.ArtifactBackedCommandReplacedCount != 0 ||
				result.Report.ResponsesChainDisabled {
				t.Fatalf("command report = %#v / top-level %#v, want raw keep and response chain preserved", commandReport, result.Report)
			}
			candidate := providerHistoryTestCommandCandidateByToolCallID(commandReport, tt.callID)
			if candidate == nil ||
				candidate.ReplacementApplied ||
				candidate.KeepReason != tt.wantReason {
				t.Fatalf("candidate = %#v, want raw keep reason %q", candidate, tt.wantReason)
			}
			if got := commandReport.KeptReasonCounts[tt.wantReason]; got != 1 {
				t.Fatalf("KeptReasonCounts = %#v, want %s:1", commandReport.KeptReasonCounts, tt.wantReason)
			}
		})
	}
}

func TestProjectApplyCompactsLintFailureMarkersAsFailure(t *testing.T) {
	commandOutput := strings.Join([]string{
		"ESLint found too many warnings",
		"2 problems found",
		"Process exited with code 0",
		strings.Repeat("lint detail\n", 90),
	}, "\n")
	history := []api.Message{
		providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, "call_lint", "bash", map[string]string{"command": "npm run lint"})),
		providerHistoryTestToolResult("call_lint", "bash", commandOutput),
		{Role: "assistant", Content: "lint failed"},
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest"),
		{Role: "assistant", Content: "done"},
	}

	result := Project(ProjectionInput{Messages: history, Policy: Policy{Mode: Apply}})

	projected := result.History[1].Content
	if projected == commandOutput ||
		!strings.Contains(projected, "[compacted old failed command output") ||
		!strings.Contains(projected, "classifier=lint_failure") {
		t.Fatalf("projected lint output = %q, want lint failure compact", projected)
	}
	for _, reject := range []string{"validation_success", "classifier=validation", "successful validation command output"} {
		if strings.Contains(projected, reject) {
			t.Fatalf("lint failure was mislabeled as validation success with %q:\n%s", reject, projected)
		}
	}
	commandReport := result.Report.CommandEditDryRun
	if commandReport.CommandReplacedCount != 1 || !result.Report.ResponsesChainDisabled {
		t.Fatalf("command report = %#v / top-level %#v, want one lint failure replacement and chain disable", commandReport, result.Report)
	}
	candidate := providerHistoryTestCommandCandidateByToolCallID(commandReport, "call_lint")
	if candidate == nil ||
		candidate.Reason != "lint_failure" ||
		candidate.Classifier != "lint_failure" ||
		!candidate.ReplacementEligible ||
		!candidate.ReplacementApplied {
		t.Fatalf("candidate = %#v, want applied lint failure candidate", candidate)
	}
	if got := commandReport.CommandReplacementClassifierCounts["lint_failure"]; got != 1 {
		t.Fatalf("CommandReplacementClassifierCounts = %#v, want lint_failure:1", commandReport.CommandReplacementClassifierCounts)
	}
}
