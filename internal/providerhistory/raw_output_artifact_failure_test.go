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

func TestProjectApplyKeepsArtifactBackedGenericToolRawOnStoreFailure(t *testing.T) {
	tests := []struct {
		name       string
		history    []api.Message
		callID     string
		store      RawOutputArtifactStore
		wantReason string
	}{
		{
			name:    "mcp materialize failure",
			history: providerHistoryTestMCPHistory("call_mcp_docs", providerHistoryTestLargeSafeMCPResult()),
			callID:  "call_mcp_docs",
			store: failingRawOutputArtifactStore{
				materializeErr: rawoutputs.Error{Reason: rawoutputs.ReasonArtifactMaterializationFailed, Err: errors.New("materialize failed")},
			},
			wantReason: string(rawoutputs.ReasonArtifactMaterializationFailed),
		},
		{
			name:    "mcp verify failure",
			history: providerHistoryTestMCPHistory("call_mcp_docs", providerHistoryTestLargeSafeMCPResult()),
			callID:  "call_mcp_docs",
			store: failingRawOutputArtifactStore{
				materializeResult: rawoutputs.CreateResult{Ref: rawoutputs.RawOutputRef{RefID: "rawout_mcp_verify_failed"}},
				verifyResult:      rawoutputs.VerifyResult{Reason: rawoutputs.ReasonArtifactHashMismatch},
			},
			wantReason: string(rawoutputs.ReasonArtifactHashMismatch),
		},
		{
			name:    "web_search materialize failure",
			history: providerHistoryTestWebSearchHistory(t, "call_web", "OpenAI Responses API usage guide", providerHistoryTestLargeWebSearchResult(), true),
			callID:  "call_web",
			store: failingRawOutputArtifactStore{
				materializeErr: rawoutputs.Error{Reason: rawoutputs.ReasonArtifactMaterializationFailed, Err: errors.New("materialize failed")},
			},
			wantReason: string(rawoutputs.ReasonArtifactMaterializationFailed),
		},
		{
			name:    "web_search verify failure",
			history: providerHistoryTestWebSearchHistory(t, "call_web", "OpenAI Responses API usage guide", providerHistoryTestLargeWebSearchResult(), true),
			callID:  "call_web",
			store: failingRawOutputArtifactStore{
				materializeResult: rawoutputs.CreateResult{Ref: rawoutputs.RawOutputRef{RefID: "rawout_web_verify_failed"}},
				verifyResult:      rawoutputs.VerifyResult{Reason: rawoutputs.ReasonArtifactHashMismatch},
			},
			wantReason: string(rawoutputs.ReasonArtifactHashMismatch),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Project(ProjectionInput{
				Messages: tt.history,
				Policy: Policy{
					Mode:                             Apply,
					RawOutputArtifactsMode:           RawOutputArtifactsApply,
					RawOutputArtifactStore:           tt.store,
					SessionID:                        "session-generic-artifact-failure",
					RawOutputRehydrateContextEnabled: true,
					ActiveContextTransportAvailable:  true,
				},
			})

			if !reflect.DeepEqual(result.History, tt.history) {
				t.Fatalf("projection changed history on fail-closed path:\n got %#v\nwant %#v", result.History, tt.history)
			}
			if result.Report.ReplacedCount != 0 || result.Report.ResponsesChainDisabled || result.Report.RawOutputRefCount != 0 {
				t.Fatalf("report = %#v, want no generic replacement, response-chain disable, or raw refs on fail-closed path", result.Report)
			}
			candidate := providerHistoryTestCandidateByToolCallID(result.Report, tt.callID)
			if candidate == nil ||
				!candidate.ArtifactBackedCandidate ||
				candidate.ArtifactBackedApplyEligible ||
				candidate.ReplacementApplied ||
				candidate.RawOutputRefID != "" ||
				candidate.KeepReason != tt.wantReason ||
				candidate.FailClosedReason != tt.wantReason {
				t.Fatalf("candidate = %#v, want artifact fail-closed reason %q", candidate, tt.wantReason)
			}
			if got := result.Report.KeptReasonCounts[tt.wantReason]; got != 1 {
				t.Fatalf("KeptReasonCounts[%q] = %d in %#v, want 1", tt.wantReason, got, result.Report.KeptReasonCounts)
			}
		})
	}
}

func TestProjectApplyKeepsArtifactBackedCommandRawWhenApplyReverifyFails(t *testing.T) {
	commandOutput := providerHistoryTestNumberedLines("api-result", 6000)
	history := []api.Message{
		providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, "call_curl", "bash", map[string]string{"command": "curl https://api.example.test/items"})),
		providerHistoryTestToolResult("call_curl", "bash", commandOutput),
		{Role: "assistant", Content: "api data reviewed"},
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest"),
		{Role: "assistant", Content: "done"},
	}
	store := &reverifyFailingRawOutputArtifactStore{
		inner:      providerHistoryTestRawOutputStore(t),
		failReason: rawoutputs.ReasonArtifactHashMismatch,
	}

	result := Project(ProjectionInput{
		Messages: history,
		Policy: Policy{
			Mode:                             Apply,
			RawOutputArtifactsMode:           RawOutputArtifactsApply,
			RawOutputArtifactStore:           store,
			SessionID:                        "session-command-reverify-failure",
			RawOutputRehydrateContextEnabled: true,
			ActiveContextTransportAvailable:  true,
		},
	})

	if store.verifyCalls != 2 {
		t.Fatalf("Verify calls = %d, want candidate verify and apply re-verify", store.verifyCalls)
	}
	if !reflect.DeepEqual(result.History, history) {
		t.Fatalf("projection changed history after apply re-verify failed:\n got %#v\nwant %#v", result.History, history)
	}
	if result.Report.ResponsesChainDisabled || result.Report.EstimatedSavedBytes != 0 || result.Report.ArtifactBackedActualSavedBytes != 0 {
		t.Fatalf("report = %#v, want no actual provider-facing savings or response-chain disable", result.Report)
	}
	if result.Report.RawOutputRefCount != 1 {
		t.Fatalf("RawOutputRefCount = %d, want kept materialized ref still reported for diagnostics", result.Report.RawOutputRefCount)
	}
	commandReport := result.Report.CommandEditDryRun
	if commandReport.ArtifactBackedCommandCandidates != 1 ||
		commandReport.ArtifactBackedCommandApplyEligible != 1 ||
		commandReport.ArtifactBackedCommandReplacedCount != 0 ||
		commandReport.ArtifactBackedKeptReasonCounts[string(rawoutputs.ReasonArtifactHashMismatch)] != 1 {
		t.Fatalf("command report = %#v, want eligible artifact-backed candidate kept by re-verify failure", commandReport)
	}
	candidate := providerHistoryTestCommandCandidateByToolCallID(commandReport, "call_curl")
	if candidate == nil ||
		!candidate.ArtifactBackedCandidate ||
		!candidate.ArtifactBackedApplyEligible ||
		candidate.ReplacementApplied ||
		candidate.KeepReason != string(rawoutputs.ReasonArtifactHashMismatch) ||
		candidate.FailClosedReason != string(rawoutputs.ReasonArtifactHashMismatch) {
		t.Fatalf("candidate = %#v, want apply re-verify fail-closed reason", candidate)
	}
}

func TestProjectApplyKeepsArtifactBackedGenericToolRawWhenApplyReverifyFails(t *testing.T) {
	content := providerHistoryTestLargeSafeMCPResult()
	history := providerHistoryTestMCPHistory("call_mcp_docs", content)
	store := &reverifyFailingRawOutputArtifactStore{
		inner:      providerHistoryTestRawOutputStore(t),
		failReason: rawoutputs.ReasonArtifactHashMismatch,
	}

	result := Project(ProjectionInput{
		Messages: history,
		Policy: Policy{
			Mode:                             Apply,
			RawOutputArtifactsMode:           RawOutputArtifactsApply,
			RawOutputArtifactStore:           store,
			SessionID:                        "session-mcp-reverify-failure",
			RawOutputRehydrateContextEnabled: true,
			ActiveContextTransportAvailable:  true,
		},
	})

	if store.verifyCalls != 2 {
		t.Fatalf("Verify calls = %d, want candidate verify and apply re-verify", store.verifyCalls)
	}
	if !reflect.DeepEqual(result.History, history) {
		t.Fatalf("projection changed history after apply re-verify failed:\n got %#v\nwant %#v", result.History, history)
	}
	if result.Report.ReplacedCount != 0 || result.Report.ResponsesChainDisabled || result.Report.EstimatedSavedBytes != 0 || result.Report.ArtifactBackedActualSavedBytes != 0 {
		t.Fatalf("report = %#v, want no actual generic replacement, savings, or response-chain disable", result.Report)
	}
	if result.Report.RawOutputRefCount != 1 {
		t.Fatalf("RawOutputRefCount = %d, want kept materialized ref still reported for diagnostics", result.Report.RawOutputRefCount)
	}
	candidate := providerHistoryTestCandidateByToolCallID(result.Report, "call_mcp_docs")
	if candidate == nil ||
		!candidate.ArtifactBackedCandidate ||
		!candidate.ArtifactBackedApplyEligible ||
		candidate.ReplacementApplied ||
		candidate.KeepReason != string(rawoutputs.ReasonArtifactHashMismatch) ||
		candidate.FailClosedReason != string(rawoutputs.ReasonArtifactHashMismatch) {
		t.Fatalf("candidate = %#v, want generic apply re-verify fail-closed reason", candidate)
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

type reverifyFailingRawOutputArtifactStore struct {
	inner       RawOutputArtifactStore
	verifyCalls int
	failReason  rawoutputs.Reason
}

func (s *reverifyFailingRawOutputArtifactStore) Create(ctx context.Context, req rawoutputs.CreateRequest) (rawoutputs.CreateResult, error) {
	return s.inner.Create(ctx, req)
}

func (s *reverifyFailingRawOutputArtifactStore) MaterializeLegacy(ctx context.Context, req rawoutputs.LegacyMaterializeRequest) (rawoutputs.CreateResult, error) {
	return s.inner.MaterializeLegacy(ctx, req)
}

func (s *reverifyFailingRawOutputArtifactStore) Verify(ctx context.Context, ref rawoutputs.RawOutputRef) (rawoutputs.VerifyResult, error) {
	s.verifyCalls++
	if s.verifyCalls == 1 {
		return s.inner.Verify(ctx, ref)
	}
	return rawoutputs.VerifyResult{Ref: ref, Reason: s.failReason}, nil
}
