package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func (a *Agent) assistantUpdateMode() string {
	if a == nil {
		return api.AssistantUpdatesVerbose
	}
	mode := strings.TrimSpace(a.cfg().Output.AssistantUpdates)
	if mode != "" {
		return api.AssistantUpdateModeFromContext(api.WithAssistantUpdateMode(context.Background(), mode))
	}
	if a.PlanModeEnabled {
		return api.AssistantUpdatesVerbose
	}
	return api.AssistantUpdatesPhase
}

func (a *Agent) shouldStreamAssistantText() bool {
	return a.assistantUpdateMode() == api.AssistantUpdatesVerbose
}

func (a *Agent) shouldEmitAssistantPhaseUpdate() bool {
	return a.assistantUpdateMode() == api.AssistantUpdatesPhase
}

func (a *Agent) printFinalAssistantResponse(response string) {
	if a == nil || a.shouldStreamAssistantText() {
		return
	}
	display := strings.TrimSpace(response)
	if display == "" {
		return
	}
	_, _ = fmt.Fprintf(a.output(), "\n💬 %s\n", display)
}

func (a *Agent) maybePrintAssistantPhaseUpdate(response string, toolCalls []*tools.ToolCall) {
	if a == nil || !a.shouldEmitAssistantPhaseUpdate() {
		return
	}

	explanation, _ := extractExplanationAndTool(response)
	summary := firstAssistantSummaryLine(explanation)
	if summary == "" {
		summary = summarizeToolCallsForPhase(toolCalls)
	}
	if summary == "" {
		return
	}
	_, _ = fmt.Fprintf(a.output(), "\n💬 %s\n", summary)
}

func firstAssistantSummaryLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = strings.TrimLeft(line, "-*0123456789. ")
		line = strings.TrimSpace(line)
		runes := []rune(line)
		if len(runes) > 120 {
			return string(runes[:120]) + "..."
		}
		return line
	}
	return ""
}

func summarizeToolCallsForPhase(toolCalls []*tools.ToolCall) string {
	if len(toolCalls) == 0 {
		return ""
	}

	counts := make(map[string]int, len(toolCalls))
	order := make([]string, 0, len(toolCalls))
	for _, tc := range toolCalls {
		name := tc.Tool
		if _, ok := counts[name]; !ok {
			order = append(order, name)
		}
		counts[name]++
	}

	parts := make([]string, 0, len(order))
	for _, name := range order {
		if counts[name] == 1 {
			parts = append(parts, name)
			continue
		}
		parts = append(parts, fmt.Sprintf("%s x%d", name, counts[name]))
	}
	return "Phase: " + strings.Join(parts, ", ")
}

// normalModeRetryInstruction は Normal Mode の retry プロンプトを段階的に返す。
func normalModeRetryInstruction(attempt int) string {
	const constraint = `
Reuse information already obtained from the failed command/output.
Do not re-run broad investigation unless the failure output is insufficient.`

	switch {
	case attempt <= 1:
		return `Do not patch blindly.
First, identify the concrete root cause in 1-2 sentences using the existing failure output.
Point to the exact file/function/command causing the failure.
Then make the smallest plausible fix and verify it immediately.

Do not write a new test yet unless the root cause is still unclear or verification is otherwise unreliable.` + constraint
	case attempt == 2:
		return `The previous fix did not work.

Do not repeat the same fix pattern.
Briefly explain why the previous attempt failed.
If the root cause is still uncertain, create the smallest possible reproduction or targeted test via bash.
If the root cause is already clear, skip test creation and apply the next evidence-based fix directly.

Then verify again.` + constraint
	default:
		return `Multiple retries have failed.

Your current approach is not working.
Explain which assumption was wrong.
Change strategy fundamentally and avoid repeating the same edit pattern.
Choose the smallest different approach that can validate the new hypothesis quickly.` + constraint
	}
}
