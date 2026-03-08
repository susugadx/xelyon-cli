package agent

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/providers/claude"
	"github.com/susugadx/xelyon-cli/internal/api/providers/openai"
	"github.com/susugadx/xelyon-cli/internal/audit"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/ui"
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

func TestAgentRuntime_RequestContextSeparatesProviderView(t *testing.T) {
	runtimeA := newIsolatedRuntime()
	runtimeB := newIsolatedRuntime()

	runtimeA.Registry.Register(&runtimeAOnlyTool{})
	runtimeA.Config.ProviderModels["openai"] = config.ProviderModelConfig{
		DefaultModel:    "gpt-5.2-codex",
		MaxOutputTokens: 111,
	}
	runtimeB.Config.ProviderModels["openai"] = config.ProviderModelConfig{
		DefaultModel:    "gpt-5.2",
		MaxOutputTokens: 222,
	}
	runtimeA.Config.PromptCache.Enabled = true
	runtimeB.Config.PromptCache.Enabled = false
	runtimeA.Config.Thinking.Enabled = true
	runtimeB.Config.Thinking.Enabled = false

	agentA := newRuntimeTestAgent(t, runtimeA)
	agentB := newRuntimeTestAgent(t, runtimeB)

	ctxA := agentA.requestContext(context.Background())
	ctxB := agentB.requestContext(context.Background())

	defsA := openai.GetCombinedOpenAIToolsWithContext(ctxA, nil)
	defsB := openai.GetCombinedOpenAIToolsWithContext(ctxB, nil)

	hasTool := func(defs []api.OpenAITool, name string) bool {
		for _, def := range defs {
			if def.Function != nil && def.Function.Name == name {
				return true
			}
		}
		return false
	}

	if !hasTool(defsA, "runtime_a_only_test") {
		t.Fatal("runtime A provider view should include runtime_a_only_test")
	}
	if hasTool(defsB, "runtime_a_only_test") {
		t.Fatal("runtime B provider view must not include runtime_a_only_test")
	}

	if got := api.GetDefaultModelWithContext(ctxA, "", "openai", "fallback"); got != "gpt-5.2-codex" {
		t.Fatalf("runtime A default model = %q, want %q", got, "gpt-5.2-codex")
	}
	if got := api.GetDefaultModelWithContext(ctxB, "", "openai", "fallback"); got != "gpt-5.2" {
		t.Fatalf("runtime B default model = %q, want %q", got, "gpt-5.2")
	}
	if got := api.GetMaxOutputTokens(ctxA, "openai", "runtime-a-unknown-model"); got != 111 {
		t.Fatalf("runtime A max tokens = %d, want %d", got, 111)
	}
	if got := api.GetMaxOutputTokens(ctxB, "openai", "runtime-b-unknown-model"); got != 222 {
		t.Fatalf("runtime B max tokens = %d, want %d", got, 222)
	}
	if !api.IsThinkingEnabled(ctxA) {
		t.Fatal("runtime A should enable thinking via request context config")
	}
	if api.IsThinkingEnabled(ctxB) {
		t.Fatal("runtime B should disable thinking via request context config")
	}

	claudeDefsA := claude.GetCombinedClaudeToolsWithContext(ctxA, nil)
	claudeDefsB := claude.GetCombinedClaudeToolsWithContext(ctxB, nil)
	if len(claudeDefsA) == 0 || len(claudeDefsB) == 0 {
		t.Fatal("expected Claude tool definitions to be available")
	}
	if claudeDefsA[len(claudeDefsA)-1].CacheControl == nil {
		t.Fatal("runtime A should enable prompt cache on Claude tool definitions")
	}
	if claudeDefsB[len(claudeDefsB)-1].CacheControl != nil {
		t.Fatal("runtime B should disable prompt cache on Claude tool definitions")
	}
}

func TestAgentRuntime_SeparatesClaudeCompactionCapability(t *testing.T) {
	originalConfig := config.GetGlobalConfig()
	globalCfg := config.DefaultConfig()
	globalCfg.Compression.ClaudeCompaction = false
	config.SetGlobalConfig(globalCfg)
	t.Cleanup(func() {
		config.SetGlobalConfig(originalConfig)
	})

	runtimeA := newIsolatedRuntime()
	runtimeB := newIsolatedRuntime()
	runtimeA.Config.Compression.ClaudeCompaction = true
	runtimeB.Config.Compression.ClaudeCompaction = false
	runtimeA.Config.ProviderModels["claude"] = config.ProviderModelConfig{DefaultModel: "claude-sonnet-4-6"}
	runtimeB.Config.ProviderModels["claude"] = config.ProviderModelConfig{DefaultModel: "claude-sonnet-4-6"}

	agentA := NewAgentWithRuntime("claude-sonnet-4-6", claude.New("test-key"), false, runtimeA)
	agentB := NewAgentWithRuntime("claude-sonnet-4-6", claude.New("test-key"), false, runtimeB)
	t.Cleanup(agentA.Cleanup)
	t.Cleanup(agentB.Cleanup)

	runtimeCapableA, ok := agentA.CurrentProvider.(api.ClaudeCompactionRuntimeCapable)
	if !ok {
		t.Fatal("agent A provider should support runtime Claude compaction checks")
	}
	runtimeCapableB, ok := agentB.CurrentProvider.(api.ClaudeCompactionRuntimeCapable)
	if !ok {
		t.Fatal("agent B provider should support runtime Claude compaction checks")
	}

	if !runtimeCapableA.SupportsClaudeCompactionWithContext(agentA.requestContext(context.Background()), agentA.CurrentModel) {
		t.Fatal("runtime A should enable Claude compaction")
	}
	if runtimeCapableB.SupportsClaudeCompactionWithContext(agentB.requestContext(context.Background()), agentB.CurrentModel) {
		t.Fatal("runtime B should disable Claude compaction")
	}

	compatCapableA, ok := agentA.CurrentProvider.(api.ClaudeCompactionCapable)
	if !ok {
		t.Fatal("agent A provider should support Claude compaction compatibility checks")
	}
	compatCapableB, ok := agentB.CurrentProvider.(api.ClaudeCompactionCapable)
	if !ok {
		t.Fatal("agent B provider should support Claude compaction compatibility checks")
	}
	if !compatCapableA.SupportsClaudeCompaction() {
		t.Fatal("compat capability should reflect injected runtime config for agent A")
	}
	if compatCapableB.SupportsClaudeCompaction() {
		t.Fatal("compat capability should reflect injected runtime config for agent B")
	}
}

func TestAgentRuntime_SwitchProviderUsesRuntimeConfig(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	originalConfig := config.GetGlobalConfig()
	globalCfg := config.DefaultConfig()
	globalCfg.ProviderModels["claude"] = config.ProviderModelConfig{DefaultModel: "claude-opus-4-6"}
	globalCfg.Compression.ClaudeCompaction = false
	config.SetGlobalConfig(globalCfg)
	t.Cleanup(func() {
		config.SetGlobalConfig(originalConfig)
	})

	runtimeA := newIsolatedRuntime()
	runtimeB := newIsolatedRuntime()
	runtimeA.Config.ProviderModels["claude"] = config.ProviderModelConfig{DefaultModel: "claude-sonnet-4-6"}
	runtimeB.Config.ProviderModels["claude"] = config.ProviderModelConfig{DefaultModel: "claude-3-7-sonnet-latest"}
	runtimeA.Config.Compression.ClaudeCompaction = true
	runtimeB.Config.Compression.ClaudeCompaction = true

	agentA := newRuntimeTestAgent(t, runtimeA)
	agentB := newRuntimeTestAgent(t, runtimeB)

	if err := agentA.SwitchProvider("claude"); err != nil {
		t.Fatalf("switch provider for runtime A: %v", err)
	}
	if err := agentB.SwitchProvider("claude"); err != nil {
		t.Fatalf("switch provider for runtime B: %v", err)
	}

	if agentA.CurrentModel != "claude-sonnet-4-6" {
		t.Fatalf("runtime A current model = %q, want %q", agentA.CurrentModel, "claude-sonnet-4-6")
	}
	if agentB.CurrentModel != "claude-3-7-sonnet-latest" {
		t.Fatalf("runtime B current model = %q, want %q", agentB.CurrentModel, "claude-3-7-sonnet-latest")
	}

	runtimeCapableA, ok := agentA.CurrentProvider.(api.ClaudeCompactionRuntimeCapable)
	if !ok {
		t.Fatal("runtime A switched provider should support runtime Claude compaction checks")
	}
	runtimeCapableB, ok := agentB.CurrentProvider.(api.ClaudeCompactionRuntimeCapable)
	if !ok {
		t.Fatal("runtime B switched provider should support runtime Claude compaction checks")
	}

	if !runtimeCapableA.SupportsClaudeCompactionWithContext(agentA.requestContext(context.Background()), agentA.CurrentModel) {
		t.Fatal("runtime A switched provider should keep runtime config for compaction")
	}
	if runtimeCapableB.SupportsClaudeCompactionWithContext(agentB.requestContext(context.Background()), agentB.CurrentModel) {
		t.Fatal("runtime B switched provider should keep runtime-specific unsupported model")
	}
}

func TestAgentRuntime_SeparatesUIRuntimeAndAuditLogger(t *testing.T) {
	logDir, err := os.MkdirTemp(".", "runtime-ui-audit-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(logDir) })

	outA := &bytes.Buffer{}
	errA := &bytes.Buffer{}
	outB := &bytes.Buffer{}
	errB := &bytes.Buffer{}
	logA := filepath.Join(logDir, "runtime-a.jsonl")
	logB := filepath.Join(logDir, "runtime-b.jsonl")

	runtimeA := newIsolatedRuntime()
	runtimeB := newIsolatedRuntime()
	runtimeA.AutoApprove = true
	runtimeB.AutoApprove = true
	runtimeA.UI = ui.NewRuntime(strings.NewReader(""), outA, errA)
	runtimeB.UI = ui.NewRuntime(strings.NewReader(""), outB, errB)
	runtimeA.AuditLogger = audit.NewLoggerWithPath(logA, true)
	runtimeB.AuditLogger = audit.NewLoggerWithPath(logB, true)
	runtimeA.Registry.Register(&runtimeProbeTool{})
	runtimeB.Registry.Register(&runtimeProbeTool{})

	agentA := newRuntimeTestAgent(t, runtimeA)
	agentB := newRuntimeTestAgent(t, runtimeB)

	ctxA := agentA.requestContext(context.Background())
	ctxB := agentB.requestContext(context.Background())

	api.PrintAIHeaderWithContext(ctxA)
	api.PrintAIHeaderWithContext(ctxB)
	if got := outA.String(); !strings.Contains(got, "💬") {
		t.Fatalf("runtime A UI output should receive header, got %q", got)
	}
	if got := outB.String(); !strings.Contains(got, "💬") {
		t.Fatalf("runtime B UI output should receive header, got %q", got)
	}

	spinnerA := api.StartThinkingSpinner(ctxA, false, "")
	spinnerB := api.StartThinkingSpinner(ctxB, false, "")
	if agentA.ui().CurrentSpinner() != spinnerA {
		t.Fatal("runtime A should keep its own spinner")
	}
	if agentB.ui().CurrentSpinner() != spinnerB {
		t.Fatal("runtime B should keep its own spinner")
	}

	agentA.ui().StopSpinner()
	if !spinnerB.IsActive() {
		t.Fatal("stopping runtime A spinner must not stop runtime B spinner")
	}
	agentB.ui().StopSpinner()

	probeA := &tools.ToolCall{
		Tool:    "runtime_probe_test",
		Args:    map[string]string{"path": "path-a"},
		RawArgs: map[string]any{"path": "path-a"},
	}
	probeB := &tools.ToolCall{
		Tool:    "runtime_probe_test",
		Args:    map[string]string{"path": "path-b"},
		RawArgs: map[string]any{"path": "path-b"},
	}
	_ = executeRuntimeTool(agentA, strings.NewReader(""), probeA)
	_ = executeRuntimeTool(agentB, strings.NewReader(""), probeB)

	dataA, err := os.ReadFile(logA)
	if err != nil {
		t.Fatalf("read runtime A audit log: %v", err)
	}
	dataB, err := os.ReadFile(logB)
	if err != nil {
		t.Fatalf("read runtime B audit log: %v", err)
	}

	if !strings.Contains(string(dataA), `"path":"path-a"`) || strings.Contains(string(dataA), `"path":"path-b"`) {
		t.Fatalf("runtime A audit log should only contain runtime A execution, got %s", string(dataA))
	}
	if !strings.Contains(string(dataB), `"path":"path-b"`) || strings.Contains(string(dataB), `"path":"path-a"`) {
		t.Fatalf("runtime B audit log should only contain runtime B execution, got %s", string(dataB))
	}

	execCtxA := agentA.toolExecutionContext(nil, nil, nil)
	if execCtxA.Stdout != runtimeA.UI.Output() || execCtxA.Stderr != runtimeA.UI.ErrorOutput() {
		t.Fatal("tool execution context should inherit runtime UI writers")
	}
}
