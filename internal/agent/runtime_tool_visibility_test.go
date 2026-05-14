package agent

import (
	"io"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestNewAgentWithRuntime_DefaultEditToolVisibility(t *testing.T) {
	runtime := newIsolatedRuntime()
	agent := NewAgentWithRuntime("gpt-5.4", &mockProvider{name: "openai"}, false, runtime)
	t.Cleanup(agent.Cleanup)

	defs := agent.registry().GetToolDefinitions()
	if !toolDefinitionNamed(defs, "gather_context") {
		t.Fatal("default mode should expose gather_context")
	}
	if !toolDefinitionNamed(defs, "apply_patch") {
		t.Fatal("default mode should expose apply_patch")
	}
	for _, name := range []string{"str_replace", "write_file", "delete_file"} {
		if toolDefinitionNamed(defs, name) {
			t.Fatalf("default mode should exclude %s", name)
		}
	}
	for _, name := range []string{"search_code", "list_dir"} {
		if toolDefinitionNamed(defs, name) {
			t.Fatalf("default mode should exclude %s", name)
		}
	}
	if !toolDefinitionNamed(defs, "read_file") {
		t.Fatal("default mode should expose read_file as exact-read override")
	}
}

func TestNewAgentWithRuntime_LegacyEditToolVisibility(t *testing.T) {
	t.Setenv("XELYON_EDIT_TOOL", "str_replace")

	runtime := newIsolatedRuntime()
	agent := newRuntimeTestAgent(t, runtime)

	defs := agent.registry().GetToolDefinitions()
	if toolDefinitionNamed(defs, "apply_patch") {
		t.Fatal("legacy mode should exclude apply_patch")
	}
	for _, name := range []string{"str_replace", "write_file", "delete_file"} {
		if !toolDefinitionNamed(defs, name) {
			t.Fatalf("legacy mode should expose %s", name)
		}
	}
	for _, name := range []string{"search_code", "read_file"} {
		if !toolDefinitionNamed(defs, name) {
			t.Fatalf("legacy mode should expose %s", name)
		}
	}
	if toolDefinitionNamed(defs, "list_dir") {
		t.Fatal("legacy mode should keep list_dir hidden")
	}
}

func TestChatOnce_DefaultEditToolVisibility(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	runtime := newIsolatedRuntime()
	runtime.UI = ui.NewRuntime(strings.NewReader(""), io.Discard, io.Discard)
	provider := &headlessToolSetProbeProvider{}
	agent := NewAgentWithRuntime("gpt-5.4", provider, false, runtime)
	t.Cleanup(agent.Cleanup)

	if err := agent.ChatOnce("probe"); err != nil {
		t.Fatalf("ChatOnce() error = %v", err)
	}
	if !toolNameInList(provider.toolNames, "gather_context") {
		t.Fatal("interactive normal mode should expose gather_context")
	}
	if !toolNameInList(provider.toolNames, "apply_patch") {
		t.Fatal("interactive normal mode should expose apply_patch")
	}
	for _, name := range []string{"str_replace", "write_file", "delete_file", "ask_user_question", "search_code", "list_dir"} {
		if toolNameInList(provider.toolNames, name) {
			t.Fatalf("interactive normal mode should exclude %s", name)
		}
	}
	if !toolNameInList(provider.toolNames, "read_file") {
		t.Fatal("interactive normal mode should expose read_file as exact-read override")
	}
}

func TestChatOnce_LegacyEditToolVisibility(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XELYON_EDIT_TOOL", "str_replace")

	runtime := newIsolatedRuntime()
	runtime.UI = ui.NewRuntime(strings.NewReader(""), io.Discard, io.Discard)
	provider := &headlessToolSetProbeProvider{}
	agent := NewAgentWithRuntime("gpt-5.4", provider, false, runtime)
	t.Cleanup(agent.Cleanup)

	if err := agent.ChatOnce("probe"); err != nil {
		t.Fatalf("ChatOnce() error = %v", err)
	}
	if toolNameInList(provider.toolNames, "apply_patch") {
		t.Fatal("interactive legacy mode should exclude apply_patch")
	}
	if toolNameInList(provider.toolNames, "ask_user_question") {
		t.Fatal("interactive legacy mode should exclude ask_user_question")
	}
	if !toolNameInList(provider.toolNames, "gather_context") {
		t.Fatal("interactive legacy mode should expose gather_context")
	}
	if toolNameInList(provider.toolNames, "list_dir") {
		t.Fatal("interactive legacy mode should exclude list_dir")
	}
	for _, name := range []string{"search_code", "read_file"} {
		if !toolNameInList(provider.toolNames, name) {
			t.Fatalf("interactive legacy mode should expose %s", name)
		}
	}
	for _, name := range []string{"str_replace", "write_file", "delete_file"} {
		if !toolNameInList(provider.toolNames, name) {
			t.Fatalf("interactive legacy mode should expose %s", name)
		}
	}
}
