package agent

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

type runtimeProbeTool struct{}

func (t *runtimeProbeTool) Name() string { return "runtime_probe_test" }

func (t *runtimeProbeTool) Description() string { return "probe runtime state for tests" }

func (t *runtimeProbeTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{"type": "string"},
		},
	}
}

func (t *runtimeProbeTool) Run(execCtx tools.ExecutionContext, args map[string]string) (string, *tools.FileChange, error) {
	cacheValue := ""
	if cache := execCtx.EffectiveToolCache(); cache != nil {
		if cached, ok := cache.GetDir(args["path"]); ok {
			cacheValue = cached
		}
	}

	decision := common.ConfirmWithAutoApproveDecisionAndOptions(
		execCtx.PromptIO(),
		execCtx.ConfirmOptions(),
		"write_file",
		"runtime probe confirmation",
	)

	return fmt.Sprintf(
		"ignore=%s cache=%s confirm=%s",
		strings.Join(execCtx.EffectiveConfig().ListDir.AdditionalIgnoreDirs, ","),
		cacheValue,
		decision.Action,
	), nil, nil
}

type runtimeAOnlyTool struct{}

func (t *runtimeAOnlyTool) Name() string { return "runtime_a_only_test" }

func (t *runtimeAOnlyTool) Description() string { return "registry isolation probe" }

func (t *runtimeAOnlyTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

func (t *runtimeAOnlyTool) Run(_ tools.ExecutionContext, _ map[string]string) (string, *tools.FileChange, error) {
	return "runtime-a-only", nil, nil
}

func newIsolatedRuntime() *AgentRuntime {
	runtime := NewAgentRuntime()
	runtime.Registry = tools.DefaultRegistry.Clone()
	runtime.ToolCache = NewToolCache()
	return runtime
}

func newRuntimeTestAgent(t *testing.T, runtime *AgentRuntime) *Agent {
	t.Helper()

	agent := NewAgentWithRuntime("test-model", &mockProvider{name: "test"}, false, runtime)
	t.Cleanup(agent.Cleanup)
	return agent
}

func executeRuntimeTool(agent *Agent, stdin io.Reader, tc *tools.ToolCall) string {
	result, _ := tools.ExecuteQuietWithContext(agent.toolExecutionContext(stdin, io.Discard, io.Discard), tc)
	return result
}

func TestAgentRuntime_SeparatesBuiltinListDirConfig(t *testing.T) {
	root, err := os.MkdirTemp(".", "runtime-listdir-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	if err := os.Mkdir(filepath.Join(root, "skipme"), 0o755); err != nil {
		t.Fatalf("mkdir skipme: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "visible.txt"), []byte("visible"), 0o644); err != nil {
		t.Fatalf("write visible file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "skipme", "hidden.txt"), []byte("hidden"), 0o644); err != nil {
		t.Fatalf("write hidden file: %v", err)
	}

	runtimeA := newIsolatedRuntime()
	runtimeB := newIsolatedRuntime()
	runtimeA.Config.ListDir.AdditionalIgnoreDirs = []string{"skipme"}
	runtimeB.Config.ListDir.AdditionalIgnoreDirs = nil

	agentA := newRuntimeTestAgent(t, runtimeA)
	agentB := newRuntimeTestAgent(t, runtimeB)

	callA := &tools.ToolCall{
		Tool:    "list_dir",
		Args:    map[string]string{"path": root, "depth": "2"},
		RawArgs: map[string]any{"path": root, "depth": "2"},
	}
	callB := &tools.ToolCall{
		Tool:    "list_dir",
		Args:    map[string]string{"path": root, "depth": "2"},
		RawArgs: map[string]any{"path": root, "depth": "2"},
	}

	resultA := executeRuntimeTool(agentA, strings.NewReader(""), callA)
	resultB := executeRuntimeTool(agentB, strings.NewReader(""), callB)

	if strings.Contains(resultA, "skipme") {
		t.Fatalf("runtime A should hide skipme, got:\n%s", resultA)
	}
	if !strings.Contains(resultB, "skipme") {
		t.Fatalf("runtime B should include skipme, got:\n%s", resultB)
	}
}

func TestAgentRuntime_SeparatesRegistryCacheAndAutoApprove(t *testing.T) {
	root, err := os.MkdirTemp(".", "runtime-cache-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })

	runtimeA := newIsolatedRuntime()
	runtimeB := newIsolatedRuntime()
	runtimeA.AutoApprove = true
	runtimeB.AutoApprove = false
	runtimeA.Config.ListDir.AdditionalIgnoreDirs = []string{"skipme"}
	runtimeB.Config.ListDir.AdditionalIgnoreDirs = []string{"other"}

	runtimeA.Registry.Register(&runtimeProbeTool{})
	runtimeB.Registry.Register(&runtimeProbeTool{})
	runtimeA.Registry.Register(&runtimeAOnlyTool{})

	runtimeA.ToolCache.SetDir(root, "cache-a")
	runtimeB.ToolCache.SetDir(root, "cache-b")

	agentA := newRuntimeTestAgent(t, runtimeA)
	agentB := newRuntimeTestAgent(t, runtimeB)

	probeA := &tools.ToolCall{
		Tool:    "runtime_probe_test",
		Args:    map[string]string{"path": root},
		RawArgs: map[string]any{"path": root},
	}
	probeB := &tools.ToolCall{
		Tool:    "runtime_probe_test",
		Args:    map[string]string{"path": root},
		RawArgs: map[string]any{"path": root},
	}

	resultA := executeRuntimeTool(agentA, strings.NewReader(""), probeA)
	resultB := executeRuntimeTool(agentB, strings.NewReader("n\n"), probeB)

	if !strings.Contains(resultA, "ignore=skipme") || !strings.Contains(resultA, "cache=cache-a") || !strings.Contains(resultA, "confirm=yes") {
		t.Fatalf("runtime A state not injected correctly: %s", resultA)
	}
	if !strings.Contains(resultB, "ignore=other") || !strings.Contains(resultB, "cache=cache-b") || !strings.Contains(resultB, "confirm=no") {
		t.Fatalf("runtime B state not injected correctly: %s", resultB)
	}

	onlyA := &tools.ToolCall{
		Tool:    "runtime_a_only_test",
		Args:    map[string]string{},
		RawArgs: map[string]any{},
	}

	if got := executeRuntimeTool(agentA, strings.NewReader(""), onlyA); got != "runtime-a-only" {
		t.Fatalf("runtime A registry result = %q, want %q", got, "runtime-a-only")
	}
	if got := executeRuntimeTool(agentB, strings.NewReader(""), &tools.ToolCall{
		Tool:    "runtime_a_only_test",
		Args:    map[string]string{},
		RawArgs: map[string]any{},
	}); got != "Unknown tool: runtime_a_only_test" {
		t.Fatalf("runtime B registry should not know runtime_a_only_test, got %q", got)
	}
}
