package agent

import "testing"

func TestSyncCurrentSurfaceToolVisibility_PreservesRuntimeSpecificExclusions(t *testing.T) {
	runtime := newIsolatedRuntime()
	agent := NewAgentWithRuntime("gpt-5.4", &mockProvider{name: "openai"}, false, runtime)
	t.Cleanup(agent.Cleanup)

	agent.registry().SetExcludedTools(append(agent.registry().GetExcludedTools(), "read_file", "mcp_github_get_issue"))
	agent.syncCurrentSurfaceToolVisibility()

	excluded := agent.registry().GetExcludedTools()
	for _, name := range []string{"read_file", "mcp_github_get_issue"} {
		if !toolNameInList(excluded, name) {
			t.Fatalf("syncCurrentSurfaceToolVisibility() should preserve runtime-specific exclusion for %s, got %v", name, excluded)
		}
	}
	if toolNameInList(excluded, "apply_patch") {
		t.Fatalf("syncCurrentSurfaceToolVisibility() should keep apply_patch visible on the default surface, got %v", excluded)
	}
}
