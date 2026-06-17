package agent

import (
	"bytes"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestHandleProvidersCommand_UsesRuntimeOutput(t *testing.T) {
	var out bytes.Buffer
	agent := &Agent{
		ProviderName: "ollama",
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
	}

	result := handleProvidersCommand(agent)
	if !result {
		t.Fatal("handleProvidersCommand() = false, want true")
	}

	output := out.String()
	if !strings.Contains(output, "Provider credential status") {
		t.Fatalf("expected runtime output to contain providers header, got %q", output)
	}
	if !strings.Contains(output, "/provider [provider] [model]") {
		t.Fatalf("expected runtime output to contain usage hint, got %q", output)
	}
	if !strings.Contains(output, "openai") {
		t.Fatalf("expected runtime output to contain provider list, got %q", output)
	}
}

func TestProviderCredentialStatusDisplayIncludesSubscriptionStates(t *testing.T) {
	if got := providerCredentialStatusDisplay(ProviderCredentialLoggedIn); got != "(logged in)" {
		t.Fatalf("logged in display = %q, want logged in", got)
	}
	if got := providerCredentialStatusDisplay(ProviderCredentialLoginRequired); got != "(login required)" {
		t.Fatalf("login required display = %q, want login required", got)
	}
}

func TestHandleProvidersCommand_MarksOnlyClaudeOwnerAsCurrent(t *testing.T) {
	var out bytes.Buffer
	agent := &Agent{
		ProviderName:      "claude",
		ProviderConfigKey: "claude",
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
	}

	if result := handleProvidersCommand(agent); !result {
		t.Fatal("handleProvidersCommand() = false, want true")
	}

	output := out.String()
	if strings.Count(output, "✓ ") != 1 {
		t.Fatalf("expected exactly one current marker, got output %q", output)
	}
	if !strings.Contains(output, "✓ claude") {
		t.Fatalf("expected claude to be marked current, got %q", output)
	}
	if strings.Contains(output, "✓ anthropic") {
		t.Fatalf("anthropic should not be marked current when claude owns the session, got %q", output)
	}
}

func TestHandleProvidersCommand_MarksOnlyAnthropicOwnerAsCurrent(t *testing.T) {
	var out bytes.Buffer
	agent := &Agent{
		ProviderName:      "claude",
		ProviderConfigKey: "anthropic",
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
	}

	if result := handleProvidersCommand(agent); !result {
		t.Fatal("handleProvidersCommand() = false, want true")
	}

	output := out.String()
	if strings.Count(output, "✓ ") != 1 {
		t.Fatalf("expected exactly one current marker, got output %q", output)
	}
	if !strings.Contains(output, "✓ anthropic") {
		t.Fatalf("expected anthropic to be marked current, got %q", output)
	}
	if strings.Contains(output, "✓ claude") {
		t.Fatalf("claude should not be marked current when anthropic owns the session, got %q", output)
	}
}

func TestHandleProvidersCommand_ShowsAddedCurrentAliasStatus(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	var out bytes.Buffer
	agent := newConfigCommandTestAgent(newProjectMapDisabledConfig(), &out)
	agent.ProviderName = "claude"
	agent.ProviderConfigKey = "anthropic"

	if !handleProvidersCommand(agent) {
		t.Fatal("handleProvidersCommand() = false, want true")
	}
	output := out.String()
	if !strings.Contains(output, "anthropic") || !strings.Contains(output, "(credential configured)") {
		t.Fatalf("output = %q, want anthropic alias entry with configured status", output)
	}
}
