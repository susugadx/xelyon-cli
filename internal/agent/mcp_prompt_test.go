package agent

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/prompt"
)

func TestDetectGitHubIntent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "issue keyword",
			input:    "Create an issue for this bug",
			expected: true,
		},
		{
			name:     "イシュー keyword",
			input:    "このバグのイシューを作成して",
			expected: true,
		},
		{
			name:     "PR keyword",
			input:    "List all open PRs",
			expected: true,
		},
		{
			name:     "プルリクエスト keyword",
			input:    "プルリクエストを確認して",
			expected: true,
		},
		{
			name:     "GitHub keyword",
			input:    "Check GitHub actions",
			expected: true,
		},
		{
			name:     "CI keyword",
			input:    "CIの状態を確認",
			expected: true,
		},
		{
			name:     "actions keyword",
			input:    "GitHub Actionsが失敗している",
			expected: true,
		},
		{
			name:     "workflow keyword",
			input:    "workflowを確認して",
			expected: true,
		},
		{
			name:     "repo keyword",
			input:    "List my repos",
			expected: true,
		},
		{
			name:     "no GitHub keywords",
			input:    "Fix the bug in main.go",
			expected: false,
		},
		{
			name:     "code review request",
			input:    "Review my code changes",
			expected: false,
		},
		{
			name:     "case insensitive - GITHUB",
			input:    "GITHUB actions status",
			expected: true,
		},
		{
			name:     "case insensitive - Issue",
			input:    "Get Issue #123",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectGitHubIntent(tt.input)
			if got != tt.expected {
				t.Errorf("detectGitHubIntent(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSanitizeToolName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "github",
			expected: "github",
		},
		{
			name:     "with hyphen",
			input:    "my-server",
			expected: "my_server",
		},
		{
			name:     "with special chars",
			input:    "test@server#1",
			expected: "test_server_1",
		},
		{
			name:     "with spaces",
			input:    "my server",
			expected: "my_server",
		},
		{
			name:     "numbers and underscores",
			input:    "server_123",
			expected: "server_123",
		},
		{
			name:     "mixed case",
			input:    "MyServer",
			expected: "MyServer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeToolName(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeToolName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestBuildGitHubMCPGuide(t *testing.T) {
	guide := prompt.BuildGitHubMCPGuide()

	// ガイドが空でないことを確認
	if guide == "" {
		t.Error("BuildGitHubMCPGuide() returned empty string")
	}

	// 必要なキーワードが含まれていることを確認
	keywords := []string{
		"GitHub MCP Usage Guide",
		"mcp_github_create_issue",
		// 新しい改善項目
		"CONTEXT INFERENCE",
		"NEVER ask",
		"CRITICAL: Array arguments",
		"RULES:",
	}

	for _, kw := range keywords {
		if !strings.Contains(guide, kw) {
			t.Errorf("BuildGitHubMCPGuide() missing keyword: %q", kw)
		}
	}
	if strings.Contains(guide, "read_file or bash") {
		t.Error("BuildGitHubMCPGuide() should not recommend read_file or bash for local repo inspection")
	}
	if !strings.Contains(guide, "gather_context") {
		t.Error("BuildGitHubMCPGuide() should recommend gather_context-first local investigation guidance")
	}
}
