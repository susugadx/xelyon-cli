package agent

import (
	"bytes"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestHandleThinkCommand_DeepSeekKeepsModelMirrorsAndSurfaceInSync(t *testing.T) {
	runtime := newIsolatedRuntime()
	agent := NewAgentWithRuntime("deepseek-chat", &mockProvider{name: "deepseek"}, false, runtime)
	t.Cleanup(agent.Cleanup)

	agent.registry().SetExcludedTools(newToolVisibilityPolicy(EditToolModeApplyPatch, toolSurfacePhaseNormal, toolVisibilityOptions{allowSubAgents: true}).excluded())

	if !handleThinkCommand(agent, []string{"on"}) {
		t.Fatal("handleThinkCommand(on) = false, want true")
	}
	if !agent.cfg().Thinking.Enabled {
		t.Fatal("Thinking.Enabled = false, want true after /think on")
	}
	if agent.CurrentModel != "deepseek-reasoner" {
		t.Fatalf("CurrentModel = %q, want %q", agent.CurrentModel, "deepseek-reasoner")
	}
	if agent.session == nil || agent.session.Model != "deepseek-reasoner" {
		t.Fatalf("session.Model = %q, want %q", agent.session.Model, "deepseek-reasoner")
	}
	if agent.Stats == nil || agent.Stats.Model != "deepseek-reasoner" {
		t.Fatalf("Stats.Model = %q, want %q", agent.Stats.Model, "deepseek-reasoner")
	}

	excluded := agent.registry().GetExcludedTools()
	if !toolNameInList(excluded, "apply_patch") {
		t.Fatalf("deepseek thinking mode should restore legacy exclusions, got %v", excluded)
	}
	if toolNameInList(excluded, "search_code") {
		t.Fatalf("deepseek thinking mode should expose search_code on legacy surfaces, got %v", excluded)
	}

	if !handleThinkCommand(agent, []string{"off"}) {
		t.Fatal("handleThinkCommand(off) = false, want true")
	}
	if agent.cfg().Thinking.Enabled {
		t.Fatal("Thinking.Enabled = true, want false after /think off")
	}
	if agent.CurrentModel != "deepseek-chat" {
		t.Fatalf("CurrentModel = %q, want %q", agent.CurrentModel, "deepseek-chat")
	}
	if agent.session == nil || agent.session.Model != "deepseek-chat" {
		t.Fatalf("session.Model = %q, want %q", agent.session.Model, "deepseek-chat")
	}
	if agent.Stats == nil || agent.Stats.Model != "deepseek-chat" {
		t.Fatalf("Stats.Model = %q, want %q", agent.Stats.Model, "deepseek-chat")
	}

	excluded = agent.registry().GetExcludedTools()
	if !toolNameInList(excluded, "apply_patch") {
		t.Fatalf("deepseek non-thinking mode should stay on legacy exclusions, got %v", excluded)
	}
	if toolNameInList(excluded, "search_code") {
		t.Fatalf("deepseek non-thinking mode should keep search_code visible on legacy surfaces, got %v", excluded)
	}
}

func TestHandleThinkCommand_DeepSeekOffPreservesNonReasonerModel(t *testing.T) {
	runtime := newIsolatedRuntime()
	agent := NewAgentWithRuntime("deepseek-r1:8b", &mockProvider{name: "deepseek"}, false, runtime)
	t.Cleanup(agent.Cleanup)
	agent.cfg().Thinking.Enabled = false

	if !handleThinkCommand(agent, []string{"off"}) {
		t.Fatal("handleThinkCommand(off) = false, want true")
	}

	if agent.CurrentModel != "deepseek-r1:8b" {
		t.Fatalf("CurrentModel = %q, want %q", agent.CurrentModel, "deepseek-r1:8b")
	}
	if agent.session == nil || agent.session.Model != "deepseek-r1:8b" {
		t.Fatalf("session.Model = %q, want %q", agent.session.Model, "deepseek-r1:8b")
	}
	if agent.Stats == nil || agent.Stats.Model != "deepseek-r1:8b" {
		t.Fatalf("Stats.Model = %q, want %q", agent.Stats.Model, "deepseek-r1:8b")
	}
}

func TestHandleThinkCommand_UsesCatalogModelForCodexMinimum(t *testing.T) {
	var out bytes.Buffer
	cfg := newProjectMapDisabledConfig()
	cfg.SetProviderModelConfig("openai", config.ProviderModelConfig{
		DefaultModel: "corp-codex-deployment",
		CatalogModel: "gpt-5.2-codex",
	})
	cfg.Thinking.Enabled = true
	cfg.Thinking.Level = "high"

	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.UI = ui.NewRuntime(strings.NewReader(""), &out, &out)
	agent := NewAgentWithRuntime("corp-codex-deployment", &mockProvider{name: "openai"}, false, runtime)
	t.Cleanup(agent.Cleanup)

	if !handleThinkCommand(agent, []string{"off"}) {
		t.Fatal("handleThinkCommand(off) = false, want true")
	}
	if agent.cfg().Thinking.Enabled {
		t.Fatal("Thinking.Enabled = true, want false after /think off")
	}
	if agent.cfg().Thinking.Level != "low" {
		t.Fatalf("Thinking.Level = %q, want low for Codex catalog model", agent.cfg().Thinking.Level)
	}
	if output := out.String(); !strings.Contains(output, "low (Codex minimum)") || strings.Contains(output, "Thinking Mode: OFF") {
		t.Fatalf("output should show Codex minimum, got:\n%s", output)
	}

	out.Reset()
	if !handleThinkCommand(agent, nil) {
		t.Fatal("handleThinkCommand(status) = false, want true")
	}
	if output := out.String(); !strings.Contains(output, "low (Codex minimum)") {
		t.Fatalf("status output should show Codex minimum, got:\n%s", output)
	}
}
