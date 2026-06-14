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
	}, nil)

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
		}, &ToolsFilter{Include: []string{"tool.one"}})

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
		}, &ToolsFilter{Exclude: []string{"tool.one"}})

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
	defaultCtx, defaultCancel := mcpServerOperationContext(context.Background())
	defer defaultCancel()
	defaultDeadline, ok := defaultCtx.Deadline()
	if !ok {
		t.Fatal("default operation context should have deadline")
	}
	defaultRemaining := time.Until(defaultDeadline)
	if defaultRemaining <= 29*time.Second || defaultRemaining > defaultMCPServerOperationTimeout {
		t.Fatalf("default deadline remaining = %v, want near %v", defaultRemaining, defaultMCPServerOperationTimeout)
	}

	parentCtx, parentCancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer parentCancel()
	parentDeadline, _ := parentCtx.Deadline()
	childCtx, childCancel := mcpServerOperationContext(parentCtx)
	defer childCancel()
	childDeadline, ok := childCtx.Deadline()
	if !ok {
		t.Fatal("child operation context should have deadline")
	}
	if !childDeadline.Equal(parentDeadline) {
		t.Fatalf("child deadline = %v, want parent deadline %v", childDeadline, parentDeadline)
	}
}
