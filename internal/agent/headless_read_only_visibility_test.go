package agent

import (
	"context"
	"testing"
)

func TestHeadlessReadOnlyHidesWriteToolsFromProviderDefinitions(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XELYON_EDIT_TOOL", "str_replace")

	provider := &headlessToolSetProbeProvider{name: "openai"}
	result := RunHeadlessWithConfigOptions(context.Background(), "probe", "gpt-5.4", provider, newProjectMapDisabledConfig(), HeadlessRunOptions{
		ReadOnly: true,
	})
	if result.Status != HeadlessStatusSuccess {
		t.Fatalf("Status = %q, want success: %+v", result.Status, result.Error)
	}
	for _, name := range []string{"apply_patch", "write_file", "str_replace", "delete_file"} {
		if toolNameInList(provider.toolNames, name) {
			t.Fatalf("read-only headless mode should hide write tool %s from provider definitions: %v", name, provider.toolNames)
		}
	}
	for _, name := range []string{"bash", "run_skill_script"} {
		if toolNameInList(provider.toolNames, name) {
			t.Fatalf("read-only headless mode should hide shell execution tool %s from provider definitions: %v", name, provider.toolNames)
		}
	}
	for _, name := range []string{"spawn_agent", "wait_agent"} {
		if toolNameInList(provider.toolNames, name) {
			t.Fatalf("read-only headless mode should hide sub-agent tool %s from provider definitions: %v", name, provider.toolNames)
		}
	}
	for _, name := range []string{"gather_context", "read_file", "search_code"} {
		if !toolNameInList(provider.toolNames, name) {
			t.Fatalf("read-only headless mode should keep read/search tool %s visible: %v", name, provider.toolNames)
		}
	}
}
