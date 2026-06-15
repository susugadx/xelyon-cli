package agent

import (
	"bytes"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/mcp"
	"github.com/susugadx/xelyon-cli/internal/mcpapproval"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestHandleMCPCommand_PrintsSnapshotOnlyStatus(t *testing.T) {
	var out bytes.Buffer
	agent := newMCPStatusTestAgent(t, &out)

	for _, input := range []string{"/mcp", "/mcp status"} {
		t.Run(input, func(t *testing.T) {
			out.Reset()
			beforeHistory := append([]api.Message(nil), agent.History...)
			beforeSessionMessageCount := 0
			if agent.session != nil {
				beforeSessionMessageCount = len(agent.session.Messages)
			}

			if !handleSpecialCommandForSurface(input, agent, commandcatalog.CommandSurfaceTUI) {
				t.Fatalf("handleSpecialCommandForSurface(%q) = false, want true", input)
			}

			got := out.String()
			for _, want := range []string{
				"MCP Status",
				"Runtime",
				"disabled (mcp.enabled=false)",
				"Tools",
				"1 visible / 2 registered, 1 omitted",
				"Visible: 1",
				"mcp_alpha_list (auto)",
				"Omitted: 1",
				"mcp_beta_big (token_budget_exceeded)",
			} {
				if !strings.Contains(got, want) {
					t.Fatalf("%s output missing %q:\n%s", input, want, got)
				}
			}
			for _, secret := range []string{"SECRET_DESCRIPTION", "SECRET_SCHEMA"} {
				if strings.Contains(got, secret) {
					t.Fatalf("%s output leaked %q:\n%s", input, secret, got)
				}
			}
			if len(agent.History) != len(beforeHistory) {
				t.Fatalf("Agent.History changed after %s", input)
			}
			if agent.session != nil && len(agent.session.Messages) != beforeSessionMessageCount {
				t.Fatalf("session.Messages changed after %s", input)
			}
		})
	}
}

func TestHandleMCPCommand_UnknownArgsPrintUsage(t *testing.T) {
	var out bytes.Buffer
	agent := newMCPStatusTestAgent(t, &out)

	if !handleSpecialCommandForSurface("/mcp refresh", agent, commandcatalog.CommandSurfaceTUI) {
		t.Fatal("handleSpecialCommandForSurface(/mcp refresh) = false, want true")
	}
	got := out.String()
	if !strings.Contains(got, "Usage: /mcp status") {
		t.Fatalf("unknown /mcp args should print usage:\n%s", got)
	}
	if strings.Contains(got, "MCP Status") {
		t.Fatalf("unknown /mcp args should not render status:\n%s", got)
	}
}

func TestHandleStatusCommandForSurface_IncludesMCPSummary(t *testing.T) {
	var out bytes.Buffer
	agent := newMCPStatusTestAgent(t, &out)

	if !handleStatusCommandForSurface(agent, commandcatalog.CommandSurfaceTUI) {
		t.Fatal("handleStatusCommandForSurface() = false, want true")
	}
	got := out.String()
	for _, want := range []string{
		"MCP",
		"0/0 servers connected, 1/2 tools visible, 1 omitted",
		"/mcp status for details",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("/status output missing %q:\n%s", want, got)
		}
	}
}

func newMCPStatusTestAgent(t *testing.T, out *bytes.Buffer) *Agent {
	t.Helper()

	cfg := config.CloneConfig(config.DefaultConfig())
	cfg.MCP.Enabled = false
	cfg.ProjectMap.Enabled = false
	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.UI = ui.NewRuntime(strings.NewReader(""), out, out)
	agent := NewAgentWithRuntime("gpt-5.4", &mockProvider{name: "openai"}, false, runtime)
	t.Cleanup(agent.Cleanup)

	tools := []mcp.MCPTool{
		{
			ServerName:  "alpha",
			Name:        "list",
			Description: "SECRET_DESCRIPTION",
			InputSchema: []byte(`{"type":"object","description":"SECRET_SCHEMA"}`),
			Approval:    mcpapproval.ModeAuto,
		},
		{
			ServerName:  "beta",
			Name:        "big",
			InputSchema: []byte(`{"type":"object"}`),
			Approval:    mcpapproval.ModeConfirm,
		},
	}
	setManagerToolsForTest(t, agent.mcpManager, tools)
	agent.mcpSurface = mcpToolSurfaceSelection{
		selected:        []mcp.MCPTool{tools[0]},
		omitted:         []mcpToolSurfaceOmission{{exportedName: "mcp_beta_big", serverName: "beta", toolName: "big", reason: "token_budget_exceeded"}},
		total:           len(tools),
		estimatedTokens: 42,
	}
	return agent
}
