package agent

import (
	"fmt"
	"io"
	"strconv"

	"github.com/susugadx/xelyon-cli/internal/agent/viewfmt"
	"github.com/susugadx/xelyon-cli/internal/termtext"
)

func renderToolObservabilitySection(out io.Writer, stats *SessionStats) {
	obs := stats.ToolObs

	_, _ = fmt.Fprintln(out)
	green.Fprintln(out, "📈 Tool Selection")
	selTable := termtext.NewTable()
	selTable.AddRow("read_file(batch)", strconv.Itoa(obs.ReadFileBatchCalls))
	selTable.AddRow("search_code(multi)", strconv.Itoa(obs.SearchCodeMultiPatternCalls))
	selTable.AddRow("search_code(missed multi)", strconv.Itoa(obs.SearchCodeMissedMultiPattern))
	_, _ = fmt.Fprint(out, selTable.RenderCompact())

	_, _ = fmt.Fprintln(out)
	green.Fprintln(out, "📍 Exploration")
	explorationTable := termtext.NewTable()
	explorationTable.AddRow("search_code(impact)", strconv.Itoa(obs.SearchCodeImpactCalls))
	explorationTable.AddRow("search_code(explicit multi)", strconv.Itoa(obs.SearchCodeExplicitMultiCalls))
	explorationTable.AddRow("read_file(targets)", strconv.Itoa(obs.ReadFileTargetCalls))
	explorationTable.AddRow("search_code(batch merges)", strconv.Itoa(obs.SearchCodeBatchMerges))
	explorationTable.AddRow("read_file(batch merges)", strconv.Itoa(obs.ReadFileBatchMerges))
	_, _ = fmt.Fprint(out, explorationTable.RenderCompact())

	if obs.ApplyPatchAttempts > 0 || obs.ApplyPatchRepairAttempts > 0 {
		_, _ = fmt.Fprintln(out)
		green.Fprintln(out, "🩹 Patch Reliability")
		patchTable := termtext.NewTable()
		patchTable.AddRow("apply_patch(success)", fmt.Sprintf("%d/%d", obs.ApplyPatchSuccesses, obs.ApplyPatchAttempts))
		patchTable.AddRow("apply_patch(repair)", fmt.Sprintf("%d/%d", obs.ApplyPatchRepairSuccesses, obs.ApplyPatchRepairAttempts))
		_, _ = fmt.Fprint(out, patchTable.RenderCompact())
	}
}

func renderSavingsSection(out io.Writer, stats *SessionStats) {
	sav := stats.Savings
	if !sav.hasAny() {
		return
	}

	_, _ = fmt.Fprintln(out)
	green.Fprintln(out, "💰 Estimated Savings (API input)")
	savTable := termtext.NewTable()
	if sav.SavedCalls > 0 {
		savTable.AddRow("Executions skipped", strconv.Itoa(sav.SavedCalls))
	}
	if sav.EstimatedInputTokensSaved > 0 {
		savTable.AddRow("~Input tokens saved", fmt.Sprintf("~%s", formatNumber(sav.EstimatedInputTokensSaved)))
	}
	if sav.EstimatedCostSaved > 0 {
		savTable.AddRow("~Cost saved", fmt.Sprintf("~%s", viewfmt.USDWithSuffix(sav.EstimatedCostSaved)))
	}
	_, _ = fmt.Fprint(out, savTable.RenderCompact())
	dim.Fprintln(out, "  (~ = estimated, dedup result diff + compaction)")
}
