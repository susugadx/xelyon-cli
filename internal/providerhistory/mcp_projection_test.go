package providerhistory

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
)

func TestProjectDryRunReportsArtifactBackedMCPToolResultCandidate(t *testing.T) {
	content := providerHistoryTestLargeSafeMCPResult()
	history := providerHistoryTestMCPHistory("call_mcp_docs", content)

	result := Project(ProjectionInput{
		Messages: history,
		Policy: Policy{
			Mode:                             DryRun,
			RawOutputArtifactsMode:           RawOutputArtifactsDryRun,
			RawOutputArtifactStore:           providerHistoryTestRawOutputStore(t),
			SessionID:                        "session-mcp-dry-run",
			RawOutputRehydrateContextEnabled: true,
			ActiveContextTransportAvailable:  true,
		},
	})

	if !reflect.DeepEqual(result.History, history) {
		t.Fatalf("dry-run projection changed MCP payload:\n got %#v\nwant %#v", result.History, history)
	}
	candidate := providerHistoryTestCandidateByToolCallID(result.Report, "call_mcp_docs")
	if candidate == nil ||
		candidate.CandidateOnly ||
		!candidate.ArtifactBackedCandidate ||
		candidate.RawOutputRefID == "" ||
		candidate.ReplacementApplied ||
		candidate.ArtifactBackedApplyEligible ||
		candidate.EstimatedSavedBytes <= 0 ||
		candidate.ApproxEstimatedSavedTokens < providerHistoryRawOutputArtifactMinSavedTokens {
		t.Fatalf("MCP candidate = %#v, want artifact-backed dry-run estimate", candidate)
	}
	if result.Report.RawOutputRefCount != 1 ||
		result.Report.RawOutputRefs[0].Surface != "mcp_tool_result" ||
		result.Report.DataBearingCandidateCount != 1 ||
		result.Report.ArtifactBackedEstimatedSavedBytes != candidate.EstimatedSavedBytes ||
		result.Report.ArtifactBackedActualSavedBytes != 0 ||
		result.Report.EstimatedSavedBytes != 0 ||
		result.Report.ResponsesChainDisabled {
		t.Fatalf("report = %#v, want MCP artifact dry-run separated from actual savings", result.Report)
	}
	if !strings.Contains(candidate.SuggestedReplacementText, "raw_output_ref="+candidate.RawOutputRefID) ||
		!strings.Contains(candidate.SuggestedReplacementText, "[compacted old MCP tool result;") {
		t.Fatalf("MCP replacement = %q, want raw ref placeholder", candidate.SuggestedReplacementText)
	}
}

func TestProjectApplyKeepsPrivateLargeMCPToolResultCandidateOnly(t *testing.T) {
	content := providerHistoryTestLargePrivateMCPResult()
	history := providerHistoryTestMCPHistory("call_mcp_private", content)

	result := Project(ProjectionInput{
		Messages: history,
		Policy: Policy{
			Mode:                             Apply,
			RawOutputArtifactsMode:           RawOutputArtifactsApply,
			RawOutputArtifactStore:           providerHistoryTestRawOutputStore(t),
			SessionID:                        "session-mcp-private",
			RawOutputRehydrateContextEnabled: true,
			ActiveContextTransportAvailable:  true,
		},
	})

	if !reflect.DeepEqual(result.History, history) {
		t.Fatalf("private MCP projection changed payload:\n got %#v\nwant %#v", result.History, history)
	}
	candidate := providerHistoryTestCandidateByToolCallID(result.Report, "call_mcp_private")
	if candidate == nil ||
		!candidate.CandidateOnly ||
		!candidate.FutureApplyCandidate ||
		candidate.ArtifactBackedCandidate ||
		candidate.RawOutputRefID != "" ||
		candidate.ReplacementApplied ||
		candidate.KeepReason != "mcp_sensitive_or_private_result_keep" {
		t.Fatalf("MCP private candidate = %#v, want candidate-only private keep", candidate)
	}
	if result.Report.RawOutputRefCount != 0 ||
		result.Report.DataBearingCandidateCount != 0 ||
		result.Report.ReplacedCount != 0 ||
		result.Report.ResponsesChainDisabled {
		t.Fatalf("report = %#v, want no raw output ref or replacement for sensitive MCP payload", result.Report)
	}
	if got := result.Report.FutureFamilyCandidateCounts["mcp"]; got != 1 {
		t.Fatalf("FutureFamilyCandidateCounts[mcp] = %d in %#v, want 1", got, result.Report.FutureFamilyCandidateCounts)
	}
	if got := result.Report.FutureFamilyKeptReasonCounts["mcp_sensitive_or_private_result_keep"]; got != 1 {
		t.Fatalf("FutureFamilyKeptReasonCounts = %#v, want private keep count", result.Report.FutureFamilyKeptReasonCounts)
	}
}

func TestProjectApplyKeepsBareSecretLargeMCPToolResultOutsideRawOutputStore(t *testing.T) {
	content := providerHistoryTestLargeBareSecretMCPResult()
	history := providerHistoryTestMCPHistory("call_mcp_bare_secret", content)

	result := Project(ProjectionInput{
		Messages: history,
		Policy: Policy{
			Mode:                             Apply,
			RawOutputArtifactsMode:           RawOutputArtifactsApply,
			RawOutputArtifactStore:           providerHistoryTestRawOutputStore(t),
			SessionID:                        "session-mcp-bare-secret",
			RawOutputRehydrateContextEnabled: true,
			ActiveContextTransportAvailable:  true,
		},
	})

	if !reflect.DeepEqual(result.History, history) {
		t.Fatalf("bare-secret MCP projection changed payload:\n got %#v\nwant %#v", result.History, history)
	}
	candidate := providerHistoryTestCandidateByToolCallID(result.Report, "call_mcp_bare_secret")
	if candidate == nil ||
		!candidate.CandidateOnly ||
		candidate.ArtifactBackedCandidate ||
		candidate.RawOutputRefID != "" ||
		candidate.KeepReason != string(rawoutputs.ReasonSensitiveArtifactForbidden) {
		t.Fatalf("MCP bare-secret candidate = %#v, want normal raw output forbidden keep", candidate)
	}
	if result.Report.RawOutputRefCount != 0 ||
		result.Report.RawOutputContextRefCount != 0 ||
		result.Report.ReplacedCount != 0 ||
		result.Report.ResponsesChainDisabled {
		t.Fatalf("report = %#v, want no raw output ref or replacement for secret-like MCP payload", result.Report)
	}
}

func TestProjectApplyKeepsQuotedAuthorizationMCPToolResultOutsideRawOutputStore(t *testing.T) {
	content := providerHistoryTestLargeQuotedAuthorizationMCPResult()
	history := providerHistoryTestMCPHistory("call_mcp_auth_json", content)

	result := Project(ProjectionInput{
		Messages: history,
		Policy: Policy{
			Mode:                             Apply,
			RawOutputArtifactsMode:           RawOutputArtifactsApply,
			RawOutputArtifactStore:           providerHistoryTestRawOutputStore(t),
			SessionID:                        "session-mcp-auth-json",
			RawOutputRehydrateContextEnabled: true,
			ActiveContextTransportAvailable:  true,
		},
	})

	if !reflect.DeepEqual(result.History, history) {
		t.Fatalf("quoted-authorization MCP projection changed payload:\n got %#v\nwant %#v", result.History, history)
	}
	candidate := providerHistoryTestCandidateByToolCallID(result.Report, "call_mcp_auth_json")
	if candidate == nil ||
		!candidate.CandidateOnly ||
		candidate.ArtifactBackedCandidate ||
		candidate.RawOutputRefID != "" ||
		candidate.KeepReason != string(rawoutputs.ReasonSensitiveArtifactForbidden) {
		t.Fatalf("MCP quoted-authorization candidate = %#v, want sensitive keep outside raw output store", candidate)
	}
	if result.Report.RawOutputRefCount != 0 ||
		result.Report.RawOutputContextRefCount != 0 ||
		result.Report.ReplacedCount != 0 ||
		result.Report.ResponsesChainDisabled {
		t.Fatalf("report = %#v, want no raw output ref or replacement for quoted authorization MCP payload", result.Report)
	}
}

func TestProjectMCPRawOutputUsesLegacyMaterializeExactSource(t *testing.T) {
	content := providerHistoryTestLargeSafeMCPResult()
	history := providerHistoryTestMCPHistory("call_mcp_docs", content)
	store := providerHistoryTestRawOutputSpyStore(t)

	result := Project(ProjectionInput{
		Messages: history,
		Policy: Policy{
			Mode:                             DryRun,
			RawOutputArtifactsMode:           RawOutputArtifactsDryRun,
			RawOutputArtifactStore:           store,
			SessionID:                        "session-mcp-legacy-materialize",
			RawOutputRehydrateContextEnabled: true,
			ActiveContextTransportAvailable:  true,
		},
	})

	if result.Report.RawOutputRefCount != 1 {
		t.Fatalf("RawOutputRefCount = %d, want one MCP raw output ref", result.Report.RawOutputRefCount)
	}
	assertProviderHistoryLegacyMaterialize(t, store, rawoutputs.SurfaceMCPToolResult)
}

func TestProjectMCPRuntimePlaceholderRestoresContextRefWithoutRematerializing(t *testing.T) {
	store := providerHistoryTestRawOutputSpyStore(t)
	created, err := store.inner.Create(context.Background(), rawoutputs.CreateRequest{
		Surface:   rawoutputs.SurfaceMCPToolResult,
		SessionID: "session-mcp-runtime-placeholder",
		Source: rawoutputs.SourceMetadata{
			ToolName:   "mcp_context7_get_library_docs",
			ToolCallID: "call_mcp_runtime",
			EventID:    "tool_call:call_mcp_runtime",
		},
		Classification: rawoutputs.ClassificationMetadata{
			SemanticRole: "data_bearing",
			Family:       "mcp",
			Classifier:   "mcp_runtime_large_result",
		},
		Body:          strings.NewReader(providerHistoryTestLargeSafeMCPResult()),
		SizeHintBytes: int64(len(providerHistoryTestLargeSafeMCPResult())),
	})
	if err != nil {
		t.Fatalf("Create(runtime raw output) error = %v", err)
	}
	placeholder := providerHistoryTestMCPRuntimePlaceholder(created.Ref.RefID)
	history := providerHistoryTestMCPHistory("call_mcp_runtime", placeholder)

	result := Project(ProjectionInput{
		Messages: history,
		Policy: Policy{
			Mode:                             Apply,
			RawOutputArtifactsMode:           RawOutputArtifactsApply,
			RawOutputArtifactStore:           store,
			SessionID:                        "session-mcp-runtime-placeholder",
			RawOutputRehydrateContextEnabled: true,
			ActiveContextTransportAvailable:  true,
		},
	})

	if !reflect.DeepEqual(result.History, history) {
		t.Fatalf("runtime MCP placeholder projection changed payload:\n got %#v\nwant %#v", result.History, history)
	}
	candidate := providerHistoryTestCandidateByToolCallID(result.Report, "call_mcp_runtime")
	if candidate == nil ||
		!candidate.RawOutputContextRequired ||
		candidate.ArtifactBackedCandidate ||
		candidate.ReplacementApplied ||
		candidate.RawOutputRefID != created.Ref.RefID ||
		candidate.KeepReason != mcpRuntimeCompactedResultKeepReason {
		t.Fatalf("runtime MCP placeholder candidate = %#v, want context-only keep", candidate)
	}
	if result.Report.RawOutputRefCount != 0 ||
		result.Report.RawOutputContextRefCount != 1 ||
		result.Report.RawOutputContextRefs[0].RefID != created.Ref.RefID ||
		result.Report.ReplacedCount != 0 ||
		result.Report.ResponsesChainDisabled {
		t.Fatalf("report = %#v, want context ref without projection replacement", result.Report)
	}
	if store.createCalls != 0 || store.materializeCalls != 0 {
		t.Fatalf("raw output materialization calls = create %d legacy %d, want none for runtime placeholder", store.createCalls, store.materializeCalls)
	}
}

func TestProjectMCPRuntimePlaceholderRestoresLatestContextRef(t *testing.T) {
	store := providerHistoryTestRawOutputSpyStore(t)
	created := providerHistoryTestCreateMCPRuntimeRawOutputRef(t, store.inner, "session-mcp-runtime-latest", "call_mcp_runtime_latest")
	placeholder := providerHistoryTestMCPRuntimePlaceholder(created.RefID)
	history := []api.Message{
		providerHistoryTestAssistantToolCall("call_mcp_runtime_latest", "mcp_context7_get_library_docs"),
		providerHistoryTestToolResult("call_mcp_runtime_latest", "mcp_context7_get_library_docs", placeholder),
		{Role: "user", Content: "continue after immediate MCP result"},
	}

	result := Project(ProjectionInput{
		Messages: history,
		Policy: Policy{
			Mode:                             Apply,
			RawOutputArtifactsMode:           RawOutputArtifactsApply,
			RawOutputArtifactStore:           store,
			SessionID:                        "session-mcp-runtime-latest",
			RawOutputRehydrateContextEnabled: true,
			ActiveContextTransportAvailable:  true,
		},
	})

	assertProviderHistoryMCPRuntimePlaceholderContextRef(t, result, history, created.RefID, "call_mcp_runtime_latest")
	if store.createCalls != 0 || store.materializeCalls != 0 {
		t.Fatalf("raw output materialization calls = create %d legacy %d, want none for latest runtime placeholder", store.createCalls, store.materializeCalls)
	}
}

func TestProjectMCPRuntimePlaceholderRestoresTrailingContextRef(t *testing.T) {
	store := providerHistoryTestRawOutputSpyStore(t)
	created := providerHistoryTestCreateMCPRuntimeRawOutputRef(t, store.inner, "session-mcp-runtime-trailing", "call_mcp_runtime_trailing")
	placeholder := providerHistoryTestMCPRuntimePlaceholder(created.RefID)
	history := []api.Message{
		providerHistoryTestAssistantToolCall("call_mcp_runtime_trailing", "mcp_context7_get_library_docs"),
		providerHistoryTestToolResult("call_mcp_runtime_trailing", "mcp_context7_get_library_docs", placeholder),
	}

	result := Project(ProjectionInput{
		Messages: history,
		Policy: Policy{
			Mode:                             Apply,
			RawOutputArtifactsMode:           RawOutputArtifactsApply,
			RawOutputArtifactStore:           store,
			SessionID:                        "session-mcp-runtime-trailing",
			RawOutputRehydrateContextEnabled: true,
			ActiveContextTransportAvailable:  true,
		},
	})

	assertProviderHistoryMCPRuntimePlaceholderContextRef(t, result, history, created.RefID, "call_mcp_runtime_trailing")
	if store.createCalls != 0 || store.materializeCalls != 0 {
		t.Fatalf("raw output materialization calls = create %d legacy %d, want none for trailing runtime placeholder", store.createCalls, store.materializeCalls)
	}
}

func TestProjectMCPRuntimePlaceholderReportsContextRefWhenActiveContextUnavailable(t *testing.T) {
	store := providerHistoryTestRawOutputSpyStore(t)
	created := providerHistoryTestCreateMCPRuntimeRawOutputRef(t, store.inner, "session-mcp-runtime-unavailable", "call_mcp_runtime_unavailable")
	placeholder := providerHistoryTestMCPRuntimePlaceholder(created.RefID)
	history := []api.Message{
		providerHistoryTestAssistantToolCall("call_mcp_runtime_unavailable", "mcp_context7_get_library_docs"),
		providerHistoryTestToolResult("call_mcp_runtime_unavailable", "mcp_context7_get_library_docs", placeholder),
	}

	result := Project(ProjectionInput{
		Messages: history,
		Policy: Policy{
			Mode:                             Apply,
			RawOutputArtifactsMode:           RawOutputArtifactsApply,
			RawOutputArtifactStore:           store,
			SessionID:                        "session-mcp-runtime-unavailable",
			RawOutputRehydrateContextEnabled: true,
			ActiveContextTransportAvailable:  false,
		},
	})

	if !reflect.DeepEqual(result.History, history) {
		t.Fatalf("runtime MCP placeholder projection changed payload:\n got %#v\nwant %#v", result.History, history)
	}
	candidate := providerHistoryTestCandidateByToolCallID(result.Report, "call_mcp_runtime_unavailable")
	if candidate == nil ||
		candidate.RawOutputContextRequired ||
		candidate.ArtifactBackedCandidate ||
		candidate.ReplacementApplied ||
		candidate.RawOutputRefID != created.RefID ||
		candidate.KeepReason != mcpRuntimeCompactedResultKeepReason {
		t.Fatalf("runtime MCP placeholder candidate = %#v, want liveness-only context ref keep", candidate)
	}
	if result.Report.RawOutputRefCount != 0 ||
		result.Report.RawOutputContextRefCount != 1 ||
		result.Report.RawOutputContextRefs[0].RefID != created.RefID ||
		result.Report.ReplacedCount != 0 ||
		result.Report.ResponsesChainDisabled {
		t.Fatalf("report = %#v, want liveness context ref without active-context requirement", result.Report)
	}
	if store.createCalls != 0 || store.materializeCalls != 0 {
		t.Fatalf("raw output materialization calls = create %d legacy %d, want none for runtime placeholder", store.createCalls, store.materializeCalls)
	}
}

func TestProjectMCPRuntimePlaceholderRejectsCopiedRawOutputRefFromDifferentToolCall(t *testing.T) {
	store := providerHistoryTestRawOutputStore(t)
	created, err := store.Create(context.Background(), rawoutputs.CreateRequest{
		Surface:   rawoutputs.SurfaceMCPToolResult,
		SessionID: "session-mcp-runtime-placeholder-spoof",
		Source: rawoutputs.SourceMetadata{
			ToolName:   "mcp_context7_get_library_docs",
			ToolCallID: "call_mcp_other",
			EventID:    "tool_call:call_mcp_other",
		},
		Classification: rawoutputs.ClassificationMetadata{
			SemanticRole: "data_bearing",
			Family:       "mcp",
			Classifier:   "mcp_runtime_large_result",
		},
		Body:          strings.NewReader(providerHistoryTestLargeSafeMCPResult()),
		SizeHintBytes: int64(len(providerHistoryTestLargeSafeMCPResult())),
	})
	if err != nil {
		t.Fatalf("Create(runtime raw output) error = %v", err)
	}
	history := providerHistoryTestMCPHistory("call_mcp_current", providerHistoryTestMCPRuntimePlaceholder(created.Ref.RefID))

	result := Project(ProjectionInput{
		Messages: history,
		Policy: Policy{
			Mode:                             Apply,
			RawOutputArtifactsMode:           RawOutputArtifactsApply,
			RawOutputArtifactStore:           store,
			SessionID:                        "session-mcp-runtime-placeholder-spoof",
			RawOutputRehydrateContextEnabled: true,
			ActiveContextTransportAvailable:  true,
		},
	})

	candidate := providerHistoryTestCandidateByToolCallID(result.Report, "call_mcp_current")
	if candidate == nil ||
		!candidate.RawOutputContextRequired ||
		candidate.ReplacementApplied ||
		candidate.FailClosedReason != rawOutputRefSourceMismatchReason {
		t.Fatalf("runtime MCP copied-ref candidate = %#v, want source mismatch fail-closed", candidate)
	}
	if result.Report.RawOutputContextRefCount != 0 ||
		len(result.Report.RawOutputContextMissingRefIDs) != 1 ||
		result.Report.RawOutputContextMissingRefIDs[0] != created.Ref.RefID ||
		result.Report.ReplacedCount != 0 ||
		result.Report.ResponsesChainDisabled {
		t.Fatalf("report = %#v, want copied ref rejected without projection replacement", result.Report)
	}
}

func TestProjectMCPRuntimePlaceholderReportsMissingContextRefWithoutReplacement(t *testing.T) {
	history := providerHistoryTestMCPHistory("call_mcp_runtime_missing", providerHistoryTestMCPRuntimePlaceholder("rawout_missing"))

	result := Project(ProjectionInput{
		Messages: history,
		Policy: Policy{
			Mode:                             Apply,
			RawOutputArtifactsMode:           RawOutputArtifactsApply,
			RawOutputArtifactStore:           providerHistoryTestRawOutputStore(t),
			SessionID:                        "session-mcp-runtime-missing",
			RawOutputRehydrateContextEnabled: true,
			ActiveContextTransportAvailable:  true,
		},
	})

	candidate := providerHistoryTestCandidateByToolCallID(result.Report, "call_mcp_runtime_missing")
	if candidate == nil ||
		!candidate.RawOutputContextRequired ||
		candidate.ReplacementApplied ||
		candidate.FailClosedReason != string(rawoutputs.ReasonArtifactMissing) {
		t.Fatalf("runtime MCP missing-ref candidate = %#v, want fail-closed missing ref", candidate)
	}
	if result.Report.RawOutputContextRefCount != 0 ||
		len(result.Report.RawOutputContextMissingRefIDs) != 1 ||
		result.Report.RawOutputContextMissingRefIDs[0] != "rawout_missing" ||
		result.Report.ReplacedCount != 0 ||
		result.Report.ResponsesChainDisabled {
		t.Fatalf("report = %#v, want missing context ref without replacement", result.Report)
	}
}

func TestProjectApplyCompactsArtifactBackedMCPToolResultWithRefAndRehydrateGate(t *testing.T) {
	content := providerHistoryTestLargeSafeMCPResult()
	history := providerHistoryTestMCPHistory("call_mcp_docs", content)

	result := Project(ProjectionInput{
		Messages: history,
		Policy: Policy{
			Mode:                             Apply,
			RawOutputArtifactsMode:           RawOutputArtifactsApply,
			RawOutputArtifactStore:           providerHistoryTestRawOutputStore(t),
			SessionID:                        "session-mcp-apply",
			RawOutputRehydrateContextEnabled: true,
			ActiveContextTransportAvailable:  true,
		},
	})

	projected := result.History[1].Content
	if projected == content ||
		!strings.Contains(projected, "[compacted old MCP tool result;") ||
		!strings.Contains(projected, "raw_output_ref=") {
		t.Fatalf("projected MCP result = %q, want artifact-backed placeholder", projected)
	}
	candidate := providerHistoryTestCandidateByToolCallID(result.Report, "call_mcp_docs")
	if candidate == nil ||
		!candidate.ArtifactBackedCandidate ||
		!candidate.ArtifactBackedApplyEligible ||
		!candidate.ReplacementApplied ||
		candidate.ArtifactBackedActualSavedBytes <= 0 ||
		candidate.ApproxArtifactBackedActualSavedTokens < providerHistoryRawOutputArtifactMinSavedTokens {
		t.Fatalf("MCP candidate = %#v, want applied artifact-backed replacement", candidate)
	}
	if result.Report.ReplacedCount != 1 ||
		result.Report.ArtifactBackedActualSavedBytes != candidate.ArtifactBackedActualSavedBytes ||
		result.Report.EstimatedSavedBytes != candidate.ArtifactBackedActualSavedBytes ||
		!result.Report.ResponsesChainDisabled {
		t.Fatalf("report = %#v, want MCP artifact actual savings and response chain disable", result.Report)
	}
}

func providerHistoryTestMCPRuntimePlaceholder(refID string) string {
	return "[compacted MCP tool result;\n" +
		" surface=mcp_tool_result;\n" +
		" bytes=100000;\n" +
		" runes=100000;\n" +
		" sha256=sha256:aaaaaaaaaaaa;\n" +
		" raw_output_ref=" + refID + ";\n" +
		" artifact_bytes=100000;\n" +
		"]\n" +
		"excerpt:\n" +
		"safe documentation result placeholder"
}

func providerHistoryTestCreateMCPRuntimeRawOutputRef(t *testing.T, store *rawoutputs.Store, sessionID, callID string) rawoutputs.RawOutputRef {
	t.Helper()
	body := providerHistoryTestLargeSafeMCPResult()
	created, err := store.Create(context.Background(), rawoutputs.CreateRequest{
		Surface:   rawoutputs.SurfaceMCPToolResult,
		SessionID: sessionID,
		Source: rawoutputs.SourceMetadata{
			ToolName:   "mcp_context7_get_library_docs",
			ToolCallID: callID,
			EventID:    "tool_call:" + callID,
		},
		Classification: rawoutputs.ClassificationMetadata{
			SemanticRole: "data_bearing",
			Family:       "mcp",
			Classifier:   "mcp_runtime_large_result",
		},
		Body:          strings.NewReader(body),
		SizeHintBytes: int64(len(body)),
	})
	if err != nil {
		t.Fatalf("Create(runtime raw output) error = %v", err)
	}
	return created.Ref
}

func assertProviderHistoryMCPRuntimePlaceholderContextRef(t *testing.T, result ProjectionResult, history []api.Message, refID, callID string) {
	t.Helper()
	if !reflect.DeepEqual(result.History, history) {
		t.Fatalf("runtime MCP placeholder projection changed payload:\n got %#v\nwant %#v", result.History, history)
	}
	candidate := providerHistoryTestCandidateByToolCallID(result.Report, callID)
	if candidate == nil ||
		!candidate.RawOutputContextRequired ||
		candidate.ArtifactBackedCandidate ||
		candidate.ReplacementApplied ||
		candidate.RawOutputRefID != refID ||
		candidate.KeepReason != mcpRuntimeCompactedResultKeepReason {
		t.Fatalf("runtime MCP placeholder candidate = %#v, want context-only keep", candidate)
	}
	if result.Report.RawOutputRefCount != 0 ||
		result.Report.RawOutputContextRefCount != 1 ||
		result.Report.RawOutputContextRefs[0].RefID != refID ||
		result.Report.ReplacedCount != 0 ||
		result.Report.ResponsesChainDisabled {
		t.Fatalf("report = %#v, want context ref without projection replacement", result.Report)
	}
}
