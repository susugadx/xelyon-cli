package agent

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/finalcheck"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

const (
	headlessSummaryStatusPassed = "passed"
	headlessSummaryStatusFailed = "failed"
	headlessCommandSourceTool   = "tool"
)

var headlessCommandExitStatusRe = regexp.MustCompile(`(?m)^Error:\s+exit status\s+(-?\d+)`)

func buildHeadlessSummary(agent *Agent, commands []HeadlessCommandSummary, finalChecks []HeadlessFinalCheckSummary) *HeadlessSummary {
	summary := HeadlessSummary{
		ChangedFiles: headlessSummaryChangedFiles(agent),
		Commands:     cloneHeadlessCommandSummaries(commands),
		FinalChecks:  cloneHeadlessFinalCheckSummaries(finalChecks),
	}
	if len(summary.ChangedFiles) == 0 && len(summary.Commands) == 0 && len(summary.FinalChecks) == 0 {
		return nil
	}
	return &summary
}

func (r *headlessRunner) attachSummary(result *HeadlessResult) *HeadlessResult {
	if r == nil || result == nil {
		return result
	}
	result.Summary = buildHeadlessSummary(r.agent, r.commands, r.finalChecks)
	return result
}

func (r *headlessRunner) runFinalChecksIfNeeded(ctx context.Context) {
	if r == nil || r.agent == nil {
		return
	}
	changedFiles := headlessSummaryChangedFiles(r.agent)
	if len(changedFiles) == 0 || len(r.agent.cfg().FinalChecks.Commands) == 0 {
		return
	}
	result := r.agent.runFinalCheckCommands(ctx, changedFiles)
	r.finalChecks = append(r.finalChecks, headlessFinalCheckSummariesFromRunResult(result)...)
	if result.Cancelled {
		r.cancelledErr = result.Err
		return
	}
	if result.NeedsContinue {
		r.finalCheckFailed = true
	}
}

func headlessSummaryChangedFiles(agent *Agent) []string {
	if agent == nil || agent.Runtime == nil || agent.Runtime.TaskLedger == nil {
		return nil
	}
	return agent.Runtime.TaskLedger.Snapshot().ChangedFiles.Paths()
}

func newHeadlessCommandSummary(toolCall *tools.ToolCall, execResult tools.ExecutionResult) (HeadlessCommandSummary, bool) {
	if toolCall == nil || toolCall.Tool != "bash" {
		return HeadlessCommandSummary{}, false
	}
	command := strings.TrimSpace(toolCall.Args["command"])
	if command == "" {
		return HeadlessCommandSummary{}, false
	}
	classification := classifyHeadlessCommandResult(execResult.Result, execResult.Error)
	return HeadlessCommandSummary{
		Command:  command,
		ExitCode: classification.exitCode,
		Status:   classification.status,
		Source:   headlessCommandSourceTool,
	}, true
}

type headlessCommandClassification struct {
	exitCode int
	status   string
}

func classifyHeadlessCommandResult(output string, toolError bool) headlessCommandClassification {
	if match := headlessCommandExitStatusRe.FindStringSubmatch(output); match != nil {
		exitCode, _ := strconv.Atoi(match[1])
		return headlessCommandClassification{exitCode: exitCode, status: headlessSummaryStatusFailed}
	}
	if toolError || tools.IsErrorResult(output) || headlessCommandOutputWasCancelled(output) {
		return headlessCommandClassification{exitCode: -1, status: headlessSummaryStatusFailed}
	}
	return headlessCommandClassification{exitCode: 0, status: headlessSummaryStatusPassed}
}

func headlessCommandOutputWasCancelled(output string) bool {
	normalized := strings.ToLower(strings.TrimSpace(output))
	return strings.Contains(normalized, "cancelled by user") ||
		strings.Contains(normalized, "command interrupted") ||
		strings.HasPrefix(normalized, "[cancelled]")
}

func headlessFinalCheckSummariesFromRunResult(result finalcheck.RunResult) []HeadlessFinalCheckSummary {
	if len(result.Checks) == 0 {
		return nil
	}
	summaries := make([]HeadlessFinalCheckSummary, 0, len(result.Checks))
	for _, check := range result.Checks {
		command := strings.TrimSpace(check.Command)
		if command == "" {
			continue
		}
		status := strings.TrimSpace(check.Status)
		if status == "" {
			status = headlessSummaryStatusPassed
			if check.ExitCode != 0 {
				status = headlessSummaryStatusFailed
			}
		}
		summaries = append(summaries, HeadlessFinalCheckSummary{
			Command:  command,
			ExitCode: check.ExitCode,
			Status:   status,
		})
	}
	return summaries
}

func cloneHeadlessCommandSummaries(items []HeadlessCommandSummary) []HeadlessCommandSummary {
	if len(items) == 0 {
		return nil
	}
	cloned := make([]HeadlessCommandSummary, len(items))
	copy(cloned, items)
	return cloned
}

func cloneHeadlessFinalCheckSummaries(items []HeadlessFinalCheckSummary) []HeadlessFinalCheckSummary {
	if len(items) == 0 {
		return nil
	}
	cloned := make([]HeadlessFinalCheckSummary, len(items))
	copy(cloned, items)
	return cloned
}
