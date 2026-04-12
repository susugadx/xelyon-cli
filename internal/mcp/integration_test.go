package mcp

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestManager_Connect_RegistersFilteredTools(t *testing.T) {
	command, args := mcpHelperCommand(t)

	manager := NewManager()
	var output bytes.Buffer
	manager.SetOutput(&output)
	manager.config = &Config{
		MCPServers: map[string]ServerConfig{
			"helper": {
				Command: command,
				Args:    args,
				Env: map[string]string{
					"GO_WANT_XELYON_MCP_HELPER": "1",
				},
				Tools: &ToolsFilter{
					Exclude: []string{"hidden"},
				},
			},
		},
	}
	t.Cleanup(manager.Close)

	if err := manager.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	if len(manager.sessions) != 1 {
		t.Fatalf("len(sessions) = %d, want 1", len(manager.sessions))
	}
	if manager.sessions["helper"] == nil {
		t.Fatal("sessions[helper] should be initialized")
	}
	if len(manager.tools) != 1 {
		t.Fatalf("len(tools) = %d, want 1", len(manager.tools))
	}
	if manager.tools[0].Name != "echo" {
		t.Fatalf("tools[0].Name = %q, want echo", manager.tools[0].Name)
	}
	if !bytes.Contains(manager.tools[0].InputSchema, []byte(`"name"`)) {
		t.Fatalf("InputSchema = %s, want to contain field name", manager.tools[0].InputSchema)
	}

	result, err := manager.CallTool(context.Background(), "helper", "echo", map[string]any{"name": "tester"})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	if result != "Hello tester\nFrom helper\n" {
		t.Fatalf("CallTool() = %q, want %q", result, "Hello tester\nFrom helper\n")
	}
	if !strings.Contains(output.String(), "filtered out") {
		t.Fatalf("Connect output = %q, want filtered out message", output.String())
	}
}

func TestManager_Reconnect_ReplacesServerToolsAndUpdatesHealth(t *testing.T) {
	command, args := mcpHelperCommand(t)

	manager := NewManager()
	var output bytes.Buffer
	manager.SetOutput(&output)
	manager.config = &Config{
		MCPServers: map[string]ServerConfig{
			"helper": {
				Command: command,
				Args:    args,
				Env: map[string]string{
					"GO_WANT_XELYON_MCP_HELPER": "1",
				},
			},
		},
	}
	t.Cleanup(manager.Close)

	if err := manager.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	if len(manager.tools) != 2 {
		t.Fatalf("len(tools) after Connect = %d, want 2", len(manager.tools))
	}

	manager.tools = append(manager.tools, MCPTool{ServerName: "other", Name: "persist"})
	output.Reset()

	manager.config.MCPServers["helper"] = ServerConfig{
		Command: command,
		Args:    args,
		Env: map[string]string{
			"GO_WANT_XELYON_MCP_HELPER": "1",
		},
		Tools: &ToolsFilter{
			Include: []string{"echo"},
		},
	}

	if err := manager.Reconnect(context.Background(), "helper"); err != nil {
		t.Fatalf("Reconnect() error = %v", err)
	}

	var helperToolNames []string
	for _, tool := range manager.tools {
		if tool.ServerName == "helper" {
			helperToolNames = append(helperToolNames, tool.Name)
		}
	}
	if len(helperToolNames) != 1 || helperToolNames[0] != "echo" {
		t.Fatalf("helper tools after Reconnect = %v, want [echo]", helperToolNames)
	}
	if len(manager.tools) != 2 {
		t.Fatalf("len(tools) after Reconnect = %d, want 2", len(manager.tools))
	}

	status := manager.HealthStatus()
	health := status["helper"]
	if !strings.Contains(health, "✅") {
		t.Fatalf("HealthStatus()[helper] = %q, want connected status", health)
	}
	if !strings.Contains(output.String(), "reconnected") {
		t.Fatalf("Reconnect output = %q, want reconnected message", output.String())
	}
}

func TestManager_CallTool_RetriesToolErrorAndSucceeds(t *testing.T) {
	var attempts int
	manager, output := newInMemoryManagerWithTool(t, "unstable", func(_ context.Context, _ *sdkmcp.CallToolRequest, _ map[string]any) (*sdkmcp.CallToolResult, any, error) {
		attempts++
		if attempts == 1 {
			return &sdkmcp.CallToolResult{
				IsError: true,
				Content: []sdkmcp.Content{
					&sdkmcp.TextContent{Text: "temporary failure"},
				},
			}, nil, nil
		}
		return &sdkmcp.CallToolResult{
			Content: []sdkmcp.Content{
				&sdkmcp.TextContent{Text: "recovered"},
			},
		}, nil, nil
	})

	got, err := manager.CallTool(context.Background(), "test-server", "unstable", nil)
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	if got != "recovered\n" {
		t.Fatalf("CallTool() = %q, want %q", got, "recovered\n")
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
	if !strings.Contains(output.String(), "attempt 1 failed") {
		t.Fatalf("CallTool output = %q, want retry warning", output.String())
	}
}

func newInMemoryManagerWithTool(
	t *testing.T,
	toolName string,
	handler func(context.Context, *sdkmcp.CallToolRequest, map[string]any) (*sdkmcp.CallToolResult, any, error),
) (*Manager, *bytes.Buffer) {
	t.Helper()

	server := sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:    "test-server",
		Version: "test",
	}, nil)
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        toolName,
		Description: "test tool",
	}, handler)

	ctx := context.Background()
	serverTransport, clientTransport := sdkmcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server.Connect() error = %v", err)
	}

	client := sdkmcp.NewClient(&sdkmcp.Implementation{
		Name:    "test-client",
		Version: "test",
	}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect() error = %v", err)
	}
	t.Cleanup(func() {
		_ = clientSession.Close()
		_ = serverSession.Wait()
	})

	manager := NewManager()
	var output bytes.Buffer
	manager.SetOutput(&output)
	manager.sessions["test-server"] = clientSession
	return manager, &output
}

func mcpHelperCommand(t *testing.T) (string, []string) {
	t.Helper()

	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}

	commandName := "xelyon-mcp-helper"
	binDir := t.TempDir()
	binPath := filepath.Join(binDir, commandName)
	if err := os.Symlink(exe, binPath); err != nil {
		copyExecutable(t, exe, binPath)
	}

	previousPath := os.Getenv("PATH")
	separator := string(os.PathListSeparator)
	if previousPath == "" {
		os.Setenv("PATH", binDir)
	} else {
		os.Setenv("PATH", binDir+separator+previousPath)
	}
	t.Cleanup(func() {
		os.Setenv("PATH", previousPath)
	})

	_, existed := allowedMCPCommands[commandName]
	previousAllowed := allowedMCPCommands[commandName]
	allowedMCPCommands[commandName] = true
	t.Cleanup(func() {
		if existed {
			allowedMCPCommands[commandName] = previousAllowed
			return
		}
		delete(allowedMCPCommands, commandName)
	})

	return commandName, []string{"-test.run=TestMCPHelperProcess", "--"}
}

func copyExecutable(t *testing.T, srcPath, dstPath string) {
	t.Helper()

	src, err := os.Open(srcPath)
	if err != nil {
		t.Fatalf("os.Open(%q) error = %v", srcPath, err)
	}
	defer src.Close()

	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		t.Fatalf("os.OpenFile(%q) error = %v", dstPath, err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		t.Fatalf("io.Copy() error = %v", err)
	}
}
