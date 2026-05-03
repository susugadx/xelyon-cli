package agent

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/toolruntime"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// printParallelToolGroup は非TUIモード向けに並列実行グループの結果を表示する。
func printParallelToolGroup(out io.Writer, cfg *config.Config, allToolCalls []*tools.ToolCall, indices []int, results []toolruntime.Result, elapsed time.Duration) {
	if len(indices) == 0 {
		return
	}

	if len(indices) == 1 {
		idx := indices[0]
		_, _ = fmt.Fprintln(out, ui.FormatToolLine(ui.ToolDisplayInfo{
			ToolName: allToolCalls[idx].Tool,
			Args:     allToolCalls[idx].Args,
			Result:   results[idx].Result,
			Error:    results[idx].Error,
		}))
		printParallelCollapsedOutput(out, cfg, allToolCalls[idx].Tool, results[idx].Result, results[idx].Error)
		return
	}

	ui.PrintParallelGroupStartToWriter(out, len(indices))
	for _, idx := range indices {
		ui.PrintParallelGroupLineToWriter(out, ui.FormatToolLine(ui.ToolDisplayInfo{
			ToolName: allToolCalls[idx].Tool,
			Args:     allToolCalls[idx].Args,
			Result:   results[idx].Result,
			Error:    results[idx].Error,
		}))
		printParallelCollapsedOutputWithPrefix(out, cfg, allToolCalls[idx].Tool, results[idx].Result, results[idx].Error, "│    ")
	}
	ui.PrintParallelGroupEndToWriter(out, formatParallelGroupSummary(allToolCalls, indices, elapsed))
}

func shouldShowParallelCollapsed(toolName, result string, isError bool) bool {
	if isError {
		return true
	}
	if toolName == "search_code" && strings.Contains(result, "\n") {
		return true
	}
	return false
}

func printParallelCollapsedOutput(out io.Writer, cfg *config.Config, toolName, result string, isError bool) {
	if !shouldShowParallelCollapsed(toolName, result, isError) {
		return
	}
	_, _ = fmt.Fprintln(out, ui.FormatToolOutput(result, ui.GetMaxVisibleLinesWithConfig(cfg)))
}

func printParallelCollapsedOutputWithPrefix(out io.Writer, cfg *config.Config, toolName, result string, isError bool, prefix string) {
	if !shouldShowParallelCollapsed(toolName, result, isError) {
		return
	}

	collapsed := ui.FormatToolOutput(result, ui.GetMaxVisibleLinesWithConfig(cfg))
	for _, line := range strings.Split(strings.TrimRight(collapsed, "\n"), "\n") {
		if line == "" {
			continue
		}
		_, _ = fmt.Fprintf(out, "%s%s\n", prefix, line)
	}
}

func formatParallelGroupSummary(allToolCalls []*tools.ToolCall, indices []int, elapsed time.Duration) string {
	counts := make(map[string]int)
	var order []string

	addCount := func(label string) {
		if counts[label] == 0 {
			order = append(order, label)
		}
		counts[label]++
	}

	for _, idx := range indices {
		addCount(parallelGroupSummaryLabel(allToolCalls[idx].Tool))
	}

	var parts []string
	for _, label := range order {
		count := counts[label]
		switch label {
		case "reads", "searches":
			parts = append(parts, fmt.Sprintf("%d %s", count, label))
		case "web":
			parts = append(parts, fmt.Sprintf("%d web", count))
		case "bash":
			parts = append(parts, fmt.Sprintf("%d bash", count))
		default:
			if count == 1 {
				parts = append(parts, fmt.Sprintf("%d %s", count, label))
			} else {
				parts = append(parts, fmt.Sprintf("%d %ss", count, label))
			}
		}
	}

	if len(parts) == 0 {
		return fmt.Sprintf("Done (%s)", ui.FormatParallelElapsed(elapsed))
	}
	return fmt.Sprintf("Done: %s (%s)", strings.Join(parts, ", "), ui.FormatParallelElapsed(elapsed))
}

// sendParallelToolResults は TUI モードで並列実行結果を個別にチャネルへ送信する。
// ToolResultCallback と同じ二重保護（closed チェック + select default）を適用する。
func (a *Agent) sendParallelToolResults(allToolCalls []*tools.ToolCall, indices []int, results []toolruntime.Result, elapsed time.Duration) {
	ch := a.tuiToolResultCh
	for _, idx := range indices {
		if a.tuiToolResultClosed.Load() {
			return
		}
		tc := allToolCalls[idx]
		select {
		case ch <- tools.ToolResultInfo{
			ToolName: tc.Tool,
			Args:     tc.Args,
			Result:   results[idx].Result,
			Error:    results[idx].Error,
			Duration: elapsed,
		}:
		default:
		}
	}
}

func parallelGroupSpinnerMessage(allToolCalls []*tools.ToolCall, indices []int) string {
	if len(indices) == 0 {
		return "Running parallel tools..."
	}

	counts := make(map[string]int)
	for _, idx := range indices {
		counts[parallelSpinnerBucket(allToolCalls[idx].Tool)]++
	}

	switch {
	case counts["reads"] > 0 && len(counts) == 1:
		return "Reading files..."
	case counts["searches"] > 0 && len(counts) == 1:
		return "Searching code..."
	case counts["spawn"] > 0 && len(counts) == 1:
		return fmt.Sprintf("Spawning %d sub-agents...", counts["spawn"])
	case counts["wait"] > 0 && len(counts) == 1:
		return waitAgentSpinnerMessage(allToolCalls[indices[0]].Args)
	default:
		return fmt.Sprintf("Running %d parallel tools...", len(indices))
	}
}
