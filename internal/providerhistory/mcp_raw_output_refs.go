package providerhistory

import (
	"context"
	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
	"strings"
)

type rawOutputRefLookupStore interface {
	LookupRef(ctx context.Context, sessionID, refID string) (rawoutputs.RawOutputRef, error)
}

func providerHistoryRuntimeMCPRawOutputRefMatchesEntry(ref rawoutputs.RawOutputRef, entry ReductionCandidate) bool {
	if strings.TrimSpace(entry.ToolCallID) == "" || strings.TrimSpace(entry.ToolName) == "" {
		return false
	}
	return ref.ToolCallID == entry.ToolCallID && ref.ToolName == entry.ToolName
}

func providerHistoryLookupRuntimeMCPRawOutputRef(policy Policy, refID string) (rawoutputs.RawOutputRef, string, bool) {
	if strings.TrimSpace(policy.SessionID) == "" {
		return rawoutputs.RawOutputRef{}, "raw_output_ref_missing", false
	}
	store, ok := policy.RawOutputArtifactStore.(rawOutputRefLookupStore)
	if !ok || store == nil {
		return rawoutputs.RawOutputRef{}, "raw_output_artifact_missing", false
	}
	ref, err := store.LookupRef(context.Background(), policy.SessionID, refID)
	if err != nil {
		return rawoutputs.RawOutputRef{}, providerHistoryRawOutputLookupFailureReason(err), false
	}
	return ref, "", true
}

func providerHistoryRawOutputLookupFailureReason(err error) string {
	if reason := rawoutputs.ReasonOf(err); reason != "" {
		return string(reason)
	}
	return "raw_output_artifact_missing"
}
