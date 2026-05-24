package agent

import "github.com/susugadx/xelyon-cli/internal/ledger"

type providerHistoryEvidencePointerKey struct {
	path       string
	startLine  int
	endLine    int
	source     string
	toolCallID string
}

func providerHistoryAppliedEvidencePointers(report ProviderHistoryProjectionReport) []ledger.EvidencePointer {
	if len(report.Candidates) == 0 {
		return nil
	}
	pointers := make([]ledger.EvidencePointer, 0, len(report.Candidates))
	seen := make(map[providerHistoryEvidencePointerKey]struct{})
	for _, candidate := range report.Candidates {
		if !candidate.ReplacementApplied || len(candidate.EvidencePointers) == 0 || !isProviderHistoryReductionCandidateTool(candidate.ToolName) {
			continue
		}
		for _, pointer := range candidate.EvidencePointers {
			key := providerHistoryEvidencePointerKeyForPointer(pointer)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			pointers = append(pointers, pointer)
		}
	}
	return cloneProviderHistoryReductionEvidencePointers(pointers)
}

func providerHistoryEvidencePointerKeyForPointer(pointer ledger.EvidencePointer) providerHistoryEvidencePointerKey {
	return providerHistoryEvidencePointerKey{
		path:       pointer.Path,
		startLine:  pointer.StartLine,
		endLine:    pointer.EndLine,
		source:     pointer.Source,
		toolCallID: pointer.ToolCallID,
	}
}

func (a *Agent) buildProviderHistoryRehydratePlan(targetPaths []string) ledger.RehydratePlan {
	if a == nil || a.Runtime == nil || a.Runtime.TaskLedger == nil {
		return ledger.RehydratePlan{}
	}
	oldEvidence := providerHistoryAppliedEvidencePointers(a.Runtime.LastProviderHistoryProjectionReport)
	if len(oldEvidence) == 0 {
		return ledger.RehydratePlan{}
	}
	opts := ledger.RehydratePlanOptions{
		OldEvidencePointers: oldEvidence,
	}
	if len(targetPaths) > 0 {
		opts.TargetPaths = append([]string(nil), targetPaths...)
	}
	return a.Runtime.TaskLedger.BuildRehydratePlan(opts)
}
