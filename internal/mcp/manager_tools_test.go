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
