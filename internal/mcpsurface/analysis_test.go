package mcpsurface

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestAnalyzeBuildsDeterministicServerReasonAndMetricSummary(t *testing.T) {
	report := Analyze([]Tool{
		{ServerName: "beta", ToolName: "two", ExportedName: "mcp_beta_two", Visible: true, SchemaBytes: 20, EstimatedTokens: 8},
		{ServerName: "alpha", ToolName: "hidden", ExportedName: "mcp_alpha_hidden", Registered: true, Visible: false, OmittedReason: "token_budget_exceeded", SchemaBytes: 30, EstimatedTokens: 11},
		{ServerName: "alpha", ToolName: "one", ExportedName: "mcp_alpha_one", Visible: true, SchemaBytes: 10, EstimatedTokens: 5},
	}, Options{TopLimit: 2, RecommendationToolLimit: 2})

	if report.TotalTools != 3 || report.RegisteredTools != 3 || report.VisibleTools != 2 || report.OmittedTools != 1 {
		t.Fatalf("counts = total:%d registered:%d visible:%d omitted:%d, want 3/3/2/1", report.TotalTools, report.RegisteredTools, report.VisibleTools, report.OmittedTools)
	}
	if report.EstimatedTokens != 24 {
		t.Fatalf("EstimatedTokens = %d, want all analyzed token sum 24", report.EstimatedTokens)
	}
	if report.SchemaBytes != 60 {
		t.Fatalf("SchemaBytes = %d, want all schema byte sum 60", report.SchemaBytes)
	}

	gotServers := []string{report.Servers[0].ServerName, report.Servers[1].ServerName}
	if !reflect.DeepEqual(gotServers, []string{"alpha", "beta"}) {
		t.Fatalf("server order = %#v, want alpha,beta", gotServers)
	}
	if report.Servers[0].TotalTools != 2 || report.Servers[0].RegisteredTools != 2 || report.Servers[0].VisibleTools != 1 || report.Servers[0].OmittedTools != 1 || report.Servers[0].EstimatedTokens != 16 {
		t.Fatalf("alpha summary = %+v, want total=2 registered=2 visible=1 omitted=1 estimated=16", report.Servers[0])
	}
	if report.Servers[0].OmittedReasons[0] != (ReasonCount{Reason: "token_budget_exceeded", Count: 1}) {
		t.Fatalf("alpha omitted reasons = %#v", report.Servers[0].OmittedReasons)
	}
	if report.OmittedReasons[0] != (ReasonCount{Reason: "token_budget_exceeded", Count: 1}) {
		t.Fatalf("omitted reasons = %#v", report.OmittedReasons)
	}

	if got := toolMetricNames(report.LargestSchemaTools); !reflect.DeepEqual(got, []string{"mcp_alpha_hidden", "mcp_beta_two"}) {
		t.Fatalf("largest schema tools = %#v", got)
	}
	if got := toolMetricNames(report.HighestEstimatedTokenTools); !reflect.DeepEqual(got, []string{"mcp_alpha_hidden", "mcp_beta_two"}) {
		t.Fatalf("highest token tools = %#v", got)
	}
	if len(report.Recommendations) != 1 || report.Recommendations[0].ServerName != "alpha" || !reflect.DeepEqual(report.Recommendations[0].IncludeTools, []string{"one"}) {
		t.Fatalf("recommendations = %#v, want alpha include one", report.Recommendations)
	}
}

func TestAnalyzeSeparatesTotalRegisteredVisibleAndOmitted(t *testing.T) {
	report := Analyze([]Tool{
		{ServerName: "helper", ToolName: "visible", ExportedName: "mcp_helper_visible", Visible: true},
		{ServerName: "helper", ToolName: "budgeted", ExportedName: "mcp_helper_budgeted", Registered: true, Visible: false, OmittedReason: "token_budget_exceeded"},
		{ServerName: "helper", ToolName: "filtered", ExportedName: "mcp_helper_filtered", Registered: false, Visible: false, OmittedReason: "filtered"},
	}, Options{})

	if report.TotalTools != 3 || report.RegisteredTools != 2 || report.VisibleTools != 1 || report.OmittedTools != 2 {
		t.Fatalf("report counts = %+v, want total=3 registered=2 visible=1 omitted=2", report)
	}
	if len(report.Servers) != 1 {
		t.Fatalf("servers = %#v, want one server", report.Servers)
	}
	server := report.Servers[0]
	if server.TotalTools != 3 || server.RegisteredTools != 2 || server.VisibleTools != 1 || server.OmittedTools != 2 {
		t.Fatalf("server counts = %+v, want total=3 registered=2 visible=1 omitted=2", server)
	}
}

func TestAnalyzeJSONDoesNotCarryRawSchemaOrDescription(t *testing.T) {
	report := Analyze([]Tool{{
		ServerName:      "alpha",
		ToolName:        "safe",
		ExportedName:    "mcp_alpha_safe",
		Visible:         true,
		SchemaBytes:     123,
		EstimatedTokens: 45,
	}}, Options{})

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	got := string(data)
	for _, secret := range []string{"SECRET_SCHEMA", "SECRET_DESCRIPTION"} {
		if strings.Contains(got, secret) {
			t.Fatalf("report JSON leaked %q:\n%s", secret, got)
		}
	}
}

func TestApplyBudgetOmitsByTokenAndSchema(t *testing.T) {
	reportTools := []Tool{
		{ServerName: "alpha", ToolName: "one", ExportedName: "mcp_alpha_one", Registered: true, Visible: true, SchemaBytes: 10, EstimatedTokens: 5},
		{ServerName: "alpha", ToolName: "two", ExportedName: "mcp_alpha_two", Registered: true, Visible: true, SchemaBytes: 10, EstimatedTokens: 5},
		{ServerName: "beta", ToolName: "huge_schema", ExportedName: "mcp_beta_huge_schema", Registered: true, Visible: true, SchemaBytes: 100, EstimatedTokens: 1},
		{ServerName: "beta", ToolName: "huge_tokens", ExportedName: "mcp_beta_huge_tokens", Registered: true, Visible: true, SchemaBytes: 10, EstimatedTokens: 100},
	}

	selection := ApplyBudget(reportTools, Budget{
		MaxTools:              10,
		EstimatedTokens:       20,
		MaxSchemaBytesPerTool: 50,
	})

	if got := toolNames(selection.Selected); !reflect.DeepEqual(got, []string{"mcp_alpha_one", "mcp_alpha_two"}) {
		t.Fatalf("selected = %#v, want alpha one/two", got)
	}
	reasons := map[string]string{}
	for _, omitted := range selection.Omitted {
		reasons[omitted.ExportedName] = omitted.OmittedReason
		if !omitted.Registered || omitted.Visible {
			t.Fatalf("omitted tool = %+v, want registered hidden", omitted)
		}
	}
	if reasons["mcp_beta_huge_schema"] != OmittedReasonSchemaTooLarge {
		t.Fatalf("schema omission reason = %q, want %q", reasons["mcp_beta_huge_schema"], OmittedReasonSchemaTooLarge)
	}
	if reasons["mcp_beta_huge_tokens"] != OmittedReasonTokenBudgetExceeded {
		t.Fatalf("token omission reason = %q, want %q", reasons["mcp_beta_huge_tokens"], OmittedReasonTokenBudgetExceeded)
	}
}

func TestApplyBudgetOmitsByToolCount(t *testing.T) {
	selection := ApplyBudget([]Tool{
		{ServerName: "alpha", ToolName: "one", ExportedName: "mcp_alpha_one", Registered: true, Visible: true, SchemaBytes: 10, EstimatedTokens: 1},
		{ServerName: "alpha", ToolName: "two", ExportedName: "mcp_alpha_two", Registered: true, Visible: true, SchemaBytes: 10, EstimatedTokens: 1},
		{ServerName: "alpha", ToolName: "three", ExportedName: "mcp_alpha_three", Registered: true, Visible: true, SchemaBytes: 10, EstimatedTokens: 1},
	}, Budget{MaxTools: 2, EstimatedTokens: 100, MaxSchemaBytesPerTool: 100})

	if got := toolNames(selection.Selected); !reflect.DeepEqual(got, []string{"mcp_alpha_one", "mcp_alpha_three"}) {
		t.Fatalf("selected = %#v, want first two sorted tools by tool name", got)
	}
	if len(selection.Omitted) != 1 || selection.Omitted[0].ExportedName != "mcp_alpha_two" || selection.Omitted[0].OmittedReason != OmittedReasonToolCountBudgetExceeded {
		t.Fatalf("omitted = %#v, want alpha two omitted by tool count", selection.Omitted)
	}
}

func TestAnalyzeIncludesEffectiveBudgetWhenProvided(t *testing.T) {
	budget := Budget{MaxTools: 3, EstimatedTokens: 123, MaxSchemaBytesPerTool: 456}
	report := Analyze([]Tool{{
		ServerName:      "alpha",
		ToolName:        "one",
		ExportedName:    "mcp_alpha_one",
		Registered:      true,
		Visible:         true,
		EstimatedTokens: 10,
	}}, Options{Budget: budget})

	if report.EffectiveBudget == nil || *report.EffectiveBudget != budget {
		t.Fatalf("EffectiveBudget = %#v, want %#v", report.EffectiveBudget, budget)
	}
}

func toolMetricNames(metrics []ToolMetric) []string {
	names := make([]string, 0, len(metrics))
	for _, metric := range metrics {
		names = append(names, metric.ExportedName)
	}
	return names
}

func toolNames(tools []Tool) []string {
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		names = append(names, tool.ExportedName)
	}
	return names
}
