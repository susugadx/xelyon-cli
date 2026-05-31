package agent

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/taskstate"
	"github.com/susugadx/xelyon-cli/internal/token"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

const providerHistoryReductionPlaceholderPrefix = "[omitted old "

const (
	providerHistoryCommandReplacementMinSavedTokens = 128
	providerHistoryEditArgReplacementMinSavedTokens = 128
)

func candidateTools(report ProviderHistoryProjectionReport) []string {
	tools := make([]string, len(report.Candidates))
	for i, candidate := range report.Candidates {
		tools[i] = candidate.ToolName
	}
	return tools
}

func candidateToolCallIDs(report ProviderHistoryProjectionReport) []string {
	ids := make([]string, len(report.Candidates))
	for i, candidate := range report.Candidates {
		ids[i] = candidate.ToolCallID
	}
	return ids
}

func keptByToolCallID(report ProviderHistoryProjectionReport, id string) *ProviderHistoryReductionCandidate {
	for i := range report.Kept {
		if report.Kept[i].ToolCallID == id {
			return &report.Kept[i]
		}
	}
	return nil
}

func assertKeepReason(t *testing.T, report ProviderHistoryProjectionReport, id, want string) {
	t.Helper()
	entry := keptByToolCallID(report, id)
	if entry == nil || entry.KeepReason != want {
		t.Fatalf("kept entry for %q = %#v, want keep reason %q", id, entry, want)
	}
}

func candidateByToolCallID(report ProviderHistoryProjectionReport, id string) *ProviderHistoryReductionCandidate {
	for i := range report.Candidates {
		if report.Candidates[i].ToolCallID == id {
			return &report.Candidates[i]
		}
	}
	return nil
}

func countKeptByToolCallIDAndReason(report ProviderHistoryProjectionReport, id, reason string) int {
	count := 0
	for _, kept := range report.Kept {
		if kept.ToolCallID == id && kept.KeepReason == reason {
			count++
		}
	}
	return count
}

func assertProviderHistoryByteMetrics(t *testing.T, original, projected []api.Message, report ProviderHistoryProjectionReport) {
	t.Helper()
	originalBytes := providerHistoryContentBytes(original)
	projectedBytes := providerHistoryContentBytes(projected)
	if report.OriginalBytes != originalBytes || report.ProjectedBytes != projectedBytes {
		t.Fatalf("byte metrics = original %d projected %d, want %d/%d", report.OriginalBytes, report.ProjectedBytes, originalBytes, projectedBytes)
	}
	wantSaved := 0
	if originalBytes > projectedBytes {
		wantSaved = originalBytes - projectedBytes
	}
	if report.EstimatedSavedBytes != wantSaved {
		t.Fatalf("EstimatedSavedBytes = %d, want %d", report.EstimatedSavedBytes, wantSaved)
	}
	wantSavedTokens := providerHistoryApproxSavedTokens(original, projected)
	if report.ApproxSavedTokens != wantSavedTokens {
		t.Fatalf("ApproxSavedTokens = %d, want %d", report.ApproxSavedTokens, wantSavedTokens)
	}
}

func providerHistoryContentBytes(messages []api.Message) int {
	total := 0
	for _, msg := range messages {
		total += len(msg.Content)
	}
	return total
}

func providerHistoryApproxSavedTokens(original, projected []api.Message) int {
	originalTokens := providerHistoryContentTokens(original)
	projectedTokens := providerHistoryContentTokens(projected)
	if originalTokens <= projectedTokens {
		return 0
	}
	return originalTokens - projectedTokens
}

func providerHistoryContentTokens(messages []api.Message) int {
	total := 0
	for _, msg := range messages {
		total += token.EstimateTokenCount(msg.Content)
	}
	return total
}

func assertLastProviderHistoryProjectionReportPreserved(t *testing.T, runtime *AgentRuntime, want ProviderHistoryProjectionReport) {
	t.Helper()
	if runtime == nil {
		t.Fatalf("runtime is nil, want preserved LastProviderHistoryProjectionReport %#v", want)
	}
	if !reflect.DeepEqual(runtime.LastProviderHistoryProjectionReport, want) {
		t.Fatalf("LastProviderHistoryProjectionReport = %#v, want preserved stale report %#v", runtime.LastProviderHistoryProjectionReport, want)
	}
}

func providerHistoryMessagesContainReductionPlaceholder(messages []api.Message) bool {
	for _, msg := range messages {
		if strings.Contains(msg.Content, providerHistoryReductionPlaceholderPrefix) {
			return true
		}
	}
	return false
}

func providerHistoryInputItemsContainReductionPlaceholder(items []api.InputItem) bool {
	for _, item := range items {
		if strings.Contains(item.Output, providerHistoryReductionPlaceholderPrefix) ||
			strings.Contains(item.Data, providerHistoryReductionPlaceholderPrefix) ||
			strings.Contains(item.Arguments, providerHistoryReductionPlaceholderPrefix) {
			return true
		}
		if content, ok := item.Content.(string); ok && strings.Contains(content, providerHistoryReductionPlaceholderPrefix) {
			return true
		}
	}
	return false
}

type providerHistoryEvidenceItem struct {
	ToolName   string
	ToolCallID string
	Path       string
	StartLine  int
	EndLine    int
	Excerpt    string
}

func providerHistoryTaskLedgerWithEvidence(t *testing.T, items ...providerHistoryEvidenceItem) *taskstate.Store {
	t.Helper()
	root := t.TempDir()
	store := taskstate.NewStoreWithRoot(root)
	for _, item := range items {
		endLine := item.EndLine
		if endLine == 0 {
			endLine = item.StartLine
		}
		excerpt := item.Excerpt
		if excerpt == "" {
			excerpt = "evidence"
		}
		store.Recorder().RecordToolObservation(taskstate.ToolObservation{
			ToolName:   item.ToolName,
			ToolCallID: item.ToolCallID,
			Structured: &tools.RuntimeObservation{
				Evidence: []tools.ObservationEvidence{{
					Path:         item.Path,
					ResolvedPath: filepath.Join(root, filepath.FromSlash(item.Path)),
					StartLine:    item.StartLine,
					EndLine:      endLine,
					Excerpt:      excerpt,
				}},
			},
		})
	}
	return store
}
