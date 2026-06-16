package providerhistory

import (
	"reflect"
	"strings"
	"testing"

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

func TestProjectApplyKeepsSensitiveLargeMCPToolResultCandidateOnly(t *testing.T) {
	content := providerHistoryTestLargeSensitiveMCPResult()
	history := providerHistoryTestMCPHistory("call_mcp_sensitive", content)

	result := Project(ProjectionInput{
		Messages: history,
		Policy: Policy{
			Mode:                             Apply,
			RawOutputArtifactsMode:           RawOutputArtifactsApply,
			RawOutputArtifactStore:           providerHistoryTestRawOutputStore(t),
			SessionID:                        "session-mcp-sensitive",
			RawOutputRehydrateContextEnabled: true,
			ActiveContextTransportAvailable:  true,
		},
	})

	if !reflect.DeepEqual(result.History, history) {
		t.Fatalf("sensitive MCP projection changed payload:\n got %#v\nwant %#v", result.History, history)
	}
	candidate := providerHistoryTestCandidateByToolCallID(result.Report, "call_mcp_sensitive")
	if candidate == nil ||
		!candidate.CandidateOnly ||
		!candidate.FutureApplyCandidate ||
		candidate.ArtifactBackedCandidate ||
		candidate.RawOutputRefID != "" ||
		candidate.ReplacementApplied ||
		candidate.KeepReason != "mcp_sensitive_or_private_result_keep" {
		t.Fatalf("MCP sensitive candidate = %#v, want candidate-only sensitive keep", candidate)
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
		t.Fatalf("FutureFamilyKeptReasonCounts = %#v, want sensitive keep count", result.Report.FutureFamilyKeptReasonCounts)
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
