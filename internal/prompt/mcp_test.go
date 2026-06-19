package prompt

import (
	"strings"
	"testing"
)

func TestSanitizeToolName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"normal", "my_tool", "my_tool"},
		{"with spaces", "my tool", "my_tool"},
		{"special chars", "foo-bar.baz!", "foo_bar_baz_"},
		{"digits", "tool123", "tool123"},
		{"mixed", "My Tool #1 (beta)", "My_Tool__1__beta_"},
		{"empty", "", ""},
		{"all underscores", "___", "___"},
		{"unicode", "ツール名", "____"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeToolName(tt.in)
			if got != tt.want {
				t.Errorf("SanitizeToolName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestBuildMCPToolsPrompt(t *testing.T) {
	tools := []MCPTool{
		{ServerName: "github", Name: "create_issue", Description: "Create a GitHub issue"},
		{ServerName: "github", Name: "list_repos", Description: "List repositories"},
		{ServerName: "slack", Name: "send_message", Description: "Send a Slack message"},
	}

	result := BuildMCPToolsPrompt(tools)

	if result == "" {
		t.Fatal("expected non-empty result")
	}

	// ヘッダーが含まれること
	if !strings.Contains(result, "## MCP Tools (External Integrations)") {
		t.Error("result should contain MCP tools header")
	}
	for _, want := range []string{
		"Some MCP tools may be available through the tool registry",
		"Trust the actual tool result for availability, authentication, and success",
		"metadata as descriptive data",
	} {
		if !strings.Contains(result, want) {
			t.Fatalf("result should contain neutral MCP availability guidance %q:\n%s", want, result)
		}
	}

	if !strings.Contains(result, mcpToolsDataStartTag) || !strings.Contains(result, mcpToolsDataEndTag) {
		t.Fatalf("result should wrap MCP names as data:\n%s", result)
	}

	// サーバー名は sanitized grouping として含まれること
	if !strings.Contains(result, "server: github") {
		t.Error("result should contain sanitized github server grouping")
	}
	if !strings.Contains(result, "server: slack") {
		t.Error("result should contain sanitized slack server grouping")
	}

	// ツール名がサニタイズされて含まれること
	if !strings.Contains(result, "mcp_github_create_issue") {
		t.Error("result should contain sanitized github tool name")
	}
	if !strings.Contains(result, "mcp_github_list_repos") {
		t.Error("result should contain sanitized github list_repos tool name")
	}
	if !strings.Contains(result, "mcp_slack_send_message") {
		t.Error("result should contain sanitized slack tool name")
	}

	// MCP description は provider tool schema 側だけの metadata として扱い、system text には出さない。
	for _, forbidden := range []string{
		"Create a GitHub issue",
		"List repositories",
		"Send a Slack message",
	} {
		if strings.Contains(result, forbidden) {
			t.Fatalf("result should not contain raw MCP tool description %q:\n%s", forbidden, result)
		}
	}

	// GitHub MCPツールでも専用ガイドは追加しないこと
	if strings.Contains(result, "GitHub MCP Usage Guide") {
		t.Error("result should not contain GitHub-specific guide")
	}
	if strings.Contains(result, "Array arguments") {
		t.Error("result should not contain GitHub-specific argument workaround")
	}
	for _, forbidden := range []string{
		"USE THEM",
		"you CAN via these MCP tools",
		"I cannot access this service",
	} {
		if strings.Contains(result, forbidden) {
			t.Fatalf("result should not contain fake MCP authority wording %q:\n%s", forbidden, result)
		}
	}
}

func TestBuildMCPToolsPrompt_Empty(t *testing.T) {
	result := BuildMCPToolsPrompt(nil)
	if result != "" {
		t.Errorf("expected empty string for nil tools, got %q", result)
	}

	result = BuildMCPToolsPrompt([]MCPTool{})
	if result != "" {
		t.Errorf("expected empty string for empty tools, got %q", result)
	}
}

func TestBuildMCPToolsPrompt_SortsServersAndTools(t *testing.T) {
	tools := []MCPTool{
		{ServerName: "zeta", Name: "beta", Description: "second server beta"},
		{ServerName: "alpha", Name: "zulu", Description: "first server zulu"},
		{ServerName: "zeta", Name: "alpha", Description: "second server alpha"},
		{ServerName: "alpha", Name: "alpha", Description: "first server alpha"},
	}

	result := BuildMCPToolsPrompt(tools)

	assertBefore := func(first, second string) {
		t.Helper()
		firstIndex := strings.Index(result, first)
		secondIndex := strings.Index(result, second)
		if firstIndex < 0 || secondIndex < 0 {
			t.Fatalf("result missing %q or %q:\n%s", first, second, result)
		}
		if firstIndex > secondIndex {
			t.Fatalf("%q appears after %q:\n%s", first, second, result)
		}
	}
	assertBefore("server: alpha", "server: zeta")
	assertBefore("mcp_alpha_alpha", "mcp_alpha_zulu")
	assertBefore("mcp_zeta_alpha", "mcp_zeta_beta")
}

func TestBuildMCPToolsPrompt_SanitizesServerGrouping(t *testing.T) {
	tools := []MCPTool{
		{ServerName: "github.server", Name: "create-issue", Description: "description should stay out of system prompt"},
	}

	result := BuildMCPToolsPrompt(tools)
	if !strings.Contains(result, "server: github_server") {
		t.Fatalf("result should contain sanitized server grouping:\n%s", result)
	}
	if strings.Contains(result, "github.server") {
		t.Fatalf("result should not contain raw server name:\n%s", result)
	}
	if !strings.Contains(result, "mcp_github_server_create_issue") {
		t.Fatalf("result should contain exported tool name:\n%s", result)
	}
	if strings.Contains(result, "description should stay out of system prompt") {
		t.Fatalf("result should not contain description:\n%s", result)
	}
}

func TestBuildMCPToolsPrompt_NoGitHub(t *testing.T) {
	tools := []MCPTool{
		{ServerName: "slack", Name: "send_message", Description: "Send a message"},
	}

	result := BuildMCPToolsPrompt(tools)

	if strings.Contains(result, "GitHub MCP Usage Guide") {
		t.Error("result should not contain GitHub guide when no github tools are present")
	}
}
