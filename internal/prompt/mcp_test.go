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

	// サーバー名がセクションヘッダーに含まれること
	if !strings.Contains(result, "### github Server") {
		t.Error("result should contain github server header")
	}
	if !strings.Contains(result, "### slack Server") {
		t.Error("result should contain slack server header")
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

	// 説明が含まれること
	if !strings.Contains(result, "Create a GitHub issue") {
		t.Error("result should contain create_issue description")
	}
	if !strings.Contains(result, "Send a Slack message") {
		t.Error("result should contain send_message description")
	}

	// GitHub MCPツールがあるのでガイドが含まれること
	if !strings.Contains(result, "GitHub MCP Usage Guide") {
		t.Error("result should contain GitHub MCP guide when github tools are present")
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

func TestBuildMCPToolsPrompt_NoGitHub(t *testing.T) {
	tools := []MCPTool{
		{ServerName: "slack", Name: "send_message", Description: "Send a message"},
	}

	result := BuildMCPToolsPrompt(tools)

	if strings.Contains(result, "GitHub MCP Usage Guide") {
		t.Error("result should not contain GitHub guide when no github tools are present")
	}
}

func TestBuildGitHubMCPGuide(t *testing.T) {
	guide := BuildGitHubMCPGuide()

	if guide == "" {
		t.Fatal("expected non-empty guide")
	}

	keywords := []string{
		"GitHub MCP Usage Guide",
		"CONTEXT INFERENCE",
		"owner/repo",
		"Array arguments",
		"RULES",
		"MCP tools",
	}
	for _, kw := range keywords {
		if !strings.Contains(guide, kw) {
			t.Errorf("guide should contain keyword %q", kw)
		}
	}
	if strings.Contains(guide, "read_file or bash") {
		t.Error("guide should not recommend hidden local tools like read_file or bash for repo-local inspection")
	}
	if !strings.Contains(guide, "gather_context") {
		t.Error("guide should recommend gather_context-first local investigation guidance")
	}
}
