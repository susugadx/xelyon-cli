package agent

import (
	"bytes"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestHandleThinkingCommand_DeepSeekKeepsModelMirrorsAndSurfaceStable(t *testing.T) {
	runtime := newIsolatedRuntime()
	agent := NewAgentWithRuntime("deepseek-v4-pro", &mockProvider{name: "deepseek"}, false, runtime)
	t.Cleanup(agent.Cleanup)

	if !handleThinkingCommand(agent, []string{"on"}) {
		t.Fatal("handleThinkingCommand(on) = false, want true")
	}
	if !agent.cfg().Thinking.Enabled {
		t.Fatal("Thinking.Enabled = false, want true after /thinking on")
	}
	if agent.CurrentModel != "deepseek-v4-pro" {
		t.Fatalf("CurrentModel = %q, want %q", agent.CurrentModel, "deepseek-v4-pro")
	}
	if agent.session == nil || agent.session.Model != "deepseek-v4-pro" {
		t.Fatalf("session.Model = %q, want %q", agent.session.Model, "deepseek-v4-pro")
	}
	if agent.Stats == nil || agent.Stats.Model != "deepseek-v4-pro" {
		t.Fatalf("Stats.Model = %q, want %q", agent.Stats.Model, "deepseek-v4-pro")
	}

	excluded := agent.registry().GetExcludedTools()
	if !toolNameInList(excluded, "apply_patch") {
		t.Fatalf("deepseek thinking mode should keep legacy exclusions, got %v", excluded)
	}
	if toolNameInList(excluded, "search_code") {
		t.Fatalf("deepseek thinking mode should expose search_code on legacy surfaces, got %v", excluded)
	}

	if !handleThinkingCommand(agent, []string{"off"}) {
		t.Fatal("handleThinkingCommand(off) = false, want true")
	}
	if agent.cfg().Thinking.Enabled {
		t.Fatal("Thinking.Enabled = true, want false after /thinking off")
	}
	if agent.CurrentModel != "deepseek-v4-pro" {
		t.Fatalf("CurrentModel = %q, want %q", agent.CurrentModel, "deepseek-v4-pro")
	}
	if agent.session == nil || agent.session.Model != "deepseek-v4-pro" {
		t.Fatalf("session.Model = %q, want %q", agent.session.Model, "deepseek-v4-pro")
	}
	if agent.Stats == nil || agent.Stats.Model != "deepseek-v4-pro" {
		t.Fatalf("Stats.Model = %q, want %q", agent.Stats.Model, "deepseek-v4-pro")
	}

	excluded = agent.registry().GetExcludedTools()
	if !toolNameInList(excluded, "apply_patch") {
		t.Fatalf("deepseek non-thinking mode should stay on legacy exclusions, got %v", excluded)
	}
	if toolNameInList(excluded, "search_code") {
		t.Fatalf("deepseek non-thinking mode should keep search_code visible on legacy surfaces, got %v", excluded)
	}
}

func TestHandleThinkingCommand_DeepSeekOffPreservesNonReasonerModel(t *testing.T) {
	runtime := newIsolatedRuntime()
	agent := NewAgentWithRuntime("deepseek-r1:8b", &mockProvider{name: "deepseek"}, false, runtime)
	t.Cleanup(agent.Cleanup)
	agent.cfg().Thinking.Enabled = false

	if !handleThinkingCommand(agent, []string{"off"}) {
		t.Fatal("handleThinkingCommand(off) = false, want true")
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

func TestHandleSpecialCommandThinkingNameAndAlias(t *testing.T) {
	runtime := newIsolatedRuntime()
	agent := NewAgentWithRuntime("gpt-test", &mockProvider{name: "openai"}, false, runtime)
	t.Cleanup(agent.Cleanup)

	if !handleSpecialCommandForSurface("/thinking high", agent, commandcatalog.CommandSurfaceTUI) {
		t.Fatal("handleSpecialCommandForSurface(/thinking high) = false, want true")
	}
	if !agent.cfg().Thinking.Enabled || agent.cfg().Thinking.Level != "high" {
		t.Fatalf("Thinking = %+v, want enabled high", agent.cfg().Thinking)
	}

	if !handleSpecialCommandForSurface("/think off", agent, commandcatalog.CommandSurfaceTUI) {
		t.Fatal("handleSpecialCommandForSurface(/think off) = false, want true")
	}
	if agent.cfg().Thinking.Enabled {
		t.Fatal("Thinking.Enabled = true, want false after /think alias off")
	}
}

func TestHandleThinkingCommand_UsesCatalogModelForCodexMinimum(t *testing.T) {
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

	if !handleThinkingCommand(agent, []string{"off"}) {
		t.Fatal("handleThinkingCommand(off) = false, want true")
	}
	if agent.cfg().Thinking.Enabled {
		t.Fatal("Thinking.Enabled = true, want false after /thinking off")
	}
	if agent.cfg().Thinking.Level != "low" {
		t.Fatalf("Thinking.Level = %q, want low for Codex catalog model", agent.cfg().Thinking.Level)
	}
	if output := out.String(); !strings.Contains(output, "low (Codex minimum)") || strings.Contains(output, "Thinking Mode: OFF") {
		t.Fatalf("output should show Codex minimum, got:\n%s", output)
	}

	out.Reset()
	if !handleThinkingCommand(agent, nil) {
		t.Fatal("handleThinkingCommand(status) = false, want true")
	}
	if output := out.String(); !strings.Contains(output, "low (Codex minimum)") {
		t.Fatalf("status output should show Codex minimum, got:\n%s", output)
	}
}
