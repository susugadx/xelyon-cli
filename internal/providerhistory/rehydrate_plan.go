package providerhistory

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/taskstate"
)

// RehydratedEvidenceActiveContextName は再水和した evidence を provider へ渡す active context 名。
const RehydratedEvidenceActiveContextName = "provider_history_rehydrated_evidence"

type providerHistoryEvidencePointerKey struct {
	path       string
	startLine  int
	endLine    int
	source     string
	toolCallID string
}

// AppliedEvidencePointers は実際に置換された provider history 候補に対応する evidence pointer を返す。
func AppliedEvidencePointers(report ProjectionReport) []taskstate.EvidencePointer {
	if len(report.Candidates) == 0 {
		return nil
	}
	pointers := make([]taskstate.EvidencePointer, 0, len(report.Candidates))
	seen := make(map[providerHistoryEvidencePointerKey]struct{})
	for _, candidate := range report.Candidates {
		if !candidate.ReplacementApplied || len(candidate.EvidencePointers) == 0 || !isReductionCandidateTool(candidate.ToolName) {
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

func providerHistoryEvidencePointerKeyForPointer(pointer taskstate.EvidencePointer) providerHistoryEvidencePointerKey {
	return providerHistoryEvidencePointerKey{
		path:       pointer.Path,
		startLine:  pointer.StartLine,
		endLine:    pointer.EndLine,
		source:     pointer.Source,
		toolCallID: pointer.ToolCallID,
	}
}

// BuildRehydratePlan は projection report から request-local evidence rehydrate plan を組み立てる。
func BuildRehydratePlan(store *taskstate.Store, report ProjectionReport, targetPaths []string) taskstate.RehydratePlan {
	if store == nil {
		return taskstate.RehydratePlan{}
	}
	oldEvidence := AppliedEvidencePointers(report)
	if len(oldEvidence) == 0 {
		return taskstate.RehydratePlan{}
	}
	opts := taskstate.RehydratePlanOptions{
		OldEvidencePointers: oldEvidence,
	}
	if len(targetPaths) > 0 {
		opts.TargetPaths = append([]string(nil), targetPaths...)
	}
	return store.BuildRehydratePlan(opts)
}

// RehydratedEvidenceActiveContextBlock は rehydrate 実行結果を active context block へ変換する。
func RehydratedEvidenceActiveContextBlock(block taskstate.RehydratedEvidenceBlock) (api.ActiveContextBlock, bool) {
	content := taskstate.RenderRehydratedEvidenceBlock(block)
	if strings.TrimSpace(content) == "" {
		return api.ActiveContextBlock{}, false
	}
	return api.ActiveContextBlock{
		Name:    RehydratedEvidenceActiveContextName,
		Content: content,
	}, true
}
