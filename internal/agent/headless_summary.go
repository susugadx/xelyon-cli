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

func buildHeadlessSummary(agent *Agent, gitBaseline headlessGitChangedFilesBaseline, commands []HeadlessCommandSummary, finalChecks []HeadlessFinalCheckSummary) *HeadlessSummary {
	summary := HeadlessSummary{
		ChangedFiles: headlessSummaryChangedFilesWithGitBaseline(agent, gitBaseline),
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
	result.Summary = buildHeadlessSummary(r.agent, r.gitChangedFilesBaseline, r.commands, r.finalChecks)
	return result
}

func (r *headlessRunner) runFinalChecksIfNeeded(ctx context.Context) {
	if r == nil || r.agent == nil {
		return
	}
	changedFiles := headlessSummaryChangedFilesWithGitBaseline(r.agent, r.gitChangedFilesBaseline)
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
	return headlessSummaryChangedFilesWithGitBaseline(agent, headlessGitChangedFilesBaseline{})
}

func headlessSummaryChangedFilesWithGitBaseline(agent *Agent, gitBaseline headlessGitChangedFilesBaseline) []string {
	if agent == nil || agent.Runtime == nil || agent.Runtime.TaskLedger == nil {
		return nil
	}
	ledgerFiles := agent.Runtime.TaskLedger.Snapshot().ChangedFiles.Paths()
	if agent.Runtime.Options.ReadOnly {
		return ledgerFiles
	}
	gitFiles, ok := headlessGitChangedFilesSinceBaseline(gitBaseline)
	if !ok {
		return ledgerFiles
	}
	return mergeHeadlessChangedFiles(ledgerFiles, gitFiles)
}

func mergeHeadlessChangedFiles(groups ...[]string) []string {
	seen := make(map[string]struct{})
	var merged []string
	for _, group := range groups {
		for _, path := range group {
			path = strings.TrimSpace(path)
			if path == "" {
				continue
			}
			if _, ok := seen[path]; ok {
				continue
			}
			seen[path] = struct{}{}
			merged = append(merged, path)
		}
	}
	return merged
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
