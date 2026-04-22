package ui

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestResolveProviderAddTarget(t *testing.T) {
	existing := map[string]config.ProviderModelConfig{
		"openai": {DefaultModel: "gpt-5"},
	}

	name, status := resolveProviderAddTarget("", existing)
	if name != "" || status != providerAddTargetEmpty {
		t.Fatalf("resolveProviderAddTarget(empty) = (%q,%v), want (\"\",%v)", name, status, providerAddTargetEmpty)
	}

	name, status = resolveProviderAddTarget("OpenAI", existing)
	if name != "openai" || status != providerAddTargetDuplicate {
		t.Fatalf("resolveProviderAddTarget(duplicate) = (%q,%v), want (\"openai\",%v)", name, status, providerAddTargetDuplicate)
	}

	name, status = resolveProviderAddTarget("Claude", existing)
	if name != "claude" || status != providerAddTargetReady {
		t.Fatalf("resolveProviderAddTarget(ready) = (%q,%v), want (\"claude\",%v)", name, status, providerAddTargetReady)
	}
}

func TestWithAddedProviderModel(t *testing.T) {
	existing := map[string]config.ProviderModelConfig{
		"openai": {DefaultModel: "gpt-5"},
	}

	updated := withAddedProviderModel(existing, "claude", "  claude-sonnet-4-6 ")
	if len(updated) != 2 {
		t.Fatalf("len(updated) = %d, want 2", len(updated))
	}
	if updated["claude"].DefaultModel != "claude-sonnet-4-6" {
		t.Fatalf("updated[claude] = %#v, want trimmed default model", updated["claude"])
	}
	if _, ok := existing["claude"]; ok {
		t.Fatalf("existing map was mutated: %#v", existing)
	}
}

func TestSelectProviderByInput(t *testing.T) {
	providers := []string{"claude", "openai"}

	name, ok := selectProviderByInput("2", providers)
	if !ok || name != "openai" {
		t.Fatalf("selectProviderByInput(valid) = (%q,%v), want (\"openai\",true)", name, ok)
	}

	name, ok = selectProviderByInput("9", providers)
	if ok || name != "" {
		t.Fatalf("selectProviderByInput(invalid) = (%q,%v), want (\"\",false)", name, ok)
	}
}

func TestNormalizeLSPServerLanguage(t *testing.T) {
	if got := normalizeLSPServerLanguage("  python  "); got != "python" {
		t.Fatalf("normalizeLSPServerLanguage() = %q, want %q", got, "python")
	}
}

func TestWithAddedLSPServer(t *testing.T) {
	existing := map[string]config.LSPServerConfig{
		"go": {Command: "gopls"},
	}

	updated := withAddedLSPServer(existing, "python", "  pyright-langserver  ")
	if len(updated) != 2 {
		t.Fatalf("len(updated) = %d, want 2", len(updated))
	}
	if updated["python"].Command != "pyright-langserver" {
		t.Fatalf("updated[python] = %#v, want trimmed command", updated["python"])
	}
	if _, ok := existing["python"]; ok {
		t.Fatalf("existing map was mutated: %#v", existing)
	}
}

func TestSelectLSPServerByInput(t *testing.T) {
	servers := []string{"go", "python"}

	lang, ok := selectLSPServerByInput("1", servers)
	if !ok || lang != "go" {
		t.Fatalf("selectLSPServerByInput(valid) = (%q,%v), want (\"go\",true)", lang, ok)
	}

	lang, ok = selectLSPServerByInput("0", servers)
	if ok || lang != "" {
		t.Fatalf("selectLSPServerByInput(invalid) = (%q,%v), want (\"\",false)", lang, ok)
	}
}

func TestUpdateLSPServerCommandValue(t *testing.T) {
	current := config.LSPServerConfig{Command: "old"}

	updated, ok := updateLSPServerCommandValue(current, "   ")
	if ok || updated.Command != "old" {
		t.Fatalf("updateLSPServerCommandValue(empty) = (%#v,%v), want unchanged,false", updated, ok)
	}

	updated, ok = updateLSPServerCommandValue(current, "  new-command ")
	if !ok || updated.Command != "new-command" {
		t.Fatalf("updateLSPServerCommandValue(valid) = (%#v,%v), want updated,true", updated, ok)
	}
}

func TestToggleLSPServerDisabledValue(t *testing.T) {
	current := config.LSPServerConfig{Disabled: false}
	updated := toggleLSPServerDisabledValue(current)
	if !updated.Disabled {
		t.Fatalf("toggleLSPServerDisabledValue(false) = %#v, want disabled=true", updated)
	}
}
