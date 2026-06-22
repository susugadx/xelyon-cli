package providerhistory

import (
	"context"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
	"reflect"
	"strings"
	"testing"
)

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

func TestProjectMCPRuntimePlaceholderKeepsOmittedReasonWithoutRawOutputRef(t *testing.T) {
	placeholder := "[compacted MCP tool result;\n" +
		" surface=mcp_tool_result;\n" +
		" bytes=100000;\n" +
		" runes=100000;\n" +
		" sha256=sha256:aaaaaaaaaaaa;\n" +
		" full_output_omitted_reason=raw_output_artifacts_dry_run;\n" +
		"]\n" +
		"excerpt:\n" +
		"safe dry-run runtime MCP excerpt"
	history := providerHistoryTestMCPHistory("call_mcp_runtime_omitted", placeholder)

	result := Project(ProjectionInput{
		Messages: history,
		Policy: Policy{
			Mode:                             Apply,
			RawOutputArtifactsMode:           RawOutputArtifactsApply,
			RawOutputArtifactStore:           providerHistoryTestRawOutputStore(t),
			SessionID:                        "session-mcp-runtime-omitted",
			RawOutputRehydrateContextEnabled: true,
			ActiveContextTransportAvailable:  true,
		},
	})

	if !reflect.DeepEqual(result.History, history) {
		t.Fatalf("runtime MCP omitted placeholder projection changed payload:\n got %#v\nwant %#v", result.History, history)
	}
	candidate := providerHistoryTestCandidateByToolCallID(result.Report, "call_mcp_runtime_omitted")
	if candidate == nil ||
		candidate.RawOutputContextRequired ||
		candidate.ReplacementApplied ||
		candidate.RawOutputRefID != "" ||
		candidate.KeepReason != "raw_output_artifacts_dry_run" ||
		candidate.FailClosedReason != "" {
		t.Fatalf("runtime MCP omitted placeholder candidate = %#v, want keep by omitted reason", candidate)
	}
	if result.Report.RawOutputContextRefCount != 0 ||
		len(result.Report.RawOutputContextMissingRefIDs) != 0 ||
		result.Report.ReplacedCount != 0 ||
		result.Report.ResponsesChainDisabled {
		t.Fatalf("report = %#v, want omitted placeholder keep without missing raw ref", result.Report)
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
