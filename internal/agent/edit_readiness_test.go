package agent

import (
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/taskstate"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

type editReadinessProbeTool struct {
	name              string
	store             *taskstate.Store
	observationsAtRun int
	result            string
	err               error
}

func (t *editReadinessProbeTool) Name() string { return t.name }

func (t *editReadinessProbeTool) Description() string { return "edit readiness probe tool" }

func (t *editReadinessProbeTool) Parameters() map[string]interface{} {
	return map[string]interface{}{}
}

func (t *editReadinessProbeTool) Run(_ tools.ExecutionContext, _ map[string]string) (string, *tools.FileChange, error) {
	t.observationsAtRun = len(t.store.EditReadinessObservations())
	return t.result, nil, t.err
}

func TestExtractEditReadinessTargets_PathTools(t *testing.T) {
	for _, toolName := range []string{"str_replace", "write_file", "delete_file"} {
		t.Run(toolName, func(t *testing.T) {
			extraction := extractEditReadinessTargets(&tools.ToolCall{
				ID:   "call_path",
				Tool: toolName,
				Args: map[string]string{"path": "src/main.go"},
			})
			if extraction.unknown {
				t.Fatal("unknown = true, want false")
			}
			assertEditReadinessTargetPaths(t, extraction.targets, []string{"src/main.go"})
			if extraction.targets[0].ToolName != toolName || extraction.targets[0].ToolCallID != "call_path" {
				t.Fatalf("target metadata = %#v", extraction.targets[0])
			}
		})
	}
}

func TestExtractEditReadinessTargets_ApplyPatch(t *testing.T) {
	tests := []struct {
		name  string
		patch string
		want  []string
	}{
		{
			name: "add",
			patch: strings.Join([]string{
				"*** Begin Patch",
				"*** Add File: added.go",
				"+package added",
				"*** End Patch",
			}, "\n"),
			want: []string{"added.go"},
		},
		{
			name: "update",
			patch: strings.Join([]string{
				"*** Begin Patch",
				"*** Update File: target.go",
				"@@",
				"-old",
				"+new",
				"*** End Patch",
			}, "\n"),
			want: []string{"target.go"},
		},
		{
			name: "delete",
			patch: strings.Join([]string{
				"*** Begin Patch",
				"*** Delete File: old.go",
				"*** End Patch",
			}, "\n"),
			want: []string{"old.go"},
		},
		{
			name: "move",
			patch: strings.Join([]string{
				"*** Begin Patch",
				"*** Update File: old.go",
				"*** Move to: new.go",
				"@@",
				"-old",
				"+new",
				"*** End Patch",
			}, "\n"),
			want: []string{"old.go", "new.go"},
		},
		{
			name: "multi-file-dedupe",
			patch: strings.Join([]string{
				"*** Begin Patch",
				"*** Update File: a.go",
				"@@",
				"-old",
				"+new",
				"*** Update File: b.go",
				"*** Move to: a.go",
				"@@",
				"-old",
				"+new",
				"*** Delete File: c.go",
				"*** End Patch",
			}, "\n"),
			want: []string{"a.go", "b.go", "c.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extraction := extractEditReadinessTargets(&tools.ToolCall{
				ID:   "call_patch",
				Tool: "apply_patch",
				Args: map[string]string{"patch": tt.patch},
			})
			if extraction.unknown {
				t.Fatal("unknown = true, want false")
			}
			assertEditReadinessTargetPaths(t, extraction.targets, tt.want)
			for _, target := range extraction.targets {
				if target.ToolName != "apply_patch" || target.ToolCallID != "call_patch" {
					t.Fatalf("target metadata = %#v", target)
				}
			}
		})
	}
}

func TestExtractEditReadinessTargets_UnknownApplyPatch(t *testing.T) {
	extraction := extractEditReadinessTargets(&tools.ToolCall{
		ID:   "call_patch",
		Tool: "apply_patch",
		Args: map[string]string{"patch": "*** Begin Patch\n*** Bad Marker\n*** End Patch"},
	})
	if !extraction.unknown {
		t.Fatalf("unknown = false, want true: %#v", extraction)
	}
	if len(extraction.targets) != 0 {
		t.Fatalf("targets = %#v, want empty", extraction.targets)
	}
}

func TestObserveEditReadinessBeforeTool_DoesNotMutateConversation(t *testing.T) {
	store := taskstate.NewStoreWithRoot(t.TempDir())
	agent := newEditReadinessProbeAgent(t, store, &editReadinessProbeTool{
		name:   "write_file",
		store:  store,
		result: "ok",
	})
	tc := &tools.ToolCall{
		ID:   "call_write",
		Tool: "write_file",
		Args: map[string]string{"path": "src/main.go"},
	}

	agent.observeEditReadinessBeforeTool(context.Background(), tc)
	assertEditReadinessConversationUnchanged(t, agent)

	assertSingleEditReadinessPathNotInLedgerObservation(t, store, "src/main.go")
}

func TestToolExecution_RecordsEditReadinessBeforeToolWithoutChangingResult(t *testing.T) {
	tests := []struct {
		name    string
		execute func(*Agent, *tools.ToolCall) tools.ExecutionResult
	}{
		{
			name: "spinner",
			execute: func(agent *Agent, tc *tools.ToolCall) tools.ExecutionResult {
				return agent.executeToolWithSpinnerResult(context.Background(), tc)
			},
		},
		{
			name: "quiet",
			execute: func(agent *Agent, tc *tools.ToolCall) tools.ExecutionResult {
				return agent.executeQuietToolResult(context.Background(), tc, strings.NewReader(""), io.Discard, io.Discard, false)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := taskstate.NewStoreWithRoot(t.TempDir())
			probe := &editReadinessProbeTool{
				name:   "write_file",
				store:  store,
				result: "probe ok",
			}
			agent := newEditReadinessProbeAgent(t, store, probe)

			execResult := tt.execute(agent, &tools.ToolCall{
				ID:      "call_write",
				Tool:    "write_file",
				RawArgs: map[string]any{"path": "src/main.go"},
				Args:    map[string]string{"path": "src/main.go"},
			})

			if execResult.Result != "probe ok" || execResult.Error {
				t.Fatalf("exec result = %#v, want unchanged success", execResult)
			}
			if probe.observationsAtRun != 1 {
				t.Fatalf("observations at Run = %d, want 1", probe.observationsAtRun)
			}
			assertEditReadinessConversationUnchanged(t, agent)
			assertSingleEditReadinessPathNotInLedgerObservation(t, store, "src/main.go")
		})
	}
}

func TestExecuteToolWithSpinnerResult_EditReadinessDoesNotChangeFailureResult(t *testing.T) {
	store := taskstate.NewStoreWithRoot(t.TempDir())
	probe := &editReadinessProbeTool{
		name:   "delete_file",
		store:  store,
		result: "will be replaced by registry error",
		err:    errors.New("delete failed"),
	}
	agent := newEditReadinessProbeAgent(t, store, probe)

	execResult := agent.executeToolWithSpinnerResult(context.Background(), &tools.ToolCall{
		ID:      "call_delete",
		Tool:    "delete_file",
		RawArgs: map[string]any{"path": "src/main.go"},
		Args:    map[string]string{"path": "src/main.go"},
	})

	if execResult.Result != "Error: delete failed" || !execResult.Error {
		t.Fatalf("exec result = %#v, want unchanged registry failure", execResult)
	}
	if probe.observationsAtRun != 1 {
		t.Fatalf("observations at Run = %d, want 1", probe.observationsAtRun)
	}
	assertEditReadinessConversationUnchanged(t, agent)
	assertSingleEditReadinessPathNotInLedgerObservation(t, store, "src/main.go")
}

func TestExecuteToolWithSpinnerResult_MalformedApplyPatchStillUsesExistingErrorPath(t *testing.T) {
	t.Chdir(t.TempDir())
	store := taskstate.NewStoreWithRoot(t.TempDir())
	agent := newEditReadinessProbeAgent(t, store, nil)

	execResult := agent.executeToolWithSpinnerResult(context.Background(), &tools.ToolCall{
		ID:      "call_patch",
		Tool:    "apply_patch",
		RawArgs: map[string]any{"patch": "*** Begin Patch\n*** Bad Marker\n*** End Patch"},
		Args:    map[string]string{"patch": "*** Begin Patch\n*** Bad Marker\n*** End Patch"},
	})

	if !execResult.Error || !strings.HasPrefix(strings.TrimSpace(execResult.Result), "Error:") {
		t.Fatalf("exec result = %#v, want existing apply_patch error", execResult)
	}
	assertEditReadinessConversationUnchanged(t, agent)
	observations := store.EditReadinessObservations()
	if len(observations) != 1 || observations[0].Status != taskstate.EditReadinessStatusUnknown {
		t.Fatalf("observations = %#v, want one unknown observation", observations)
	}
}

func TestGeminiApplyPatchRepair_RecordsOriginalAndRepairedReadiness(t *testing.T) {
	tests := []struct {
		name    string
		execute func(*Agent) tools.ExecutionResult
	}{
		{
			name: "spinner",
			execute: func(agent *Agent) tools.ExecutionResult {
				result, change := agent.executeToolWithSpinner(context.Background(), applyPatchCall(invalidAddFilePatch))
				return tools.ExecutionResult{
					Result: result,
					Change: change,
					Error:  tools.IsErrorResult(result),
				}
			},
		},
		{
			name: "quiet",
			execute: func(agent *Agent) tools.ExecutionResult {
				return agent.executeQuietToolResult(context.Background(), applyPatchCall(invalidAddFilePatch), strings.NewReader(""), io.Discard, io.Discard, true)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspace := t.TempDir()
			t.Chdir(workspace)

			provider := &sequenceMockProvider{name: "gemini", responses: []string{validAddFilePatch}}
			agent := newGeminiRepairTestAgent(t, provider)
			store := taskstate.NewStoreWithRoot(workspace)
			agent.Runtime.TaskLedger = store

			execResult := tt.execute(agent)
			if execResult.Error || strings.HasPrefix(strings.TrimSpace(execResult.Result), "Error:") {
				t.Fatalf("execution result = %#v, want repaired success", execResult)
			}

			assertOriginalAndRepairedEditReadinessObservations(t, store, "hello.txt")
		})
	}
}

func newEditReadinessProbeAgent(t *testing.T, store *taskstate.Store, tool tools.Tool) *Agent {
	t.Helper()

	runtime := NewAgentRuntimeWithConfig(newProjectMapDisabledConfig())
	runtime.AutoApprove = true
	runtime.ToolCache = nil
	runtime.TaskLedger = store
	runtime.UI = uiruntime.NewRuntime(strings.NewReader(""), io.Discard, io.Discard)
	if tool != nil {
		runtime.Registry.Register(tool)
	}
	agent := NewAgentWithRuntime("test-model", &mockProvider{name: "test"}, false, runtime)
	agent.registry().ClearExcludedTools()
	t.Cleanup(agent.Cleanup)
	return agent
}

func assertEditReadinessTargetPaths(t *testing.T, targets []taskstate.EditReadinessTarget, want []string) {
	t.Helper()
	paths := make([]string, 0, len(targets))
	for _, target := range targets {
		paths = append(paths, target.Path)
	}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("target paths = %v, want %v", paths, want)
	}
}

func assertEditReadinessConversationUnchanged(t *testing.T, agent *Agent) {
	t.Helper()
	if len(agent.History) != 0 {
		t.Fatalf("History len = %d, want 0", len(agent.History))
	}
	if agent.session != nil && len(agent.session.Messages) != 0 {
		t.Fatalf("session messages len = %d, want 0", len(agent.session.Messages))
	}
}

func assertSingleEditReadinessPathNotInLedgerObservation(t *testing.T, store *taskstate.Store, path string) {
	t.Helper()

	observations := store.EditReadinessObservations()
	if len(observations) != 1 {
		t.Fatalf("observations = %#v, want one path_not_in_ledger warning", observations)
	}
	assertEditReadinessPathNotInLedgerObservation(t, observations[0], path)
}

func assertOriginalAndRepairedEditReadinessObservations(t *testing.T, store *taskstate.Store, repairedPath string) {
	t.Helper()

	observations := store.EditReadinessObservations()
	if len(observations) != 2 {
		t.Fatalf("observations = %#v, want original unknown and repaired target warning", observations)
	}
	if observations[0].Status != taskstate.EditReadinessStatusUnknown {
		t.Fatalf("original observation = %#v, want unknown", observations[0])
	}
	assertEditReadinessPathNotInLedgerObservation(t, observations[1], repairedPath)
}

func assertEditReadinessPathNotInLedgerObservation(t *testing.T, observation taskstate.EditReadinessObservation, path string) {
	t.Helper()

	if observation.Path != path ||
		observation.Status != taskstate.EditReadinessStatusWarning ||
		!reflect.DeepEqual(observation.Reasons, []taskstate.EditReadinessReason{taskstate.EditReadinessReasonPathNotInLedger}) {
		t.Fatalf("observation = %#v, want %s path_not_in_ledger warning", observation, path)
	}
}
