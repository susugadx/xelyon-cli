package mcp

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestManagerBuildServerToolsSkipsSanitizedNameCollision(t *testing.T) {
	manager := NewManager()
	var output bytes.Buffer
	manager.SetOutput(&output)
	manager.tools = []MCPTool{
		{
			ServerName:  "server-a",
			Name:        "tool.one",
			Description: "first",
		},
	}

	got, summary := manager.buildServerTools("server_a", nil, []*sdkmcp.Tool{
		{Name: "tool_one", Description: "duplicate"},
		{Name: "kept", Description: "kept"},
	}, nil, defaultMCPToolCallTimeout)

	if summary.registered != 1 || summary.skipped != 1 {
		t.Fatalf("summary = %+v, want registered=1 skipped=1", summary)
	}
	if len(got) != 1 || got[0].Name != "kept" {
		t.Fatalf("buildServerTools() = %#v, want only kept tool", got)
	}
	if !strings.Contains(output.String(), `exported name "mcp_server_a_tool_one" already registered`) {
		t.Fatalf("warning output = %q, want collision warning", output.String())
	}
}

func TestManagerBuildServerToolsFiltersRawToolNameBeforeExportedCollision(t *testing.T) {
	t.Run("include uses raw tool name and then collision skips", func(t *testing.T) {
		manager := NewManager()
		manager.tools = []MCPTool{{
			ServerName:  "server_a",
			Name:        "tool_one",
			Description: "existing",
		}}

		got, summary := manager.buildServerTools("server-a", nil, []*sdkmcp.Tool{
			{Name: "tool.one", Description: "raw included but exported duplicate"},
			{Name: "tool_two", Description: "raw not included"},
		}, &ToolsFilter{Include: []string{"tool.one"}}, defaultMCPToolCallTimeout)

		if len(got) != 0 {
			t.Fatalf("buildServerTools() = %#v, want no registered tools", got)
		}
		if summary.registered != 0 || summary.skipped != 2 {
			t.Fatalf("summary = %+v, want registered=0 skipped=2", summary)
		}
	})

	t.Run("exclude uses raw tool name before collision check", func(t *testing.T) {
		manager := NewManager()
		var output bytes.Buffer
		manager.SetOutput(&output)
		manager.tools = []MCPTool{{
			ServerName:  "server_a",
			Name:        "tool_one",
			Description: "existing",
		}}

		got, summary := manager.buildServerTools("server-a", nil, []*sdkmcp.Tool{
			{Name: "tool.one", Description: "raw excluded duplicate"},
			{Name: "tool_two", Description: "kept"},
		}, &ToolsFilter{Exclude: []string{"tool.one"}}, defaultMCPToolCallTimeout)

		if len(got) != 1 || got[0].Name != "tool_two" {
			t.Fatalf("buildServerTools() = %#v, want only raw non-excluded tool", got)
		}
		if summary.registered != 1 || summary.skipped != 1 {
			t.Fatalf("summary = %+v, want registered=1 skipped=1", summary)
		}
		if strings.Contains(output.String(), "already registered") {
			t.Fatalf("warning output = %q, want raw excluded duplicate to skip collision warning", output.String())
		}
	})
}

func TestMCPServerOperationContextUsesDefaultAndCallerDeadline(t *testing.T) {
	defaultCtx, defaultCancel := mcpServerOperationContext(context.Background(), 0)
	defer defaultCancel()
	defaultDeadline, ok := defaultCtx.Deadline()
	if !ok {
		t.Fatal("default operation context should have deadline")
	}
	defaultRemaining := time.Until(defaultDeadline)
	if defaultRemaining <= defaultMCPServerOperationTimeout-time.Second || defaultRemaining > defaultMCPServerOperationTimeout {
		t.Fatalf("default deadline remaining = %v, want near %v", defaultRemaining, defaultMCPServerOperationTimeout)
	}

	parentCtx, parentCancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer parentCancel()
	parentDeadline, _ := parentCtx.Deadline()
	childCtx, childCancel := mcpServerOperationContext(parentCtx, defaultMCPServerOperationTimeout)
	defer childCancel()
	childDeadline, ok := childCtx.Deadline()
	if !ok {
		t.Fatal("child operation context should have deadline")
	}
	if !childDeadline.Equal(parentDeadline) {
		t.Fatalf("child deadline = %v, want parent deadline %v", childDeadline, parentDeadline)
	}
}

func TestMCPServerOperationContextUsesConfiguredTimeout(t *testing.T) {
	ctx, cancel := mcpServerOperationContext(context.Background(), 45*time.Second)
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("operation context should have deadline")
	}
	remaining := time.Until(deadline)
	if remaining <= 44*time.Second || remaining > 45*time.Second {
		t.Fatalf("deadline remaining = %v, want near 45s", remaining)
	}
}

func TestServerConfigTimeoutDurations(t *testing.T) {
	tests := []struct {
		name        string
		config      ServerConfig
		wantStartup time.Duration
		wantTool    time.Duration
	}{
		{
			name:        "defaults",
			config:      ServerConfig{},
			wantStartup: defaultMCPServerOperationTimeout,
			wantTool:    defaultMCPToolCallTimeout,
		},
		{
			name: "explicit values",
			config: ServerConfig{
				StartupTimeoutSeconds: 45,
				ToolTimeoutSeconds:    300,
			},
			wantStartup: 45 * time.Second,
			wantTool:    300 * time.Second,
		},
		{
			name: "negative values use defaults",
			config: ServerConfig{
				StartupTimeoutSeconds: -1,
				ToolTimeoutSeconds:    -1,
			},
			wantStartup: defaultMCPServerOperationTimeout,
			wantTool:    defaultMCPToolCallTimeout,
		},
		{
			name: "too large values are clamped",
			config: ServerConfig{
				StartupTimeoutSeconds: int(maxMCPServerOperationTimeout/time.Second) + 1,
				ToolTimeoutSeconds:    int(maxMCPToolCallTimeout/time.Second) + 1,
			},
			wantStartup: maxMCPServerOperationTimeout,
			wantTool:    maxMCPToolCallTimeout,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.config.startupTimeoutDuration(); got != tc.wantStartup {
				t.Fatalf("startupTimeoutDuration() = %v, want %v", got, tc.wantStartup)
			}
			if got := tc.config.toolTimeoutDuration(); got != tc.wantTool {
				t.Fatalf("toolTimeoutDuration() = %v, want %v", got, tc.wantTool)
			}
		})
	}
}

func TestManagerBuildServerToolsCarriesConfiguredCallTimeout(t *testing.T) {
	manager := NewManager()
	got, summary := manager.buildServerTools("server", nil, []*sdkmcp.Tool{
		{Name: "slow", Description: "slow"},
	}, nil, 5*time.Minute)

	if summary.registered != 1 || summary.skipped != 0 {
		t.Fatalf("summary = %+v, want registered=1 skipped=0", summary)
	}
	if len(got) != 1 {
		t.Fatalf("tools = %d, want 1", len(got))
	}
	if got[0].CallTimeout != 5*time.Minute {
		t.Fatalf("CallTimeout = %v, want 5m", got[0].CallTimeout)
	}
}
