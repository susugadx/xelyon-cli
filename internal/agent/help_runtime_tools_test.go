package agent

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintHelpToWriter_ShowsConnectedMCPToolsSeparately(t *testing.T) {
	runtime := newIsolatedRuntime()
	runtime.Registry.Register(&helpTestTool{
		name:        "mcp_github_get_issue",
		description: "Get issue details from GitHub. Requires the repository and issue number.",
	})
	runtime.Registry.Register(&helpTestTool{
		name:        "mcp_slack_post_message",
		description: "Post a Slack message to a channel.",
	})
	agent := NewAgentWithRuntime("gpt-5.4", &mockProvider{name: "openai"}, false, runtime)
	t.Cleanup(agent.Cleanup)

	var out bytes.Buffer
	printHelpToWriter(&out, agent)
	got := out.String()

	if !strings.Contains(got, "Connected MCP tools available in current runtime:") {
		t.Fatalf("help should include MCP section when MCP tools are registered\noutput:\n%s", got)
	}
	if !strings.Contains(got, "These depend on the current MCP connections and registry state.") {
		t.Fatalf("help should explain MCP availability source\noutput:\n%s", got)
	}
	if !strings.Contains(got, "mcp_github_get_issue - Get issue details from GitHub.") {
		t.Fatalf("help should include MCP tool summary\noutput:\n%s", got)
	}
	if !strings.Contains(got, "mcp_slack_post_message - Post a Slack message to a channel.") {
		t.Fatalf("help should include second MCP tool summary\noutput:\n%s", got)
	}
	if strings.Index(got, "mcp_github_get_issue") > strings.Index(got, "mcp_slack_post_message") {
		t.Fatalf("MCP tools should be sorted by name\noutput:\n%s", got)
	}
}

func TestPrintHelpToWriter_HonorsAdditionalRuntimeExclusions(t *testing.T) {
	runtime := newIsolatedRuntime()
	runtime.Registry.Register(&helpTestTool{
		name:        "mcp_github_get_issue",
		description: "Get issue details from GitHub.",
	})
	agent := NewAgentWithRuntime("gpt-5.4", &mockProvider{name: "openai"}, false, runtime)
	t.Cleanup(agent.Cleanup)

	agent.registry().SetExcludedTools(append(agent.registry().GetExcludedTools(), "read_file", "mcp_github_get_issue"))

	var out bytes.Buffer
	printHelpToWriter(&out, agent)
	got := out.String()

	if strings.Contains(got, "read_file         - ") {
		t.Fatalf("help should respect additional runtime exclusion for read_file\noutput:\n%s", got)
	}
	if strings.Contains(got, "mcp_github_get_issue - ") {
		t.Fatalf("help should respect additional runtime exclusion for MCP tool\noutput:\n%s", got)
	}
}

func TestPrintHelpToWriter_NormalizesMultilineToolDescriptions(t *testing.T) {
	runtime := newIsolatedRuntime()
	runtime.Registry.Register(&helpTestTool{
		name:        "mcp_github_get_issue",
		description: "Get issue details\nfrom GitHub. Requires the repository and issue number.",
	})
	agent := NewAgentWithRuntime("gpt-5.4", &mockProvider{name: "openai"}, false, runtime)
	t.Cleanup(agent.Cleanup)

	var out bytes.Buffer
	printHelpToWriter(&out, agent)
	got := out.String()

	if !strings.Contains(got, "mcp_github_get_issue - Get issue details from GitHub.") {
		t.Fatalf("help should normalize multiline descriptions into one line\noutput:\n%s", got)
	}
	if strings.Contains(got, "mcp_github_get_issue - Get issue details\nfrom GitHub.") {
		t.Fatalf("help should not emit multiline descriptions in the tool table\noutput:\n%s", got)
	}
}

func TestMergeSurfaceManagedExcludedTools_PreservesRuntimeSpecificExclusionsAcrossPhaseReset(t *testing.T) {
	planExcluded := newToolVisibilityPolicy(EditToolModeApplyPatch, toolSurfacePhasePlan, toolVisibilityOptions{allowSubAgents: true}).excluded()
	policy := newToolVisibilityPolicy(EditToolModeApplyPatch, toolSurfacePhaseNormal, toolVisibilityOptions{allowSubAgents: true})

	got := mergeSurfaceManagedExcludedTools(append(planExcluded, "read_file", "mcp_github_get_issue"), policy)

	for _, name := range []string{"read_file", "mcp_github_get_issue", "ask_user_question"} {
		if !toolNameInList(got, name) {
			t.Fatalf("mergeSurfaceManagedExcludedTools() should keep %s excluded, got %v", name, got)
		}
	}
	if toolNameInList(got, "apply_patch") {
		t.Fatalf("mergeSurfaceManagedExcludedTools() should not hide apply_patch on the normal apply_patch surface, got %v", got)
	}
}
