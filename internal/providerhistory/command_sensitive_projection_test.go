package providerhistory

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestProjectApplyKeepsSensitiveCommandRawAndOutOfRawOutputArtifacts(t *testing.T) {
	output := strings.Repeat("TOKEN=secret-value\nPATH=/tmp/bin\n", 180)
	history := []api.Message{
		{Role: "user", Content: "inspect old env output"},
		providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, "call_env", "bash", map[string]string{"command": "env"})),
		providerHistoryTestToolResult("call_env", "bash", output),
		{Role: "assistant", Content: "env reviewed"},
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest"),
	}

	result := Project(ProjectionInput{
		Messages: history,
		Policy: Policy{
			Mode:                             Apply,
			RawOutputArtifactsMode:           RawOutputArtifactsApply,
			RawOutputArtifactStore:           providerHistoryTestRawOutputStore(t),
			SessionID:                        "session-sensitive",
			RawOutputRehydrateContextEnabled: true,
			ActiveContextTransportAvailable:  true,
		},
	})

	projected := result.History[2].Content
	if projected != output {
		t.Fatalf("sensitive command projection = %q, want raw keep", projected)
	}
	report := result.Report.CommandEditDryRun
	if report.ArtifactBackedCommandCandidates != 0 ||
		report.ArtifactBackedCommandReplacedCount != 0 ||
		result.Report.RawOutputRefCount != 0 ||
		len(result.Report.RawOutputRefs) != 0 {
		t.Fatalf("report = %#v, want sensitive command outside normal raw output artifact path", result.Report)
	}
	candidate := providerHistoryTestCommandCandidateByToolCallID(report, "call_env")
	if candidate == nil ||
		candidate.ArtifactBackedCandidate ||
		candidate.RawOutputRefID != "" ||
		candidate.ReplacementApplied ||
		candidate.KeepReason != "sensitive_output_artifact_forbidden" {
		t.Fatalf("sensitive command candidate = %#v, want raw keep without raw ref", candidate)
	}
}
