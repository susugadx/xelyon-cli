package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/mcp"
)

type mockMCPProvider struct {
	name             string
	setMCPToolsCalls int
	lastTools        []api.ToolDefinition
}

var _ api.Provider = (*mockMCPProvider)(nil)
var _ api.MCPProvider = (*mockMCPProvider)(nil)

func (p *mockMCPProvider) Name() string { return p.name }

func (p *mockMCPProvider) ChatWithTools(_ context.Context, _ string, _ []api.Message, _ string) (string, error) {
	return "", nil
}

func (p *mockMCPProvider) SupportsImages() bool { return false }

func (p *mockMCPProvider) ChatWithImage(_ context.Context, _ string, _ []api.Message, _ string, _ *api.ImageData, _ string) (string, error) {
	return "", nil
}

func (p *mockMCPProvider) IsFunctionCallingEnabled() bool { return true }

func (p *mockMCPProvider) SetMCPEnabled(_ bool) {}

func (p *mockMCPProvider) SetMCPTools(tools []api.ToolDefinition) {
	p.setMCPToolsCalls++
	p.lastTools = tools
}

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

// --- detectGitHubIntent additional tests ---

func TestDetectGitHubIntent_Keywords(t *testing.T) {
	positives := []struct {
		name  string
		input string
	}{
		{name: "issue", input: "create an issue for this"},
		{name: "PR", input: "open a PR for this change"},
		{name: "pull request", input: "review the pull request"},
		{name: "プルリクエスト", input: "プルリクエストを作成して"},
		{name: "プルリク", input: "プルリクを確認して"},
		{name: "イシュー", input: "イシューを作って"},
		{name: "actions", input: "check GitHub Actions"},
		{name: "workflow", input: "the workflow is failing"},
		{name: "ワークフロー", input: "ワークフローを確認"},
		{name: "CI", input: "CI is broken"},
		{name: "repo", input: "list my repos"},
		{name: "repository", input: "clone the repository"},
		{name: "リポジトリ", input: "リポジトリを確認"},
		{name: "github", input: "check github"},
		{name: "ギットハブ", input: "ギットハブを見て"},
		{name: "gh", input: "use gh to list issues"},
		{name: "issues plural", input: "show all issues"},
	}

	for _, tt := range positives {
		t.Run(tt.name, func(t *testing.T) {
			if !detectGitHubIntent(tt.input) {
				t.Errorf("detectGitHubIntent(%q) = false, want true", tt.input)
			}
		})
	}
}

func TestDetectGitHubIntent_NegativeCases(t *testing.T) {
	negatives := []struct {
		name  string
		input string
	}{
		{name: "code fix", input: "fix the bug in main.go"},
		{name: "code review", input: "review my code changes"},
		{name: "read file", input: "read the config file"},
		{name: "empty", input: ""},
		{name: "generic", input: "hello, how are you?"},
		{name: "build", input: "build the binary"},
		{name: "test", input: "run all tests"},
	}

	for _, tt := range negatives {
		t.Run(tt.name, func(t *testing.T) {
			if detectGitHubIntent(tt.input) {
				t.Errorf("detectGitHubIntent(%q) = true, want false", tt.input)
			}
		})
	}
}

func TestDetectGitHubIntent_CaseInsensitive(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "GITHUB uppercase", input: "GITHUB actions status", want: true},
		{name: "Issue mixed case", input: "Get Issue #123", want: true},
		{name: "Pr uppercase", input: "Create a PR", want: true},
		{name: "ci lowercase", input: "ci is failing", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detectGitHubIntent(tt.input); got != tt.want {
				t.Errorf("detectGitHubIntent(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
