package agent

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/toolruntime"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/subagent"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestExecuteToolCallsWithParallel_PrintsParallelGroup(t *testing.T) {
	t.Setenv("XELYON_EDIT_TOOL", "str_replace")

	provider := &mockProvider{name: "test"}
	var out bytes.Buffer
	agent := NewAgentWithRuntime("test-model", provider, false, &AgentRuntime{
		UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
	})
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	toolCalls := []*tools.ToolCall{
		{
			ID:      "c1",
			Tool:    "read_file",
			Args:    testReadFileArgs("auto_compress.go"),
			RawArgs: testReadFileRawArgs("auto_compress.go"),
		},
		{
			ID:   "c2",
			Tool: "search_code",
			Args: map[string]string{
				"pattern": "maybeAutoCompress",
				"path":    ".",
			},
			RawArgs: map[string]any{
				"pattern": "maybeAutoCompress",
				"path":    ".",
			},
		},
	}

	agent.addToolCallsToHistory("test", toolCalls)

	agent.executeToolCallsWithParallel(context.Background(), toolCalls, nil, nil,
		func(_ int, _ *tools.ToolCall, _ toolruntime.Result) {})
	output := out.String()

	for _, want := range []string{
		"┌ Parallel (2 calls)",
		"📄 read_file: auto_compress.go",
		`🔍 search_code: "maybeAutoCompress" in . →`,
		"Found ",
		"└ Done:",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("parallel group output missing %q:\n%s", want, output)
		}
	}
}

func TestExecuteToolCallsWithParallel_ShowsSpinnerDuringParallelRun(t *testing.T) {
	t.Setenv("XELYON_EDIT_TOOL", "str_replace")

	provider := &mockProvider{name: "test"}
	runtime := NewAgentRuntime()
	runtime.UI = ui.NewRuntime(strings.NewReader(""), io.Discard, io.Discard)

	started := make(chan struct{}, 1)
	release := make(chan struct{})
	origTool := runtime.Registry.GetTool("read_file")
	runtime.Registry.Register(&blockingParallelTool{
		started: started,
		release: release,
	})

	agent := NewAgentWithRuntime("test-model", provider, false, runtime)
	t.Cleanup(func() {
		agent.Cleanup()
		if origTool != nil {
			runtime.Registry.Register(origTool)
		}
	})

	toolCalls := []*tools.ToolCall{{
		ID:      "c1",
		Tool:    "read_file",
		Args:    testReadFileArgs("a.go"),
		RawArgs: testReadFileRawArgs("a.go"),
	}}

	done := make(chan struct{})
	go func() {
		agent.executeToolCallsWithParallel(context.Background(), toolCalls, nil, nil, func(_ int, _ *tools.ToolCall, _ toolruntime.Result) {})
		close(done)
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for parallel tool to start")
	}

	spinner := agent.ui().CurrentSpinner()
	if spinner == nil || !spinner.IsActive() {
		t.Fatal("expected spinner to be active during parallel tool execution")
	}

	close(release)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for parallel execution to finish")
	}

	if spinner := agent.ui().CurrentSpinner(); spinner != nil {
		t.Fatal("expected spinner to be cleared after parallel execution")
	}
}

func TestExecuteToolCallsWithParallel_StopsSpinnerBeforeNonTUIReport(t *testing.T) {
	t.Setenv("XELYON_EDIT_TOOL", "str_replace")

	provider := &mockProvider{name: "test"}
	runtime := NewAgentRuntime()
	checkWriter := &spinnerOrderCheckWriter{}
	runtime.UI = ui.NewRuntime(strings.NewReader(""), checkWriter, checkWriter)

	agent := NewAgentWithRuntime("test-model", provider, false, runtime)
	checkWriter.agent = agent
	t.Cleanup(agent.Cleanup)

	toolCalls := []*tools.ToolCall{
		{
			ID:      "c1",
			Tool:    "read_file",
			Args:    testReadFileArgs("auto_compress.go"),
			RawArgs: testReadFileRawArgs("auto_compress.go"),
		},
		{
			ID:   "c2",
			Tool: "search_code",
			Args: map[string]string{
				"pattern": "maybeAutoCompress",
				"path":    ".",
			},
			RawArgs: map[string]any{
				"pattern": "maybeAutoCompress",
				"path":    ".",
			},
		},
	}

	agent.executeToolCallsWithParallel(context.Background(), toolCalls, nil, nil, func(_ int, _ *tools.ToolCall, _ toolruntime.Result) {})

	if !checkWriter.sawParallelGroup {
		t.Fatal("expected non-TUI parallel group output")
	}
	if checkWriter.groupWhileSpinnerOn {
		t.Fatal("parallel report should not run while spinner is still active")
	}
}

func TestExecuteToolCallsWithParallel_StopsSpinnerBeforeTUIReport(t *testing.T) {
	t.Setenv("XELYON_EDIT_TOOL", "str_replace")

	provider := &mockProvider{name: "test"}
	runtime := NewAgentRuntime()
	runtime.UI = ui.NewRuntime(strings.NewReader(""), io.Discard, io.Discard)

	started := make(chan struct{}, 1)
	release := make(chan struct{})
	origTool := runtime.Registry.GetTool("read_file")
	runtime.Registry.Register(&blockingParallelTool{
		started: started,
		release: release,
	})

	agent := NewAgentWithRuntime("test-model", provider, false, runtime)
	t.Cleanup(func() {
		agent.Cleanup()
		if origTool != nil {
			runtime.Registry.Register(origTool)
		}
	})

	toolResultCh := make(chan tools.ToolResultInfo)
	agent.tuiToolResultCh = toolResultCh

	toolCalls := []*tools.ToolCall{{
		ID:      "c1",
		Tool:    "read_file",
		Args:    testReadFileArgs("a.go"),
		RawArgs: testReadFileRawArgs("a.go"),
	}}

	done := make(chan struct{})
	go func() {
		agent.executeToolCallsWithParallel(context.Background(), toolCalls, nil, nil, func(_ int, _ *tools.ToolCall, _ toolruntime.Result) {})
		close(done)
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for parallel tool to start")
	}

	if spinner := agent.ui().CurrentSpinner(); spinner == nil || !spinner.IsActive() {
		t.Fatal("expected spinner to be active before TUI parallel report")
	}

	go func() {
		time.Sleep(10 * time.Millisecond)
		close(release)
	}()

	select {
	case <-toolResultCh:
		if spinner := agent.ui().CurrentSpinner(); spinner != nil {
			t.Fatal("TUI parallel report should not run while spinner is still active")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for TUI parallel tool result")
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for parallel execution to finish")
	}
}

// --- Test: Text-based (non-FC) tool calls also handled correctly ---

func TestExecuteToolWithSpinner_WaitAgentLiveView(t *testing.T) {
	prevNoColor := color.NoColor
	color.NoColor = true
	defer func() { color.NoColor = prevNoColor }()

	provider := &mockProvider{name: "openai"}
	cfg := config.DefaultConfig()
	var out bytes.Buffer
	manager := subagent.NewManagerWithOptions(subagent.ManagerOptions{
		RunHeadless: func(ctx context.Context, _ string, _ string, _ api.Provider, _ *config.Config) *subagent.RunResult {
			subagent.EmitEvent(ctx, subagent.SubAgentEvent{
				Tool:     "read_file",
				Phase:    "start",
				FilePath: "manager.go",
			})
			subagent.EmitEvent(ctx, subagent.SubAgentEvent{
				Tool:     "str_replace",
				Phase:    "start",
				FilePath: "manager.go",
			})
			subagent.EmitEvent(ctx, subagent.SubAgentEvent{
				Tool:     "str_replace",
				Phase:    "end",
				FilePath: "manager.go",
				Success:  true,
				Output:   "Successfully replaced text in manager.go (lines 58-59 → 58-60)",
				OldStr:   "id        string\nstatus    string",
				NewStr:   "id        string\ntaskType  string\nstatus    string",
			})
			return &subagent.RunResult{
				Status:         "completed",
				Response:       "done",
				ToolExecutions: 2,
				DurationMs:     1200,
			}
		},
		ProviderFactory: func(providerName string) (api.Provider, error) {
			return &mockProvider{name: providerName}, nil
		},
	})

	id, err := manager.Spawn(context.Background(), "inspect", "", "", "", provider, cfg)
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}

	runtime := &AgentRuntime{
		Config:          cfg,
		UI:              ui.NewRuntime(strings.NewReader(""), &out, &out),
		SubAgentManager: manager,
	}
	agent := NewAgentWithRuntime("test-model", provider, false, runtime)
	t.Cleanup(agent.Cleanup)

	toolCall := &tools.ToolCall{
		Tool:    "wait_agent",
		Args:    map[string]string{"ids": fmt.Sprintf(`["%s"]`, id)},
		RawArgs: map[string]any{"ids": []string{id}},
	}

	_, _ = agent.executeToolWithSpinner(context.Background(), toolCall)

	output := out.String()
	for _, want := range []string{
		fmt.Sprintf("%s │ 📖 read_file manager.go", id),
		fmt.Sprintf("%s │ 🔧 str_replace manager.go", id),
		"+ taskType  string",
		fmt.Sprintf("%s │ ✅ completed (2 tools, 1.2s)", id),
		"⏳ wait_agent: 1 agent",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func TestWaitAgentSpinnerMessage(t *testing.T) {
	tests := []struct {
		name string
		args map[string]string
		want string
	}{
		{
			name: "3 agents",
			args: map[string]string{"ids": `["a","b","c"]`},
			want: "Waiting for 3 agents...",
		},
		{
			name: "1 agent",
			args: map[string]string{"ids": `["x"]`},
			want: "Waiting for 1 agents...",
		},
		{
			name: "empty ids",
			args: map[string]string{"ids": `[]`},
			want: "Waiting for agents...",
		},
		{
			name: "invalid json",
			args: map[string]string{"ids": "not-json"},
			want: "Waiting for agents...",
		},
		{
			name: "no ids key",
			args: map[string]string{},
			want: "Waiting for agents...",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := waitAgentSpinnerMessage(tt.args)
			if got != tt.want {
				t.Errorf("waitAgentSpinnerMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParallelGroupSpinnerMessage_SpawnAndWait(t *testing.T) {
	tests := []struct {
		name  string
		calls []*tools.ToolCall
		want  string
	}{
		{
			name: "spawn_agent x3",
			calls: []*tools.ToolCall{
				{Tool: "spawn_agent"},
				{Tool: "spawn_agent"},
				{Tool: "spawn_agent"},
			},
			want: "Spawning 3 sub-agents...",
		},
		{
			name: "wait_agent with 5 ids",
			calls: []*tools.ToolCall{
				{Tool: "wait_agent", Args: map[string]string{"ids": `["a","b","c","d","e"]`}},
			},
			want: "Waiting for 5 agents...",
		},
		{
			name: "wait_agent without ids",
			calls: []*tools.ToolCall{
				{Tool: "wait_agent", Args: map[string]string{}},
			},
			want: "Waiting for agents...",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			indices := make([]int, len(tt.calls))
			for i := range indices {
				indices[i] = i
			}
			got := parallelGroupSpinnerMessage(tt.calls, indices)
			if got != tt.want {
				t.Errorf("parallelGroupSpinnerMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}
