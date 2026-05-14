package agent

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/toolruntime"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

const (
	applyPatchBeginMarker = "*** Begin Patch"
	applyPatchEndMarker   = "*** End Patch"
)

const geminiApplyPatchRepairSystemPrompt = `You repair failed apply_patch function tool arguments.
Return only the corrected patch text.
Do not include markdown fences or commentary.
The patch must start with *** Begin Patch and end with *** End Patch.
The patch may modify multiple files.`

const geminiApplyPatchRepairCacheNamespace = "gemini_apply_patch_repair"

func (a *Agent) maybeRepairGeminiApplyPatchExecution(ctx context.Context, tc *tools.ToolCall, execResult tools.ExecutionResult, execCtx tools.ExecutionContext, quiet bool) tools.ExecutionResult {
	if !a.shouldRepairGeminiApplyPatch(tc) {
		return execResult
	}

	a.recordGeminiApplyPatchAttempt()
	if !execResult.Error {
		a.recordGeminiApplyPatchSuccess()
		return execResult
	}

	originalPatch := strings.TrimSpace(tc.Args["patch"])
	if originalPatch == "" {
		return execResult
	}
	originalArgsJSON := toolruntime.ArgsToJSON(tc.RawArgs)
	if originalArgsJSON == "{}" {
		originalArgsJSON = toolruntime.ArgsToJSON(map[string]any{"patch": tc.Args["patch"]})
	}

	a.recordGeminiApplyPatchRepairAttempt()
	repairedPatch, err := a.requestGeminiApplyPatchRepair(ctx, originalPatch, execResult.Result)
	if err != nil || strings.TrimSpace(repairedPatch) == "" {
		return execResult
	}

	repairedTC := &tools.ToolCall{
		ID:      tc.ID,
		Tool:    tc.Tool,
		RawArgs: map[string]any{"patch": repairedPatch},
		Args:    map[string]string{"patch": repairedPatch},
	}

	repairedExecResult := executeRepairedApplyPatch(execCtx, repairedTC, quiet)
	if repairedExecResult.Error {
		return execResult
	}

	a.syncRepairedApplyPatchToolCall(tc, originalArgsJSON, repairedPatch)
	a.recordGeminiApplyPatchSuccess()
	a.recordGeminiApplyPatchRepairSuccess()
	return repairedExecResult
}

func executeRepairedApplyPatch(execCtx tools.ExecutionContext, tc *tools.ToolCall, quiet bool) tools.ExecutionResult {
	if quiet {
		return tools.ExecuteQuietUnpublishedWithContext(execCtx, tc)
	}
	return tools.ExecuteUnpublishedWithContext(execCtx, tc)
}

func (a *Agent) shouldRepairGeminiApplyPatch(tc *tools.ToolCall) bool {
	return a != nil &&
		tc != nil &&
		tc.Tool == "apply_patch" &&
		config.CanonicalProviderName(a.ProviderName) == "gemini" &&
		a.CurrentProvider != nil
}

func (a *Agent) requestGeminiApplyPatchRepair(ctx context.Context, originalPatch, errorResult string) (string, error) {
	ctx = a.requestContextWithoutActiveContext(ctx)
	ctx = ui.WithRuntime(ctx, ui.NewRuntime(strings.NewReader(""), io.Discard, io.Discard))
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)
	ctx = api.WithProviderCacheNamespace(ctx, geminiApplyPatchRepairCacheNamespace)
	response, err := a.CurrentProvider.ChatWithTools(
		ctx,
		geminiApplyPatchRepairSystemPrompt,
		[]api.Message{{
			Role:    "user",
			Content: buildGeminiApplyPatchRepairPrompt(originalPatch, errorResult),
		}},
		a.CurrentModel,
	)
	if err != nil {
		return "", err
	}
	if patch, ok := extractRepairedApplyPatch(response); ok {
		return patch, nil
	}
	return "", fmt.Errorf("repair response did not contain an apply_patch patch")
}

func buildGeminiApplyPatchRepairPrompt(originalPatch, errorResult string) string {
	return fmt.Sprintf(`The previous apply_patch function tool call failed.

Error:
%s

Original patch:
%s

Return only a corrected apply_patch patch.`, strings.TrimSpace(errorResult), strings.TrimSpace(originalPatch))
}

func (a *Agent) syncRepairedApplyPatchToolCall(tc *tools.ToolCall, originalArgsJSON, repairedPatch string) {
	if tc == nil {
		return
	}
	if tc.Args == nil {
		tc.Args = make(map[string]string)
	}
	tc.Args["patch"] = repairedPatch
	if tc.RawArgs == nil {
		tc.RawArgs = make(map[string]any)
	}
	tc.RawArgs["patch"] = repairedPatch

	if a == nil {
		return
	}

	argsJSON := toolruntime.ArgsToJSON(tc.RawArgs)
	a.syncRepairedApplyPatchHistoryToolCall(tc, originalArgsJSON, argsJSON)
	if a.syncRepairedApplyPatchSessionToolCall(tc, originalArgsJSON, argsJSON) {
		a.rewriteSessionWithWarning("⚠️  Warning: Failed to rewrite session after apply_patch repair: %v\n")
	}
}

func (a *Agent) syncRepairedApplyPatchHistoryToolCall(tc *tools.ToolCall, originalArgsJSON, repairedArgsJSON string) {
	if a == nil || tc == nil {
		return
	}
	for i := len(a.History) - 1; i >= 0; i-- {
		for j := range a.History[i].ToolCalls {
			historyTC := &a.History[i].ToolCalls[j]
			if matchesRepairedApplyPatchToolCall(historyTC.ID, historyTC.Function.Name, historyTC.Function.Arguments, tc, originalArgsJSON) {
				historyTC.Function.Arguments = repairedArgsJSON
				return
			}
		}
	}
}

func (a *Agent) syncRepairedApplyPatchSessionToolCall(tc *tools.ToolCall, originalArgsJSON, repairedArgsJSON string) bool {
	if a == nil || a.session == nil || tc == nil {
		return false
	}
	for i := len(a.session.Messages) - 1; i >= 0; i-- {
		for j := range a.session.Messages[i].ToolCalls {
			sessionTC := &a.session.Messages[i].ToolCalls[j]
			if matchesRepairedApplyPatchToolCall(sessionTC.ID, sessionTC.Function.Name, sessionTC.Function.Arguments, tc, originalArgsJSON) {
				sessionTC.Function.Arguments = repairedArgsJSON
				a.session.LastModified = time.Now()
				return true
			}
		}
	}
	return false
}

func matchesRepairedApplyPatchToolCall(id, name, args string, tc *tools.ToolCall, originalArgsJSON string) bool {
	if tc == nil || name != tc.Tool {
		return false
	}
	if tc.ID != "" {
		return id == tc.ID
	}
	return id == "" && args == originalArgsJSON
}

func extractRepairedApplyPatch(response string) (string, bool) {
	for _, tc := range tools.ParseToolCalls(response) {
		if tc.Tool == "apply_patch" && strings.TrimSpace(tc.Args["patch"]) != "" {
			return strings.TrimSpace(tc.Args["patch"]), true
		}
	}

	return extractRepairedApplyPatchMarkerLines(response)
}

func extractRepairedApplyPatchMarkerLines(response string) (string, bool) {
	normalized := strings.ReplaceAll(response, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	lines := strings.Split(normalized, "\n")
	start := -1
	for i, line := range lines {
		if line == applyPatchBeginMarker {
			start = i
			break
		}
	}
	if start < 0 {
		return "", false
	}
	for i := start + 1; i < len(lines); i++ {
		if lines[i] == applyPatchEndMarker {
			return strings.TrimSpace(strings.Join(lines[start:i+1], "\n")), true
		}
	}
	return "", false
}

func (a *Agent) recordGeminiApplyPatchAttempt() {
	a.statsMu.Lock()
	defer a.statsMu.Unlock()
	if a.Stats != nil {
		a.Stats.ToolObs.ApplyPatchAttempts++
	}
}

func (a *Agent) recordGeminiApplyPatchSuccess() {
	a.statsMu.Lock()
	defer a.statsMu.Unlock()
	if a.Stats != nil {
		a.Stats.ToolObs.ApplyPatchSuccesses++
	}
}

func (a *Agent) recordGeminiApplyPatchRepairAttempt() {
	a.statsMu.Lock()
	defer a.statsMu.Unlock()
	if a.Stats != nil {
		a.Stats.ToolObs.ApplyPatchRepairAttempts++
	}
}

func (a *Agent) recordGeminiApplyPatchRepairSuccess() {
	a.statsMu.Lock()
	defer a.statsMu.Unlock()
	if a.Stats != nil {
		a.Stats.ToolObs.ApplyPatchRepairSuccesses++
	}
}
