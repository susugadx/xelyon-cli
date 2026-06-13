package providerhistory

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
)

func TestProjectApplyKeepsArtifactBackedCommandRawOnStoreFailure(t *testing.T) {
	tests := []struct {
		name       string
		store      RawOutputArtifactStore
		wantReason string
	}{
		{
			name: "materialize failure",
			store: failingRawOutputArtifactStore{
				materializeErr: rawoutputs.Error{Reason: rawoutputs.ReasonArtifactMaterializationFailed, Err: errors.New("materialize failed")},
			},
			wantReason: string(rawoutputs.ReasonArtifactMaterializationFailed),
		},
		{
			name: "verify failure",
			store: failingRawOutputArtifactStore{
				materializeResult: rawoutputs.CreateResult{Ref: rawoutputs.RawOutputRef{RefID: "rawout_verify_failed"}},
				verifyResult:      rawoutputs.VerifyResult{Reason: rawoutputs.ReasonArtifactHashMismatch},
			},
			wantReason: string(rawoutputs.ReasonArtifactHashMismatch),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
					RawOutputArtifactStore:           tt.store,
					SessionID:                        "session-artifact-failure",
					RawOutputRehydrateContextEnabled: true,
					ActiveContextTransportAvailable:  true,
				},
			})

			if !reflect.DeepEqual(result.History, history) {
				t.Fatalf("projection changed history on fail-closed path:\n got %#v\nwant %#v", result.History, history)
			}
			if result.Report.ResponsesChainDisabled || result.Report.RawOutputRefCount != 0 || len(result.Report.RawOutputRefs) != 0 {
				t.Fatalf("report = %#v, want no response-chain disable or raw refs on fail-closed path", result.Report)
			}
			commandReport := result.Report.CommandEditDryRun
			if commandReport.ArtifactBackedCommandCandidates != 1 ||
				commandReport.ArtifactBackedCommandApplyEligible != 0 ||
				commandReport.ArtifactBackedCommandReplacedCount != 0 ||
				commandReport.ArtifactBackedKeptReasonCounts[tt.wantReason] != 1 {
				t.Fatalf("command report = %#v, want one kept artifact-backed candidate with reason %q", commandReport, tt.wantReason)
			}
			candidate := providerHistoryTestCommandCandidateByToolCallID(commandReport, "call_curl")
			if candidate == nil ||
				!candidate.ArtifactBackedCandidate ||
				candidate.ArtifactBackedApplyEligible ||
				candidate.ArtifactGateStatus == "verified" ||
				candidate.KeepReason != tt.wantReason ||
				candidate.FailClosedReason != tt.wantReason {
				t.Fatalf("candidate = %#v, want artifact fail-closed reason %q", candidate, tt.wantReason)
			}
		})
	}
}

type failingRawOutputArtifactStore struct {
	materializeResult rawoutputs.CreateResult
	materializeErr    error
	verifyResult      rawoutputs.VerifyResult
	verifyErr         error
}

func (s failingRawOutputArtifactStore) Create(context.Context, rawoutputs.CreateRequest) (rawoutputs.CreateResult, error) {
	return rawoutputs.CreateResult{}, errors.New("Create must not be called for command artifact projection")
}

func (s failingRawOutputArtifactStore) MaterializeLegacy(context.Context, rawoutputs.LegacyMaterializeRequest) (rawoutputs.CreateResult, error) {
	return s.materializeResult, s.materializeErr
}

func (s failingRawOutputArtifactStore) Verify(context.Context, rawoutputs.RawOutputRef) (rawoutputs.VerifyResult, error) {
	return s.verifyResult, s.verifyErr
}
