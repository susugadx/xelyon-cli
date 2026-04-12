package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/subagent"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

var subAgentEventLinePattern = regexp.MustCompile(`lines (\d+)-\d+`)

func (a *Agent) executeWaitAgentWithLiveView(ctx context.Context, toolCall *tools.ToolCall) (string, *tools.FileChange) {
	manager := a.subAgentManager()
	if manager == nil {
		return tools.ExecuteWithContext(a.toolExecutionContext(ctx, nil, nil, nil), toolCall)
	}

	out := a.output()
	eventCh := manager.Events()
	var toolOut bytes.Buffer
	execCtx := a.toolExecutionContext(ctx, nil, &toolOut, &toolOut)

	type waitResult struct {
		output  string
		change  *tools.FileChange
		display string
	}

	resultCh := make(chan waitResult, 1)
	go func() {
		output, change := tools.ExecuteWithContext(execCtx, toolCall)
		resultCh <- waitResult{
			output:  output,
			change:  change,
			display: toolOut.String(),
		}
	}()

	_, _ = fmt.Fprintln(out)

	for {
		select {
		case event, ok := <-eventCh:
			if ok {
				// スピナー行をクリアしてからイベントを表示（スピナーとの混在防止）
				fmt.Fprint(out, "\r\033[K")
				printSubAgentEvent(out, event)
			}
		case result := <-resultCh:
			fmt.Fprint(out, "\r\033[K")
			drainEvents(out, eventCh)
			if result.display != "" {
				_, _ = io.WriteString(out, result.display)
			}
			a.recordToolResultOptimizations(toolCall, result.output)
			if a.ToolCache != nil {
				a.ToolCache.SetNegativeCache(toolCall.Tool, toolCall.RawArgs, result.output)
			}
			return result.output, result.change
		}
	}
}

// drainEvents はチャネルに残っているイベントを全て表示する。
func drainEvents(out io.Writer, ch <-chan subagent.SubAgentEvent) {
	if ch == nil {
		return
	}
	for {
		select {
		case event, ok := <-ch:
			if !ok {
				return
			}
			printSubAgentEvent(out, event)
		default:
			return
		}
	}
}

// printSubAgentEvent はサブエージェントのイベントを表示する。
func printSubAgentEvent(out io.Writer, event subagent.SubAgentEvent) {
	if out == nil {
		return
	}

	agentID := event.AgentID
	if agentID == "" {
		agentID = "sub-???"
	}
	prefix := dim.Sprintf("  %s │ ", agentID)

	switch {
	case event.Tool == "_completed":
		icon := green.Sprint("✅")
		if !event.Success {
			icon = red.Sprint("❌")
		}
		fmt.Fprintf(out, "%s%s %s\n", prefix, icon, event.Output)

	case event.Phase == "start":
		icon := toolStartIcon(event.Tool)
		detail := event.FilePath
		if detail == "" {
			detail = event.Tool
		}
		dim.Fprintf(out, "%s%s %s %s\n", prefix, icon, event.Tool, detail)

	case event.Phase == "end":
		suffix := subAgentEventSuffix(event)
		if event.Tool == "str_replace" && (event.OldStr != "" || event.NewStr != "") {
			if event.Success {
				green.Fprintf(out, "%s✓ %s %s%s\n", prefix, event.Tool, event.FilePath, suffix)
			} else {
				red.Fprintf(out, "%s✗ %s %s%s\n", prefix, event.Tool, event.FilePath, suffix)
			}
			printEventDiff(out, agentID, event)
			return
		}

		icon := green.Sprint("✓")
		if !event.Success {
			icon = red.Sprint("✗")
		}
		detail := event.FilePath
		if detail == "" {
			detail = event.Tool
		}
		fmt.Fprintf(out, "%s%s %s %s%s\n", prefix, icon, event.Tool, detail, suffix)
	}
}

func subAgentEventSuffix(event subagent.SubAgentEvent) string {
	if strings.Contains(event.Output, "normalized whitespace matching") {
		return " (normalized match)"
	}
	return ""
}

// toolStartIcon はツール名に応じた開始アイコンを返す。
func toolStartIcon(tool string) string {
	switch tool {
	case "read_file", "read_files":
		return "📖"
	case "search_code":
		return "🔍"
	case "str_replace":
		return "🔧"
	case "write_file":
		return "📝"
	case "bash":
		return "🔨"
	case "delete_file":
		return "🗑️"
	case "list_dir":
		return "📂"
	default:
		return "⚡"
	}
}

// printEventDiff は str_replace の old_str/new_str から色付き diff を表示する。
func printEventDiff(out io.Writer, agentID string, event subagent.SubAgentEvent) {
	if event.OldStr == "" && event.NewStr == "" {
		return
	}

	var buf bytes.Buffer
	opts := &ui.DiffOptions{
		ContextLines:  1,
		ShowLineNums:  true,
		InlineMode:    true,
		MaxTotalLines: 8,
	}
	if offset := subAgentDiffLineOffset(event.Output); offset > 0 {
		opts.LineNumOffset = offset - 1
	}
	ui.ShowColoredDiffToWriter(&buf, event.OldStr, event.NewStr, opts)

	diffPrefix := dim.Sprintf("  %s │ ", strings.Repeat(" ", len(agentID)))
	scanner := bufio.NewScanner(&buf)
	for scanner.Scan() {
		line := scanner.Text()
		if shouldSkipSubAgentDiffLine(line) {
			continue
		}
		fmt.Fprintf(out, "%s%s\n", diffPrefix, line)
	}
}

func subAgentDiffLineOffset(output string) int {
	matches := subAgentEventLinePattern.FindStringSubmatch(output)
	if len(matches) != 2 {
		return 0
	}
	start := 0
	_, _ = fmt.Sscanf(matches[1], "%d", &start)
	return start
}

func shouldSkipSubAgentDiffLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return true
	}
	if strings.Contains(trimmed, "Diff / 差分表示") {
		return true
	}
	if strings.Contains(trimmed, "─") {
		return true
	}
	if strings.Contains(trimmed, "(net:") {
		return true
	}
	return false
}

// waitAgentSpinnerMessage は wait_agent 用のスピナーメッセージを生成する。
// エージェント数を表示して待機中であることを明示する。
func waitAgentSpinnerMessage(args map[string]string) string {
	if raw := args["ids"]; raw != "" {
		var ids []string
		if err := json.Unmarshal([]byte(raw), &ids); err == nil && len(ids) > 0 {
			return fmt.Sprintf("Waiting for %d agents...", len(ids))
		}
	}
	return "Waiting for agents..."
}

// addToolCallsToHistory はパラレル FC の全ツール呼び出しを1つの assistant メッセージにまとめて履歴に追加する。
// ThoughtSignature/ThoughtParts は最初の ToolCall からのみ取得（Gemini 3 仕様）。
