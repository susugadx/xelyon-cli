package agent

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/mcp"
	"github.com/susugadx/xelyon-cli/internal/mcpapproval"
	"github.com/susugadx/xelyon-cli/internal/mcptool"
	"github.com/susugadx/xelyon-cli/internal/prompt"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestConfigureMCPTools_SetMCPToolsCalledOnceAndConverted(t *testing.T) {
	inputSchema := json.RawMessage(`{"type":"object","properties":{"id":{"type":"string"}},"required":["id"]}`)
	mcpTools := []mcp.MCPTool{
		{
			ServerName:  "github",
			Name:        "get_issue",
			Description: "Get issue",
			InputSchema: inputSchema,
		},
	}

	for _, providerName := range []string{"openai", "gemini", "deepseek"} {
		t.Run(providerName, func(t *testing.T) {
			p := &mockMCPProvider{name: providerName}

			configureMCPTools(p, mcpTools, nil)

			if p.setMCPToolsCalls != 1 {
				t.Fatalf("SetMCPTools calls = %d, want 1", p.setMCPToolsCalls)
			}
			if len(p.lastTools) != 1 {
				t.Fatalf("registered tools = %d, want 1", len(p.lastTools))
			}

			got := p.lastTools[0]
			if got.Name != "mcp_github_get_issue" {
				t.Fatalf("tool name = %q, want %q", got.Name, "mcp_github_get_issue")
			}
			if got.Description != "Get issue" {
				t.Fatalf("tool description = %q, want %q", got.Description, "Get issue")
			}

			if got.Parameters == nil {
				t.Fatalf("tool parameters should not be nil")
			}
			if _, ok := got.Parameters["type"]; !ok {
				t.Fatalf("tool parameters should include 'type'")
			}
			props, ok := got.Parameters["properties"].(map[string]any)
			if !ok {
				t.Fatalf("tool parameters['properties'] type = %T, want map[string]any", got.Parameters["properties"])
			}
			if _, ok := props["id"]; !ok {
				t.Fatalf("tool parameters['properties'] should include 'id'")
			}

			req, ok := got.Parameters["required"].([]any)
			if !ok {
				t.Fatalf("tool parameters['required'] type = %T, want []any", got.Parameters["required"])
			}
			found := false
			for _, v := range req {
				if s, ok := v.(string); ok && s == "id" {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("tool parameters['required'] should include 'id'")
			}
		})
	}
}

func TestConfigureMCPToolsClearsProviderWhenNoMCPToolsSelected(t *testing.T) {
	p := &mockMCPProvider{
		name: "openai",
		lastTools: []api.ToolDefinition{{
			Name: "mcp_stale_tool",
		}},
	}

	configureMCPTools(p, nil, nil)

	if p.setMCPToolsCalls != 1 {
		t.Fatalf("SetMCPTools calls = %d, want 1", p.setMCPToolsCalls)
	}
	if len(p.lastTools) != 0 {
		t.Fatalf("registered tools = %#v, want empty MCP provider surface", p.lastTools)
	}
}

func TestDeniedMCPToolsDoNotReachPromptProviderOrRegistrySurface(t *testing.T) {
	mcpTools := []mcp.MCPTool{
		{ServerName: "github", Name: "list_issues", Description: "List issues", Approval: mcpapproval.ModeAuto},
		{ServerName: "github", Name: "delete_repository", Description: "Delete repository", Approval: mcpapproval.ModeDeny},
	}

	selection := selectMCPToolSurfaceWithBudget("gpt-4o", mcpTools, mcpToolSurfaceBudget{
		MaxTools:              10,
		EstimatedTokens:       32000,
		MaxSchemaBytesPerTool: 1024,
	})
	if got := exportedMCPToolNamesForTest(selection.selected); !reflect.DeepEqual(got, []string{"mcp_github_list_issues"}) {
		t.Fatalf("selected MCP tools = %#v, want only allowed tool", got)
	}

	promptText := buildMCPToolsPromptForTools(mcpTools)
	if strings.Contains(promptText, "mcp_github_delete_repository") {
		t.Fatalf("prompt contains denied MCP tool:\n%s", promptText)
	}

	provider := &mockMCPProvider{name: "openai"}
	configureMCPTools(provider, mcpTools, nil)
	if got := toolDefinitionNamesForTest(provider.lastTools); !reflect.DeepEqual(got, []string{"mcp_github_list_issues"}) {
		t.Fatalf("provider MCP tools = %#v, want only allowed tool", got)
	}

	registry := tools.NewRegistry()
	mcptool.RegisterToRegistry(registry, mcpSurfaceTestCaller{}, mcpToolDefinitions(mcpTools))
	if registry.GetTool("mcp_github_delete_repository") != nil {
		t.Fatal("registry should not contain denied MCP tool")
	}
	if registry.GetTool("mcp_github_list_issues") == nil {
		t.Fatal("registry should contain allowed MCP tool")
	}
}

func TestMCPToolSurfaceBudgetExcludesProviderPromptAndRegistrySurface(t *testing.T) {
	mcpTools := []mcp.MCPTool{
		{ServerName: "alpha", Name: "one", Description: "One"},
		{ServerName: "alpha", Name: "two", Description: "Two"},
		{ServerName: "alpha", Name: "three", Description: "Three"},
	}
	selection := selectMCPToolSurfaceWithBudget("gpt-4o", mcpTools, mcpToolSurfaceBudget{
		MaxTools:              2,
		EstimatedTokens:       32000,
		MaxSchemaBytesPerTool: defaultMCPToolSurfaceBudget().MaxSchemaBytesPerTool,
	})

	provider := &mockMCPProvider{name: "openai"}
	configureMCPTools(provider, selection.selectedTools(), nil)
	if got := toolDefinitionNamesForTest(provider.lastTools); !reflect.DeepEqual(got, []string{"mcp_alpha_one", "mcp_alpha_three"}) {
		t.Fatalf("provider MCP tools = %#v, want selected only", got)
	}

	promptText := buildMCPToolsPromptForTools(selection.selectedTools())
	if strings.Contains(promptText, "mcp_alpha_two") {
		t.Fatalf("prompt should omit budget-excluded tool:\n%s", promptText)
	}
	for _, want := range []string{"mcp_alpha_one", "mcp_alpha_three"} {
		if !strings.Contains(promptText, want) {
			t.Fatalf("prompt missing selected tool %q:\n%s", want, promptText)
		}
	}

	registry := tools.NewRegistry()
	mcptool.RegisterToRegistry(registry, mcpSurfaceTestCaller{}, mcpToolDefinitions(mcpTools))
	registry.SetExcludedTools(selection.omittedExportedNames())
	if got := toolDefinitionNamesForTest(registry.GetAPIToolDefinitions()); !reflect.DeepEqual(got, []string{"mcp_alpha_one", "mcp_alpha_three"}) {
		t.Fatalf("registry API tools = %#v, want selected only", got)
	}
	if result := registry.ExecuteDetailedWithContext(tools.ExecutionContext{}, &tools.ToolCall{Tool: "mcp_alpha_two"}); !result.Error {
		t.Fatalf("budget-excluded MCP tool execution result = %#v, want error", result)
	}
}

func TestMCPExportedNameConsistentAcrossPromptProviderAndRegistry(t *testing.T) {
	const wantName = "mcp_github_server_create_issue"
	inputSchema := json.RawMessage(`{"type":"object","properties":{"title":{"type":"string"}}}`)
	mcpTools := []mcp.MCPTool{{
		ServerName:  "github.server",
		Name:        "create-issue",
		Description: "Create issue",
		InputSchema: inputSchema,
	}}

	promptText := prompt.BuildMCPToolsPrompt([]prompt.MCPTool{{
		ServerName:  mcpTools[0].ServerName,
		Name:        mcpTools[0].Name,
		Description: mcpTools[0].Description,
	}})
	if !strings.Contains(promptText, wantName) {
		t.Fatalf("prompt = %q, want exported MCP name %s", promptText, wantName)
	}
	if strings.Contains(promptText, "Create issue") {
		t.Fatalf("prompt should not duplicate MCP tool description:\n%s", promptText)
	}

	provider := &mockMCPProvider{name: "openai"}
	configureMCPTools(provider, mcpTools, nil)
	if len(provider.lastTools) != 1 || provider.lastTools[0].Name != wantName {
		t.Fatalf("provider tools = %#v, want exported MCP name %s", provider.lastTools, wantName)
	}

	registry := tools.NewRegistry()
	mcptool.RegisterToRegistry(registry, mcpSurfaceTestCaller{}, mcpToolDefinitions(mcpTools))
	if registry.GetTool(wantName) == nil {
		t.Fatalf("registry missing exported MCP tool %s", wantName)
	}
}

func TestMCPToolDefinitionsPreserveCallTimeoutForRegistry(t *testing.T) {
	mcpTools := []mcp.MCPTool{{
		ServerName:  "github",
		Name:        "slow",
		Description: "Slow tool",
		CallTimeout: 7 * time.Minute,
	}}

	defs := mcpToolDefinitions(mcpTools)
	if len(defs) != 1 {
		t.Fatalf("definitions = %d, want 1", len(defs))
	}
	if defs[0].CallTimeout != 7*time.Minute {
		t.Fatalf("CallTimeout = %v, want 7m", defs[0].CallTimeout)
	}
}

func TestConfigureMCPTools_SkipsDuplicateExportedNames(t *testing.T) {
	mcpTools := []mcp.MCPTool{
		{
			ServerName:  "github-server",
			Name:        "get.issue",
			Description: "First",
			InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
		},
		{
			ServerName:  "github_server",
			Name:        "get_issue",
			Description: "Second",
			InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
		},
		{
			ServerName:  "github",
			Name:        "list_issues",
			Description: "List",
			InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
		},
	}
	p := &mockMCPProvider{name: "openai"}

	configureMCPTools(p, mcpTools, nil)

	if p.setMCPToolsCalls != 1 {
		t.Fatalf("SetMCPTools calls = %d, want 1", p.setMCPToolsCalls)
	}
	if len(p.lastTools) != 2 {
		t.Fatalf("registered tools = %d, want 2: %#v", len(p.lastTools), p.lastTools)
	}
	if p.lastTools[0].Name != "mcp_github_server_get_issue" || p.lastTools[0].Description != "First" {
		t.Fatalf("first tool = %#v, want first duplicate to win", p.lastTools[0])
	}
	if p.lastTools[1].Name != "mcp_github_list_issues" {
		t.Fatalf("second tool = %#v, want list_issues", p.lastTools[1])
	}
}

func TestConfigureMCPTools_DebugLoggingByProvider(t *testing.T) {
	inputSchema := json.RawMessage(`{"type":"object","properties":{"id":{"type":"string"}},"required":["id"]}`)
	mcpTools := []mcp.MCPTool{
		{
			ServerName:  "github",
			Name:        "get_issue",
			Description: "Get issue",
			InputSchema: inputSchema,
		},
	}

	{
		t.Setenv("XELYON_DEBUG_OPENAI", "1")
		var buf bytes.Buffer
		p := &mockMCPProvider{name: "openai"}
		configureMCPTools(p, mcpTools, &buf)
		if p.setMCPToolsCalls != 1 {
			t.Fatalf("SetMCPTools calls = %d, want 1", p.setMCPToolsCalls)
		}
		if !strings.Contains(buf.String(), "[DEBUG OpenAI] MCP tool registered: mcp_github_get_issue\n") {
			t.Fatalf("debug log should contain OpenAI label, got: %q", buf.String())
		}
	}

	{
		t.Setenv("XELYON_DEBUG_OPENAI", "")
		t.Setenv("XELYON_DEBUG_GEMINI", "1")
		var buf bytes.Buffer
		p := &mockMCPProvider{name: "gemini"}
		configureMCPTools(p, mcpTools, &buf)
		if p.setMCPToolsCalls != 1 {
			t.Fatalf("SetMCPTools calls = %d, want 1", p.setMCPToolsCalls)
		}
		if !strings.Contains(buf.String(), "[DEBUG Gemini] MCP tool registered: mcp_github_get_issue\n") {
			t.Fatalf("debug log should contain Gemini label, got: %q", buf.String())
		}
	}

	{
		t.Setenv("XELYON_DEBUG_GEMINI", "")
		t.Setenv("XELYON_DEBUG_OPENAI", "1")
		var buf bytes.Buffer
		p := &mockMCPProvider{name: "deepseek"}
		configureMCPTools(p, mcpTools, &buf)
		if p.setMCPToolsCalls != 1 {
			t.Fatalf("SetMCPTools calls = %d, want 1", p.setMCPToolsCalls)
		}
		if buf.Len() != 0 {
			t.Fatalf("debug log should be empty for non OpenAI/Gemini provider, got: %q", buf.String())
		}
	}
}

func TestConfigureMCPTools_NoPanicWithNilWriter(t *testing.T) {
	t.Setenv("XELYON_DEBUG_OPENAI", "1")

	inputSchema := json.RawMessage(`{"type":"object","properties":{"id":{"type":"string"}},"required":["id"]}`)
	mcpTools := []mcp.MCPTool{
		{
			ServerName:  "github",
			Name:        "get_issue",
			Description: "Get issue",
			InputSchema: inputSchema,
		},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("configureMCPTools should not panic with nil writer, but panicked: %v", r)
		}
	}()

	p := &mockMCPProvider{name: "openai"}
	configureMCPTools(p, mcpTools, nil)
	if p.setMCPToolsCalls != 1 {
		t.Fatalf("SetMCPTools calls = %d, want 1", p.setMCPToolsCalls)
	}
}
