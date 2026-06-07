package providerhistory

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
)

func TestProjectApplyKeepsOldDataBearingCommandOutputRaw(t *testing.T) {
	commandOutput := providerHistoryTestNumberedLines("api-result", 120)
	history := []api.Message{
		providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, "call_curl", "bash", map[string]string{"command": "curl 'https://api.example.test/items?token=secret-value'"})),
		providerHistoryTestToolResult("call_curl", "bash", commandOutput),
		{Role: "assistant", Content: "api data reviewed"},
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest"),
		{Role: "assistant", Content: "done"},
	}

	result := Project(ProjectionInput{Messages: history, Policy: Policy{Mode: Apply}})

	if projected := result.History[1].Content; projected != commandOutput {
		t.Fatalf("projected command output changed:\n got %q\nwant %q", projected, commandOutput)
	}
	commandReport := result.Report.CommandEditDryRun
	if commandReport.CommandReplacedCount != 0 || result.Report.ResponsesChainDisabled {
		t.Fatalf("command report = %#v / top-level %#v, want data-bearing keep without replacement or chain disable", commandReport, result.Report)
	}
	if got := commandReport.CommandCandidates; got != 1 {
		t.Fatalf("CommandCandidates = %d, want 1", got)
	}
	if got := commandReport.KeptReasonCounts["data_bearing_network_command_output_keep"]; got != 1 {
		t.Fatalf("KeptReasonCounts = %#v, want data_bearing_network_command_output_keep:1", commandReport.KeptReasonCounts)
	}
}

func TestProjectDryRunReportsArtifactBackedCommandOutputCandidate(t *testing.T) {
	commandOutput := providerHistoryTestNumberedLines("api-result", 6000)
	command := "curl 'https://api.example.test/items?foo=bar#frag'"
	history := []api.Message{
		providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, "call_curl", "bash", map[string]string{"command": command})),
		providerHistoryTestToolResult("call_curl", "bash", commandOutput),
		{Role: "assistant", Content: "api data reviewed"},
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest"),
		{Role: "assistant", Content: "done"},
	}
	store := providerHistoryTestRawOutputStore(t)

	result := Project(ProjectionInput{
		Messages: history,
		Policy: Policy{
			Mode:                             DryRun,
			RawOutputArtifactsMode:           RawOutputArtifactsDryRun,
			RawOutputArtifactStore:           store,
			SessionID:                        "session-artifact-dry-run",
			RawOutputRehydrateContextEnabled: true,
			ActiveContextTransportAvailable:  true,
		},
	})

	if !reflect.DeepEqual(result.History, history) {
		t.Fatalf("dry-run projection changed history:\n got %#v\nwant %#v", result.History, history)
	}
	report := result.Report
	commandReport := report.CommandEditDryRun
	if commandReport.ArtifactBackedCommandCandidates != 1 ||
		commandReport.ArtifactBackedCommandApplyEligible != 0 ||
		commandReport.ArtifactBackedCommandDryRunEstimatedSavedBytes <= 0 ||
		commandReport.ApproxArtifactBackedCommandDryRunEstimatedSavedTokens < providerHistoryRawOutputArtifactMinSavedTokens ||
		commandReport.ArtifactBackedCommandReplacementSavedBytes != 0 ||
		report.ArtifactBackedActualSavedBytes != 0 ||
		report.ResponsesChainDisabled {
		t.Fatalf("artifact dry-run report = %#v / top-level %#v, want candidate-only estimate without actual replacement", commandReport, report)
	}
	if report.EstimatedSavedBytes != 0 || report.ApproxSavedTokens != 0 {
		t.Fatalf("provider-facing savings = %d/%d, want dry-run artifact estimate kept separate", report.EstimatedSavedBytes, report.ApproxSavedTokens)
	}
	if report.RawOutputRefCount != 1 || report.RawOutputArtifactCount != 1 || len(report.RawOutputRefs) != 1 {
		t.Fatalf("raw output refs = count %d artifacts %d refs %#v, want one report-level ref", report.RawOutputRefCount, report.RawOutputArtifactCount, report.RawOutputRefs)
	}
	ref := report.RawOutputRefs[0]
	for _, reject := range []string{"foo=bar", "#frag"} {
		if strings.Contains(ref.CommandPreview, reject) {
			t.Fatalf("report raw output ref leaked %q in command preview: %#v", reject, ref)
		}
	}
	for _, want := range []string{"?redacted", "#redacted"} {
		if !strings.Contains(ref.CommandPreview, want) {
			t.Fatalf("report raw output ref command preview = %q, want %q", ref.CommandPreview, want)
		}
	}
	if report.DataBearingCandidateCount != 1 ||
		report.ArtifactBackedEstimatedSavedBytes != commandReport.ArtifactBackedCommandDryRunEstimatedSavedBytes ||
		report.ApproxArtifactBackedEstimatedSavedTokens != commandReport.ApproxArtifactBackedCommandDryRunEstimatedSavedTokens {
		t.Fatalf("top-level artifact metrics = %#v, command report = %#v, want aggregated artifact dry-run metrics", report, commandReport)
	}
	candidate := providerHistoryTestCommandCandidateByToolCallID(commandReport, "call_curl")
	if candidate == nil {
		t.Fatalf("CommandEditDryRun candidates = %#v, want call_curl candidate", commandReport.Candidates)
	}
	if !candidate.ArtifactBackedCandidate || candidate.RawOutputRefID == "" || candidate.ArtifactBackedApplyEligible {
		t.Fatalf("artifact candidate = %#v, want dry-run raw ref candidate without apply eligibility", candidate)
	}
	if candidate.RawOutputRefID != report.RawOutputRefs[0].RefID {
		t.Fatalf("candidate RawOutputRefID = %q, report ref = %q", candidate.RawOutputRefID, report.RawOutputRefs[0].RefID)
	}
	if !strings.Contains(candidate.SuggestedReplacementText, "raw_output_ref="+candidate.RawOutputRefID) ||
		!strings.Contains(candidate.SuggestedReplacementText, "excerpt:") {
		t.Fatalf("suggested replacement = %q, want raw_output_ref and bounded excerpt", candidate.SuggestedReplacementText)
	}
	for _, reject := range []string{"foo=bar", "#frag", t.TempDir(), "https://api.example.test/items?foo=bar"} {
		if strings.Contains(candidate.SuggestedReplacementText, reject) {
			t.Fatalf("suggested replacement leaked %q:\n%s", reject, candidate.SuggestedReplacementText)
		}
	}
}

func TestProjectKeepsSensitiveDataBearingCommandMetadataOutOfRawOutputArtifacts(t *testing.T) {
	commandOutput := providerHistoryTestNumberedLines("api-result", 6000)
	command := "curl -H 'Authorization: Basic dXNlcjpwYXNz' https://api.example.test/items"
	history := []api.Message{
		providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, "call_curl", "bash", map[string]string{"command": command})),
		providerHistoryTestToolResult("call_curl", "bash", commandOutput),
		{Role: "assistant", Content: "api data reviewed"},
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest"),
		{Role: "assistant", Content: "done"},
	}

	result := Project(ProjectionInput{
		Messages: history,
		Policy: Policy{
			Mode:                             Apply,
			RawOutputArtifactsMode:           RawOutputArtifactsApply,
			RawOutputArtifactStore:           panicRawOutputArtifactStore{},
			SessionID:                        "session-sensitive-command",
			RawOutputRehydrateContextEnabled: true,
			ActiveContextTransportAvailable:  true,
		},
	})

	if result.History[1].Content != commandOutput {
		t.Fatalf("projected command output changed:\n got %q\nwant raw output", result.History[1].Content)
	}
	if result.Report.RawOutputRefCount != 0 || len(result.Report.RawOutputRefs) != 0 || result.Report.ResponsesChainDisabled {
		t.Fatalf("report = %#v, want no raw refs and no provider-facing replacement for sensitive command", result.Report)
	}
	commandReport := result.Report.CommandEditDryRun
	if commandReport.ArtifactBackedCommandCandidates != 1 ||
		commandReport.ArtifactBackedCommandApplyEligible != 0 ||
		commandReport.ArtifactBackedCommandReplacedCount != 0 ||
		commandReport.ArtifactBackedKeptReasonCounts[string(rawoutputs.ReasonSensitiveArtifactForbidden)] != 1 {
		t.Fatalf("command report = %#v, want sensitive command raw keep", commandReport)
	}
	candidate := providerHistoryTestCommandCandidateByToolCallID(commandReport, "call_curl")
	if candidate == nil ||
		candidate.SafetyStatus != "sensitive" ||
		candidate.ArtifactGateStatus != "sensitive_command" ||
		candidate.KeepReason != string(rawoutputs.ReasonSensitiveArtifactForbidden) {
		t.Fatalf("candidate = %#v, want sensitive command fail-closed metadata", candidate)
	}
}

func TestProjectReadOnlyPolicyDoesNotMaterializeRawOutputArtifacts(t *testing.T) {
	commandOutput := providerHistoryTestNumberedLines("api-result", 6000)
	history := []api.Message{
		providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, "call_curl", "bash", map[string]string{"command": "curl https://api.example.test/items"})),
		providerHistoryTestToolResult("call_curl", "bash", commandOutput),
		{Role: "assistant", Content: "api data reviewed"},
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest"),
		{Role: "assistant", Content: "done"},
	}

	result := Project(ProjectionInput{
		Messages: history,
		Policy: Policy{
			Mode:                             Apply,
			RawOutputArtifactsMode:           RawOutputArtifactsApply,
			RawOutputArtifactStore:           panicRawOutputArtifactStore{},
			SessionID:                        "session-artifact-read-only",
			RawOutputRehydrateContextEnabled: true,
			ActiveContextTransportAvailable:  true,
			SideEffects:                      ProjectionSideEffectsReadOnly,
		},
	})

	if result.History[1].Content != commandOutput {
		t.Fatalf("projected command output changed:\n got %q\nwant raw output", result.History[1].Content)
	}
	if result.Report.RawOutputRefCount != 0 || len(result.Report.RawOutputRefs) != 0 || result.Report.ResponsesChainDisabled {
		t.Fatalf("report = %#v, want no raw refs and no provider-facing replacement in read-only projection", result.Report)
	}
	commandReport := result.Report.CommandEditDryRun
	if commandReport.ArtifactBackedCommandCandidates != 1 ||
		commandReport.ArtifactBackedCommandApplyEligible != 0 ||
		commandReport.ArtifactBackedCommandReplacedCount != 0 ||
		commandReport.ArtifactBackedKeptReasonCounts[providerHistoryRawOutputMaterializationReadOnlyReason] != 1 {
		t.Fatalf("command report = %#v, want read-only artifact keep", commandReport)
	}
}

func TestProjectKeepsSensitiveDataBearingCommandBodyOutOfRawOutputArtifacts(t *testing.T) {
	commandOutput := strings.Repeat("Authorization: Bearer abcdef\napi-result\n", 240)
	history := []api.Message{
		providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, "call_curl", "bash", map[string]string{"command": "curl https://api.example.test/items"})),
		providerHistoryTestToolResult("call_curl", "bash", commandOutput),
		{Role: "assistant", Content: "api data reviewed"},
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest"),
		{Role: "assistant", Content: "done"},
	}

	result := Project(ProjectionInput{
		Messages: history,
		Policy: Policy{
			Mode:                             Apply,
			RawOutputArtifactsMode:           RawOutputArtifactsApply,
			RawOutputArtifactStore:           panicRawOutputArtifactStore{},
			SessionID:                        "session-sensitive-body",
			RawOutputRehydrateContextEnabled: true,
			ActiveContextTransportAvailable:  true,
		},
	})

	if result.History[1].Content != commandOutput {
		t.Fatalf("projected command output changed:\n got %q\nwant raw output", result.History[1].Content)
	}
	if result.Report.RawOutputRefCount != 0 || len(result.Report.RawOutputRefs) != 0 || result.Report.ResponsesChainDisabled {
		t.Fatalf("report = %#v, want no raw refs and no provider-facing replacement for sensitive body", result.Report)
	}
	commandReport := result.Report.CommandEditDryRun
	if commandReport.ArtifactBackedCommandCandidates != 1 ||
		commandReport.ArtifactBackedCommandApplyEligible != 0 ||
		commandReport.ArtifactBackedCommandReplacedCount != 0 ||
		commandReport.ArtifactBackedKeptReasonCounts[string(rawoutputs.ReasonSensitiveArtifactForbidden)] != 1 {
		t.Fatalf("command report = %#v, want sensitive body raw keep", commandReport)
	}
	candidate := providerHistoryTestCommandCandidateByToolCallID(commandReport, "call_curl")
	if candidate == nil ||
		candidate.SafetyStatus != "sensitive" ||
		candidate.ArtifactGateStatus != "sensitive_body" ||
		candidate.KeepReason != string(rawoutputs.ReasonSensitiveArtifactForbidden) {
		t.Fatalf("candidate = %#v, want sensitive body fail-closed metadata", candidate)
	}
}

func TestProjectApplyKeepsArtifactBackedCommandRawWhenChildModeDryRun(t *testing.T) {
	commandOutput := providerHistoryTestNumberedLines("api-result", 6000)
	history := []api.Message{
		providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, "call_curl", "bash", map[string]string{"command": "curl https://api.example.test/items"})),
		providerHistoryTestToolResult("call_curl", "bash", commandOutput),
		{Role: "assistant", Content: "api data reviewed"},
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest"),
		{Role: "assistant", Content: "done"},
	}

	result := Project(ProjectionInput{
		Messages: history,
		Policy: Policy{
			Mode:                             Apply,
			RawOutputArtifactsMode:           RawOutputArtifactsDryRun,
			RawOutputArtifactStore:           providerHistoryTestRawOutputStore(t),
			SessionID:                        "session-artifact-apply-child-dry-run",
			RawOutputRehydrateContextEnabled: true,
			ActiveContextTransportAvailable:  true,
		},
	})

	if result.History[1].Content != commandOutput {
		t.Fatalf("projected command output changed:\n got %q\nwant raw output", result.History[1].Content)
	}
	commandReport := result.Report.CommandEditDryRun
	if commandReport.ArtifactBackedCommandCandidates != 1 ||
		commandReport.ArtifactBackedCommandApplyEligible != 0 ||
		commandReport.ArtifactBackedCommandReplacedCount != 0 ||
		commandReport.ArtifactBackedKeptReasonCounts["raw_output_artifacts_apply_mode_disabled"] != 1 ||
		result.Report.ResponsesChainDisabled {
		t.Fatalf("command report = %#v / top-level %#v, want child dry-run candidate with raw keep and no chain disable", commandReport, result.Report)
	}
	if result.Report.RawOutputRefCount != 1 {
		t.Fatalf("RawOutputRefCount = %d, want one created dry-run ref", result.Report.RawOutputRefCount)
	}
}

type panicRawOutputArtifactStore struct{}

func (panicRawOutputArtifactStore) Create(context.Context, rawoutputs.CreateRequest) (rawoutputs.CreateResult, error) {
	panic("Create must not be called for read-only projection")
}

func (panicRawOutputArtifactStore) Verify(context.Context, rawoutputs.RawOutputRef) (rawoutputs.VerifyResult, error) {
	panic("Verify must not be called for read-only projection")
}

func TestProjectApplyCompactsArtifactBackedCommandOutputWithRefAndRehydrateGate(t *testing.T) {
	commandOutput := providerHistoryTestNumberedLines("api-result", 6000)
	history := []api.Message{
		providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, "call_curl", "bash", map[string]string{"command": "curl https://api.example.test/items"})),
		providerHistoryTestToolResult("call_curl", "bash", commandOutput),
		{Role: "assistant", Content: "api data reviewed"},
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
			SessionID:                        "session-artifact-apply",
			RawOutputRehydrateContextEnabled: true,
			ActiveContextTransportAvailable:  true,
		},
	})

	projected := result.History[1].Content
	if projected == commandOutput ||
		!strings.Contains(projected, "[compacted old data-bearing command output;") ||
		!strings.Contains(projected, "raw_output_ref=") {
		t.Fatalf("projected command output = %q, want artifact-backed placeholder", projected)
	}
	commandReport := result.Report.CommandEditDryRun
	if commandReport.ArtifactBackedCommandCandidates != 1 ||
		commandReport.ArtifactBackedCommandApplyEligible != 1 ||
		commandReport.ArtifactBackedCommandReplacedCount != 1 ||
		commandReport.ArtifactBackedCommandReplacementSavedBytes <= 0 ||
		commandReport.ApproxArtifactBackedCommandReplacementSavedTokens < providerHistoryRawOutputArtifactMinSavedTokens ||
		!result.Report.ResponsesChainDisabled {
		t.Fatalf("command report = %#v / top-level %#v, want artifact-backed replacement and chain disable", commandReport, result.Report)
	}
	if commandReport.CommandReplacedCount != 0 || commandReport.CommandReplacementSavedBytes != 0 {
		t.Fatalf("inline command counters = %#v, want artifact replacement counted separately", commandReport)
	}
	if result.Report.ArtifactBackedActualSavedBytes != commandReport.ArtifactBackedCommandReplacementSavedBytes ||
		result.Report.EstimatedSavedBytes != commandReport.ArtifactBackedCommandReplacementSavedBytes ||
		result.Report.ApproxSavedTokens != commandReport.ApproxArtifactBackedCommandReplacementSavedTokens {
		t.Fatalf("top-level savings = %#v, command report = %#v, want artifact actual savings only", result.Report, commandReport)
	}
}

func TestProjectApplyKeepsValidationWrapperSuccessWithExpectedExceptionLogsSuccessful(t *testing.T) {
	commandOutput := strings.Join([]string{
		"=== RUN   TestExpectedExceptionPath",
		"expected exception: sentinel value",
		"Traceback (most recent call last):",
		"ValueError: expected sentinel",
		"panic: expected sentinel recovered by test",
		"ok\tgithub.com/acme/project\t0.001s",
		"Process exited with code 0",
		strings.Repeat("verbose passing validation detail\n", 120),
	}, "\n")
	history := []api.Message{
		providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, "call_ci", "bash", map[string]string{"command": "make ci-check"})),
		providerHistoryTestToolResult("call_ci", "bash", commandOutput),
		{Role: "assistant", Content: "validation passed"},
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest"),
		{Role: "assistant", Content: "done"},
	}

	result := Project(ProjectionInput{Messages: history, Policy: Policy{Mode: Apply}})

	projected := result.History[1].Content
	if projected == commandOutput || !strings.Contains(projected, "successful validation command output") {
		t.Fatalf("projected command output = %q, want validation success placeholder", projected)
	}
	for _, reject := range []string{"validation_failure", "lint_failure", "build_failure"} {
		if strings.Contains(projected, reject) {
			t.Fatalf("projected validation success mislabeled with %q:\n%s", reject, projected)
		}
	}
	commandReport := result.Report.CommandEditDryRun
	if commandReport.CommandReplacedCount != 1 || !result.Report.ResponsesChainDisabled {
		t.Fatalf("command report = %#v / top-level %#v, want one validation replacement and chain disable", commandReport, result.Report)
	}
	if got := commandReport.CommandReplacementClassifierCounts["validation"]; got != 1 {
		t.Fatalf("CommandReplacementClassifierCounts = %#v, want validation:1", commandReport.CommandReplacementClassifierCounts)
	}
}

func TestProjectApplyCompactsEvidenceBearingCommandOutputWithRefAndRehydrateGate(t *testing.T) {
	tests := []struct {
		name       string
		callID     string
		command    string
		output     string
		classifier string
	}{
		{
			name:       "observation grep output",
			callID:     "call_rg",
			command:    `rg "error:" internal`,
			output:     strings.Repeat(`internal/a.go:12: return fmt.Errorf("error: expected sentinel")`+"\n", 6000),
			classifier: "observation",
		},
		{
			name:       "file dump output",
			callID:     "call_cat",
			command:    "cat internal/errors.txt",
			output:     providerHistoryTestNumberedLines("file-line", 6000),
			classifier: "file_dump",
		},
		{
			name:    "git diff output",
			callID:  "call_git_diff",
			command: "git diff",
			output: strings.Repeat(
				"diff --git a/internal/a.go b/internal/a.go\n@@ -1,2 +1,2 @@ func f()\n-old\n+new\n",
				1200,
			),
			classifier: "git_diff",
		},
		{
			name:    "git show output",
			callID:  "call_git_show",
			command: "git show HEAD",
			output: strings.Repeat(
				"commit abcdef1234567890\nAuthor: Test <test@example.com>\n\n    update feature\n\ndiff --git a/internal/a.go b/internal/a.go\n@@ -1,2 +1,2 @@ func f()\n-old\n+new\n",
				1200,
			),
			classifier: "git_show",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			history := []api.Message{
				providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, tt.callID, "bash", map[string]string{"command": tt.command})),
				providerHistoryTestToolResult(tt.callID, "bash", tt.output),
				{Role: "assistant", Content: "evidence reviewed"},
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
					SessionID:                        "session-evidence-" + tt.callID,
					RawOutputRehydrateContextEnabled: true,
					ActiveContextTransportAvailable:  true,
				},
			})

			projected := result.History[1].Content
			if projected == tt.output ||
				!strings.Contains(projected, "[compacted old data-bearing command output;") ||
				!strings.Contains(projected, "raw_output_ref=") {
				t.Fatalf("projected command output = %q, want artifact-backed placeholder", projected)
			}
			for _, reject := range []string{"successful observation command output", "successful file dump command output", "[compacted old git diff output", "[compacted old git show output"} {
				if strings.Contains(projected, reject) {
					t.Fatalf("projected evidence output used inline compact marker %q:\n%s", reject, projected)
				}
			}
			commandReport := result.Report.CommandEditDryRun
			if commandReport.ArtifactBackedCommandCandidates != 1 ||
				commandReport.ArtifactBackedCommandApplyEligible != 1 ||
				commandReport.ArtifactBackedCommandReplacedCount != 1 ||
				commandReport.CommandReplacedCount != 0 ||
				!result.Report.ResponsesChainDisabled {
				t.Fatalf("command report = %#v / top-level %#v, want artifact-backed evidence replacement only", commandReport, result.Report)
			}
			candidate := providerHistoryTestCommandCandidateByToolCallID(commandReport, tt.callID)
			if candidate == nil ||
				!candidate.ArtifactBackedCandidate ||
				candidate.RawOutputRefID == "" ||
				candidate.Classifier != tt.classifier ||
				candidate.RehydrateGateStatus != "available" ||
				!candidate.ArtifactBackedApplyEligible {
				t.Fatalf("candidate = %#v, want artifact-backed %s evidence candidate", candidate, tt.classifier)
			}
			if result.Report.RawOutputRefCount != 1 || len(result.Report.RawOutputRefs) != 1 || result.Report.RawOutputRefs[0].RefID != candidate.RawOutputRefID {
				t.Fatalf("raw output refs = %#v / candidate %#v, want one matching raw ref", result.Report.RawOutputRefs, candidate)
			}
		})
	}
}

func TestProjectApplyKeepsEvidenceBearingCommandOutputRawWithoutRehydrateGate(t *testing.T) {
	commandOutput := strings.Repeat(`internal/a.go:12: return fmt.Errorf("error: expected sentinel")`+"\n", 6000)
	history := []api.Message{
		providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, "call_rg", "bash", map[string]string{"command": `rg "error:" internal`})),
		providerHistoryTestToolResult("call_rg", "bash", commandOutput),
		{Role: "assistant", Content: "matches reviewed"},
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest"),
		{Role: "assistant", Content: "done"},
	}

	result := Project(ProjectionInput{Messages: history, Policy: Policy{Mode: Apply}})

	if result.History[1].Content != commandOutput {
		t.Fatalf("projected command output changed without raw artifact gate:\n got %q\nwant raw evidence", result.History[1].Content)
	}
	commandReport := result.Report.CommandEditDryRun
	if commandReport.CommandReplacedCount != 0 ||
		commandReport.ArtifactBackedCommandReplacedCount != 0 ||
		commandReport.ArtifactBackedCommandCandidates != 0 ||
		result.Report.RawOutputRefCount != 0 ||
		result.Report.ResponsesChainDisabled {
		t.Fatalf("command report = %#v / top-level %#v, want raw keep without artifact side effects", commandReport, result.Report)
	}
	if got := commandReport.KeptReasonCounts["evidence_bearing_observation_command_output_keep"]; got != 1 {
		t.Fatalf("KeptReasonCounts = %#v, want evidence_bearing_observation_command_output_keep:1", commandReport.KeptReasonCounts)
	}
}

func TestProjectApplyCompactsPackageTextualFailureAsFailure(t *testing.T) {
	commandOutput := strings.Repeat("error: dependency resolution failed\ncommand failed: npm install\n", 120)
	history := []api.Message{
		providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, "call_install", "bash", map[string]string{"command": "npm install"})),
		providerHistoryTestToolResult("call_install", "bash", commandOutput),
		{Role: "assistant", Content: "install failed"},
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest"),
		{Role: "assistant", Content: "done"},
	}

	result := Project(ProjectionInput{Messages: history, Policy: Policy{Mode: Apply}})

	projected := result.History[1].Content
	if projected == commandOutput || !strings.Contains(projected, "[compacted old failed command output") {
		t.Fatalf("projected command output = %q, want failed command compact", projected)
	}
	for _, reject := range []string{"successful side-effect command output", "exit=0"} {
		if strings.Contains(projected, reject) {
			t.Fatalf("projected package failure mislabeled with %q:\n%s", reject, projected)
		}
	}
	commandReport := result.Report.CommandEditDryRun
	if commandReport.CommandReplacedCount != 1 || !result.Report.ResponsesChainDisabled {
		t.Fatalf("command report = %#v / top-level %#v, want one package failure replacement and chain disable", commandReport, result.Report)
	}
	if got := commandReport.CommandReplacementClassifierCounts["package_failure"]; got != 1 {
		t.Fatalf("CommandReplacementClassifierCounts = %#v, want package_failure:1", commandReport.CommandReplacementClassifierCounts)
	}
}
