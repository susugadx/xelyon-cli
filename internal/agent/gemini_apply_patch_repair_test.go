package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/toolruntime"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

const (
	invalidAddFilePatch = "*** Begin Patch\n*** Add File: hello.txt\nhello\n*** End Patch"
	validAddFilePatch   = "*** Begin Patch\n*** Add File: hello.txt\n+hello\n*** End Patch"
)

func TestGeminiApplyPatchRepairSuccess(t *testing.T) {
	t.Chdir(t.TempDir())

	provider := &sequenceMockProvider{name: "gemini", responses: []string{validAddFilePatch}}
	agent := newGeminiRepairTestAgent(t, provider)
	agent.Runtime.Options.EnableCurrentTaskStateContext = true
	agent.Runtime.TaskLedger = newTaskLedgerWithPassedTest(t)

	tc := applyPatchCall(invalidAddFilePatch)
	result, change := agent.executeToolWithSpinner(context.Background(), tc)
	if strings.HasPrefix(strings.TrimSpace(result), "Error:") {
		t.Fatalf("executeToolWithSpinner() result = %q, want repaired success", result)
	}
	if change == nil {
		t.Fatal("repair success should return file change")
	}
	if got, err := os.ReadFile("hello.txt"); err != nil || string(got) != "hello\n" {
		t.Fatalf("hello.txt = %q, %v; want hello with trailing newline", string(got), err)
	}
	if provider.callCount != 1 {
		t.Fatalf("repair provider calls = %d, want 1", provider.callCount)
	}
	if got := tc.Args["patch"]; got != validAddFilePatch {
		t.Fatalf("recorded Args patch = %q, want repaired patch", got)
	}
	if got, _ := tc.RawArgs["patch"].(string); got != validAddFilePatch {
		t.Fatalf("recorded RawArgs patch = %q, want repaired patch", got)
	}
	if len(provider.contexts) != 1 {
		t.Fatalf("repair provider contexts = %d, want 1", len(provider.contexts))
	}
	if got := api.ProviderCacheNamespaceFromContext(provider.contexts[0]); got != geminiApplyPatchRepairCacheNamespace {
		t.Fatalf("repair cache namespace = %q, want %q", got, geminiApplyPatchRepairCacheNamespace)
	}
	if got := api.AssistantUpdateModeFromContext(provider.contexts[0]); got != api.AssistantUpdatesOff {
		t.Fatalf("repair assistant update mode = %q, want off", got)
	}
	if got := api.RuntimeFromContext(provider.contexts[0]); got == agent.ui() {
		t.Fatal("repair context should use a silent UI runtime instead of the agent UI runtime")
	}
	if got := tools.RegistryFromContext(provider.contexts[0]); got != agent.registry() {
		t.Fatal("repair context should carry agent tool registry")
	}
	if got := api.ActiveContextBlocksFromContext(provider.contexts[0]); got != nil {
		t.Fatalf("repair active context = %#v, want nil for isolated repair model call", got)
	}

	obs := agent.Stats.ToolObs
	if obs.ApplyPatchAttempts != 1 || obs.ApplyPatchSuccesses != 1 || obs.ApplyPatchRepairAttempts != 1 || obs.ApplyPatchRepairSuccesses != 1 {
		t.Fatalf("unexpected patch metrics: %+v", obs)
	}
}

func TestGeminiApplyPatchRepairInteractivePublishesOnlyFinalResult(t *testing.T) {
	t.Chdir(t.TempDir())

	provider := &sequenceMockProvider{name: "gemini", responses: []string{validAddFilePatch}}
	agent := newGeminiRepairTestAgent(t, provider)
	toolResultCh := make(chan tools.ToolResultInfo, 4)
	agent.tuiToolResultCh = toolResultCh

	tc := applyPatchCall(invalidAddFilePatch)
	result, _ := agent.executeToolWithSpinner(context.Background(), tc)
	if strings.HasPrefix(strings.TrimSpace(result), "Error:") {
		t.Fatalf("executeToolWithSpinner() result = %q, want repaired success", result)
	}

	if got := len(toolResultCh); got != 2 {
		t.Fatalf("published tool results = %d, want running + final", got)
	}
	running := <-toolResultCh
	if running.Status != tools.ToolStatusRunning || running.ToolName != "apply_patch" {
		t.Fatalf("first published result = %+v, want running apply_patch", running)
	}
	info := <-toolResultCh
	if info.ToolName != "apply_patch" {
		t.Fatalf("published tool = %q, want apply_patch", info.ToolName)
	}
	if info.Status != tools.ToolStatusOK {
		t.Fatalf("published status = %q, want ok", info.Status)
	}
	if info.Error || strings.HasPrefix(strings.TrimSpace(info.Result), "Error:") {
		t.Fatalf("published result should be repaired success: %+v", info)
	}
	if info.Args["patch"] != validAddFilePatch {
		t.Fatalf("published args patch = %q, want repaired patch", info.Args["patch"])
	}
}

func TestGeminiApplyPatchRepairFailureReturnsOriginalFailure(t *testing.T) {
	t.Chdir(t.TempDir())

	provider := &sequenceMockProvider{name: "gemini", responses: []string{invalidAddFilePatch}}
	agent := newGeminiRepairTestAgent(t, provider)

	tc := applyPatchCall(invalidAddFilePatch)
	result, change := agent.executeToolWithSpinner(context.Background(), tc)
	if !strings.HasPrefix(strings.TrimSpace(result), "Error:") {
		t.Fatalf("executeToolWithSpinner() result = %q, want original failure", result)
	}
	if change != nil {
		t.Fatalf("change = %+v, want nil", change)
	}
	if provider.callCount != 1 {
		t.Fatalf("repair provider calls = %d, want 1", provider.callCount)
	}
	if _, err := os.Stat("hello.txt"); !os.IsNotExist(err) {
		t.Fatalf("hello.txt should not be created, stat err = %v", err)
	}
	if got := tc.Args["patch"]; got != invalidAddFilePatch {
		t.Fatalf("failed repair should keep original Args patch = %q", got)
	}
	if got, _ := tc.RawArgs["patch"].(string); got != invalidAddFilePatch {
		t.Fatalf("failed repair should keep original RawArgs patch = %q", got)
	}

	obs := agent.Stats.ToolObs
	if obs.ApplyPatchAttempts != 1 || obs.ApplyPatchSuccesses != 0 || obs.ApplyPatchRepairAttempts != 1 || obs.ApplyPatchRepairSuccesses != 0 {
		t.Fatalf("unexpected patch metrics: %+v", obs)
	}
}

func TestApplyPatchRepairSkippedForNonGeminiProvider(t *testing.T) {
	t.Chdir(t.TempDir())

	provider := &sequenceMockProvider{name: "openai", responses: []string{validAddFilePatch}}
	agent := newGeminiRepairTestAgent(t, provider)

	result, change := agent.executeToolWithSpinner(context.Background(), applyPatchCall(invalidAddFilePatch))
	if !strings.HasPrefix(strings.TrimSpace(result), "Error:") {
		t.Fatalf("executeToolWithSpinner() result = %q, want failure without repair", result)
	}
	if change != nil {
		t.Fatalf("change = %+v, want nil", change)
	}
	if provider.callCount != 0 {
		t.Fatalf("non-Gemini provider calls = %d, want 0", provider.callCount)
	}
	if agent.Stats.ToolObs.ApplyPatchAttempts != 0 {
		t.Fatalf("non-Gemini patch metrics should not be recorded: %+v", agent.Stats.ToolObs)
	}
}

func TestGeminiApplyPatchRepairSuccessUpdatesAssistantToolCallHistory(t *testing.T) {
	t.Chdir(t.TempDir())

	provider := &sequenceMockProvider{name: "gemini", responses: []string{validAddFilePatch}}
	agent := newGeminiRepairTestAgent(t, provider)

	tc := applyPatchCall(invalidAddFilePatch)
	tc.ID = "call_patch"
	agent.History = append(agent.History, api.Message{
		Role: "assistant",
		ToolCalls: []api.OpenAIToolCall{{
			ID:   tc.ID,
			Type: "function",
			Function: api.OpenAIToolCallFunction{
				Name:      tc.Tool,
				Arguments: toolruntime.ArgsToJSON(tc.RawArgs),
			},
		}},
	})
	agent.session.AddMessageFromAPI(agent.History[0], agent.CurrentModel)

	result, _ := agent.executeToolWithSpinner(context.Background(), tc)
	if strings.HasPrefix(strings.TrimSpace(result), "Error:") {
		t.Fatalf("executeToolWithSpinner() result = %q, want repaired success", result)
	}

	gotArgs := agent.History[0].ToolCalls[0].Function.Arguments
	var parsed map[string]string
	if err := json.Unmarshal([]byte(gotArgs), &parsed); err != nil {
		t.Fatalf("history tool arguments are not JSON: %v", err)
	}
	if parsed["patch"] != validAddFilePatch {
		t.Fatalf("history patch = %q, want repaired patch", parsed["patch"])
	}

	sessionArgs := agent.session.Messages[0].ToolCalls[0].Function.Arguments
	parsed = nil
	if err := json.Unmarshal([]byte(sessionArgs), &parsed); err != nil {
		t.Fatalf("session tool arguments are not JSON: %v", err)
	}
	if parsed["patch"] != validAddFilePatch {
		t.Fatalf("session patch = %q, want repaired patch", parsed["patch"])
	}
}

func TestGeminiApplyPatchRepairSuccessUpdatesAssistantToolCallHistoryWithoutID(t *testing.T) {
	t.Chdir(t.TempDir())

	provider := &sequenceMockProvider{name: "gemini", responses: []string{validAddFilePatch}}
	agent := newGeminiRepairTestAgent(t, provider)

	tc := applyPatchCall(invalidAddFilePatch)
	agent.History = append(agent.History, api.Message{
		Role: "assistant",
		ToolCalls: []api.OpenAIToolCall{{
			Type: "function",
			Function: api.OpenAIToolCallFunction{
				Name:      tc.Tool,
				Arguments: toolruntime.ArgsToJSON(tc.RawArgs),
			},
		}},
	})
	agent.session.AddMessageFromAPI(agent.History[0], agent.CurrentModel)

	result, _ := agent.executeToolWithSpinner(context.Background(), tc)
	if strings.HasPrefix(strings.TrimSpace(result), "Error:") {
		t.Fatalf("executeToolWithSpinner() result = %q, want repaired success", result)
	}

	gotArgs := agent.History[0].ToolCalls[0].Function.Arguments
	var parsed map[string]string
	if err := json.Unmarshal([]byte(gotArgs), &parsed); err != nil {
		t.Fatalf("history tool arguments are not JSON: %v", err)
	}
	if parsed["patch"] != validAddFilePatch {
		t.Fatalf("history patch = %q, want repaired patch", parsed["patch"])
	}

	sessionArgs := agent.session.Messages[0].ToolCalls[0].Function.Arguments
	parsed = nil
	if err := json.Unmarshal([]byte(sessionArgs), &parsed); err != nil {
		t.Fatalf("session tool arguments are not JSON: %v", err)
	}
	if parsed["patch"] != validAddFilePatch {
		t.Fatalf("session patch = %q, want repaired patch", parsed["patch"])
	}
}

func TestExtractRepairedApplyPatchMatchesMarkerLines(t *testing.T) {
	response := `Some preface
*** Begin Patch
*** Add File: docs.md
+The literal marker can appear in content:
+*** End Patch
+but only a marker line should terminate the patch.
*** End Patch
trailing commentary`

	got, ok := extractRepairedApplyPatch(response)
	if !ok {
		t.Fatal("extractRepairedApplyPatch() should find patch")
	}
	want := `*** Begin Patch
*** Add File: docs.md
+The literal marker can appear in content:
+*** End Patch
+but only a marker line should terminate the patch.
*** End Patch`
	if got != want {
		t.Fatalf("extractRepairedApplyPatch() = %q, want %q", got, want)
	}
}

func TestExtractRepairedApplyPatchDoesNotTreatContextMarkerLineAsTerminator(t *testing.T) {
	response := `*** Begin Patch
*** Update File: docs.md
@@
 *** End Patch
-old line
+new line
*** End Patch`

	got, ok := extractRepairedApplyPatch(response)
	if !ok {
		t.Fatal("extractRepairedApplyPatch() should find patch")
	}
	want := `*** Begin Patch
*** Update File: docs.md
@@
 *** End Patch
-old line
+new line
*** End Patch`
	if got != want {
		t.Fatalf("extractRepairedApplyPatch() = %q, want %q", got, want)
	}
}

func newGeminiRepairTestAgent(t *testing.T, provider *sequenceMockProvider) *Agent {
	t.Helper()

	var out bytes.Buffer
	runtime := NewAgentRuntimeWithConfig(newProjectMapDisabledConfig())
	runtime.AutoApprove = true
	runtime.ToolCache = nil
	runtime.UI = uiruntime.NewRuntime(strings.NewReader(""), &out, &out)

	agent := NewAgentWithRuntime("test-model", provider, false, runtime)
	t.Cleanup(agent.Cleanup)
	return agent
}

func applyPatchCall(patch string) *tools.ToolCall {
	return &tools.ToolCall{
		Tool:    "apply_patch",
		RawArgs: map[string]any{"patch": patch},
		Args:    map[string]string{"patch": patch},
	}
}
