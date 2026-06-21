package agent

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/mcp"
)

func TestSelectMCPToolSurfaceRoundRobinBudget(t *testing.T) {
	mcpTools := []mcp.MCPTool{
		{ServerName: "zeta", Name: "z3", Description: "z3"},
		{ServerName: "alpha", Name: "a2", Description: "a2"},
		{ServerName: "zeta", Name: "z1", Description: "z1"},
		{ServerName: "alpha", Name: "a1", Description: "a1"},
		{ServerName: "beta", Name: "b1", Description: "b1"},
	}

	selection := selectMCPToolSurfaceWithBudget("gpt-4o", mcpTools, mcpToolSurfaceBudget{
		MaxTools:              4,
		EstimatedTokens:       32000,
		MaxSchemaBytesPerTool: defaultMCPToolSurfaceBudget().MaxSchemaBytesPerTool,
	})

	got := exportedMCPToolNamesForTest(selection.selected)
	want := []string{
		"mcp_alpha_a1",
		"mcp_beta_b1",
		"mcp_zeta_z1",
		"mcp_alpha_a2",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("selected tools = %#v, want %#v", got, want)
	}
	omitted := selection.omittedExportedNames()
	if !reflect.DeepEqual(omitted, []string{"mcp_zeta_z3"}) {
		t.Fatalf("omitted tools = %#v, want zeta z3", omitted)
	}
}

func TestMCPToolSurfaceBudgetOmitsOversizedSchema(t *testing.T) {
	mcpTools := []mcp.MCPTool{
		{ServerName: "alpha", Name: "small", Description: "Small", InputSchema: json.RawMessage(`{"type":"object"}`)},
		{ServerName: "alpha", Name: "huge", Description: "Huge", InputSchema: json.RawMessage(strings.Repeat("x", 200))},
	}

	selection := selectMCPToolSurfaceWithBudget("gpt-4o", mcpTools, mcpToolSurfaceBudget{
		MaxTools:              10,
		EstimatedTokens:       32000,
		MaxSchemaBytesPerTool: 64,
	})

	if got := exportedMCPToolNamesForTest(selection.selected); !reflect.DeepEqual(got, []string{"mcp_alpha_small"}) {
		t.Fatalf("selected tools = %#v, want small only", got)
	}
	if got := selection.omittedExportedNames(); !reflect.DeepEqual(got, []string{"mcp_alpha_huge"}) {
		t.Fatalf("omitted tools = %#v, want huge only", got)
	}
	if selection.omitted[0].reason != "schema_too_large" {
		t.Fatalf("omission reason = %q, want schema_too_large", selection.omitted[0].reason)
	}
}

func TestMCPToolSurfaceBudgetSkipsTokenEstimateForOversizedSchema(t *testing.T) {
	hugeSchema := json.RawMessage(`{"type":"object","properties":{"payload":{"type":"string","description":"` + strings.Repeat("x", 128) + `"}}}`)
	mcpTools := []mcp.MCPTool{
		{ServerName: "alpha", Name: "small", Description: "Small", InputSchema: json.RawMessage(`{"type":"object"}`)},
		{ServerName: "alpha", Name: "huge", Description: "Huge", InputSchema: hugeSchema},
	}

	selection := selectMCPToolSurfaceWithBudget("gpt-4o", mcpTools, mcpToolSurfaceBudget{
		MaxTools:              10,
		EstimatedTokens:       32000,
		MaxSchemaBytesPerTool: 64,
	})

	if got := exportedMCPToolNamesForTest(selection.selected); !reflect.DeepEqual(got, []string{"mcp_alpha_small"}) {
		t.Fatalf("selected tools = %#v, want small only", got)
	}
	if got := selection.omittedExportedNames(); !reflect.DeepEqual(got, []string{"mcp_alpha_huge"}) {
		t.Fatalf("omitted tools = %#v, want huge only", got)
	}
	if selection.omitted[0].reason != "schema_too_large" {
		t.Fatalf("omission reason = %q, want schema_too_large", selection.omitted[0].reason)
	}
	if selection.omitted[0].estimatedTokens != 0 {
		t.Fatalf("oversized schema estimated tokens = %d, want 0 before schema parsing", selection.omitted[0].estimatedTokens)
	}
}

func TestMCPToolSurfaceBudgetCanOmitAllToolsWhenTokenBudgetExceeded(t *testing.T) {
	mcpTools := []mcp.MCPTool{
		{ServerName: "alpha", Name: "huge", Description: strings.Repeat("large description ", 50)},
	}

	selection := selectMCPToolSurfaceWithBudget("gpt-4o", mcpTools, mcpToolSurfaceBudget{
		MaxTools:              10,
		EstimatedTokens:       1,
		MaxSchemaBytesPerTool: defaultMCPToolSurfaceBudget().MaxSchemaBytesPerTool,
	})

	if len(selection.selected) != 0 {
		t.Fatalf("selected tools = %#v, want empty when first tool exceeds token budget", exportedMCPToolNamesForTest(selection.selected))
	}
	if got := selection.omittedExportedNames(); !reflect.DeepEqual(got, []string{"mcp_alpha_huge"}) {
		t.Fatalf("omitted tools = %#v, want huge only", got)
	}
	if selection.omitted[0].reason != "token_budget_exceeded" {
		t.Fatalf("omission reason = %q, want token_budget_exceeded", selection.omitted[0].reason)
	}
}

func TestMCPToolSurfaceAnalysisReportsMetricsAndRecommendations(t *testing.T) {
	mcpTools := []mcp.MCPTool{
		{ServerName: "alpha", Name: "one", Description: "One", InputSchema: json.RawMessage(`{"type":"object"}`)},
		{ServerName: "alpha", Name: "two", Description: "Two", InputSchema: json.RawMessage(`{"type":"object"}`)},
	}
	selection := selectMCPToolSurfaceWithBudget("gpt-4o", mcpTools, mcpToolSurfaceBudget{
		MaxTools:              1,
		EstimatedTokens:       32000,
		MaxSchemaBytesPerTool: defaultMCPToolSurfaceBudget().MaxSchemaBytesPerTool,
	})

	report := selection.analysis()
	if report.TotalTools != 2 || report.RegisteredTools != 2 || report.VisibleTools != 1 || report.OmittedTools != 1 {
		t.Fatalf("surface report counts = %+v, want total=2 registered=2 visible=1 omitted=1", report)
	}
	if report.EstimatedTokens <= 0 {
		t.Fatalf("EstimatedTokens = %d, want selected tool estimate", report.EstimatedTokens)
	}
	if len(report.Servers) != 1 || report.Servers[0].ServerName != "alpha" || report.Servers[0].OmittedTools != 1 {
		t.Fatalf("server summaries = %#v, want alpha omitted summary", report.Servers)
	}
	if len(report.OmittedReasons) != 1 || report.OmittedReasons[0].Reason != "tool_count_budget_exceeded" {
		t.Fatalf("omitted reasons = %#v, want tool_count_budget_exceeded", report.OmittedReasons)
	}
	if len(report.LargestSchemaTools) == 0 || report.LargestSchemaTools[0].ExportedName == "" {
		t.Fatalf("largest schema tools = %#v, want exported metric names", report.LargestSchemaTools)
	}
	if len(report.Recommendations) != 1 || report.Recommendations[0].ServerName != "alpha" {
		t.Fatalf("recommendations = %#v, want alpha narrowing recommendation", report.Recommendations)
	}
}

func TestCurrentMCPToolSurfaceUsesConfigBudget(t *testing.T) {
	cfg := config.CloneConfig(config.DefaultConfig())
	cfg.MCP.SurfaceBudget.MaxTools = 1
	runtime := NewAgentRuntimeWithConfig(cfg)
	manager := mcp.NewManager()
	tools := []mcp.MCPTool{
		{ServerName: "alpha", Name: "one", Description: "One"},
		{ServerName: "alpha", Name: "two", Description: "Two"},
	}
	setManagerToolsForTest(t, manager, tools)
	agent := &Agent{
		CurrentModel: "gpt-4o",
		Runtime:      runtime,
		mcpManager:   manager,
	}

	selection := agent.currentMCPToolSurface()
	if got := exportedMCPToolNamesForTest(selection.selected); !reflect.DeepEqual(got, []string{"mcp_alpha_one"}) {
		t.Fatalf("selected tools = %#v, want one tool from config max_tools=1", got)
	}
	if got := selection.omittedExportedNames(); !reflect.DeepEqual(got, []string{"mcp_alpha_two"}) {
		t.Fatalf("omitted tools = %#v, want second tool omitted", got)
	}
	if selection.budget.MaxTools != 1 {
		t.Fatalf("selection budget = %#v, want config max_tools=1", selection.budget)
	}
}

func TestCurrentMCPToolSurfaceInvalidatesCacheWhenToolsChangeWithSameCount(t *testing.T) {
	cfg := config.CloneConfig(config.DefaultConfig())
	runtime := NewAgentRuntimeWithConfig(cfg)
	manager := mcp.NewManager()
	setManagerToolsForTest(t, manager, []mcp.MCPTool{
		{ServerName: "alpha", Name: "one", Description: "One"},
	})
	agent := &Agent{
		CurrentModel: "gpt-4o",
		Runtime:      runtime,
		mcpManager:   manager,
	}
	agent.refreshMCPToolSurface()

	setManagerToolsForTest(t, manager, []mcp.MCPTool{
		{ServerName: "alpha", Name: "two", Description: "Two"},
	})

	selection := agent.currentMCPToolSurface()
	if got := exportedMCPToolNamesForTest(selection.selected); !reflect.DeepEqual(got, []string{"mcp_alpha_two"}) {
		t.Fatalf("selected tools = %#v, want refreshed tool after same-count manager update", got)
	}
}
