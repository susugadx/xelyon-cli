package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/providerpicker"
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
)

func TestProviderPicker_CommandOpensProviderThenModelAndSwitches(t *testing.T) {
	agent := &stubAgent{
		statusLine: "ready",
		providerCandidates: []providerpicker.ProviderCandidate{
			{Key: "openai", Label: "openai", Current: true, CredentialStatus: providerpicker.ProviderCredentialConfigured},
		},
		modelCandidates: map[string][]providerpicker.ModelCandidate{
			"openai": {
				{Name: "gpt-current", Current: true},
				{Name: "gpt-next"},
			},
		},
	}
	m := newSizedPromptTestModel(agent, 80, 24)

	m = submitProviderPickerCommand(t, m, "/provider")
	if m.providerPicker == nil || m.providerPicker.mode != providerPickerProviders {
		t.Fatalf("/provider should open provider picker, got %#v", m.providerPicker)
	}

	m = updateProviderPickerKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.providerPicker == nil || m.providerPicker.mode != providerPickerModels {
		t.Fatalf("Enter on provider should open model picker, got %#v", m.providerPicker)
	}

	m = updateProviderPickerKey(t, m, tea.KeyMsg{Type: tea.KeyDown})
	m = updateProviderPickerKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.providerPicker != nil {
		t.Fatal("model selection should close picker")
	}
	want := providerModelSwitchCall{Provider: "openai", Model: "gpt-next"}
	if got := agent.switchedProviders; len(got) != 1 || got[0] != want {
		t.Fatalf("switchedProviders = %#v, want %#v", got, want)
	}
}

func TestProviderPicker_ModelCommandSwitchesCurrentProviderModel(t *testing.T) {
	agent := &stubAgent{
		statusLine:        "ready",
		providerName:      "openai",
		providerConfigKey: "openai",
		modelCandidates: map[string][]providerpicker.ModelCandidate{
			"openai": {
				{Name: "gpt-current", Current: true},
				{Name: "gpt-next"},
			},
		},
	}
	m := newSizedPromptTestModel(agent, 80, 24)

	m = submitProviderPickerCommand(t, m, "/model")
	if m.providerPicker == nil || m.providerPicker.mode != providerPickerModels {
		t.Fatalf("/model should open model picker, got %#v", m.providerPicker)
	}

	m = updateProviderPickerKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updateProviderPickerKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if got := agent.switchedModels; len(got) != 1 || got[0] != "gpt-next" {
		t.Fatalf("switchedModels = %#v, want gpt-next", got)
	}
	if len(agent.switchedProviders) != 0 {
		t.Fatalf("/model should not switch provider, got %#v", agent.switchedProviders)
	}
}

func TestProviderPicker_EscBackspaceAndFilter(t *testing.T) {
	agent := &stubAgent{
		statusLine: "ready",
		providerCandidates: []providerpicker.ProviderCandidate{
			{Key: "openai", Label: "openai", CredentialStatus: providerpicker.ProviderCredentialConfigured},
			{Key: "gemini", Label: "gemini", CredentialStatus: providerpicker.ProviderCredentialMissingKey},
		},
		modelCandidates: map[string][]providerpicker.ModelCandidate{
			"gemini": {{Name: "gemini-3.1-pro"}},
		},
	}

	t.Run("esc closes", func(t *testing.T) {
		m := submitProviderPickerCommand(t, newSizedPromptTestModel(agent, 80, 24), "/provider")
		m = updateProviderPickerKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
		if m.providerPicker != nil {
			t.Fatal("Esc should close provider picker")
		}
	})

	t.Run("filter narrows provider rows", func(t *testing.T) {
		m := submitProviderPickerCommand(t, newSizedPromptTestModel(agent, 80, 24), "/provider")
		m = updateProviderPickerKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
		m = updateProviderPickerKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("gem")})
		rows := m.providerPicker.providerRows()
		if len(rows) != 1 || rows[0].Key != "gemini" {
			t.Fatalf("filtered provider rows = %#v, want gemini", rows)
		}
		m = updateProviderPickerKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
		if m.providerPicker == nil || m.providerPicker.provider != "gemini" {
			t.Fatalf("filtered Enter should choose gemini, got %#v", m.providerPicker)
		}
	})

	t.Run("backspace returns from model list to provider list", func(t *testing.T) {
		m := submitProviderPickerCommand(t, newSizedPromptTestModel(agent, 80, 24), "/provider")
		m = updateProviderPickerKey(t, m, tea.KeyMsg{Type: tea.KeyDown})
		m = updateProviderPickerKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
		if m.providerPicker.mode != providerPickerModels {
			t.Fatalf("mode = %v, want models", m.providerPicker.mode)
		}
		m = updateProviderPickerKey(t, m, tea.KeyMsg{Type: tea.KeyBackspace})
		if m.providerPicker == nil || m.providerPicker.mode != providerPickerProviders {
			t.Fatalf("Backspace should return to providers, got %#v", m.providerPicker)
		}
	})
}

func TestProviderPicker_CustomInput(t *testing.T) {
	agent := &stubAgent{
		statusLine:        "ready",
		providerName:      "openai",
		providerConfigKey: "openai",
		modelCandidates: map[string][]providerpicker.ModelCandidate{
			"openai": {{Name: "Custom model...", Custom: true}},
		},
	}
	m := submitProviderPickerCommand(t, newSizedPromptTestModel(agent, 80, 24), "/model")

	m = updateProviderPickerKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.providerPicker == nil || m.providerPicker.mode != providerPickerCustom {
		t.Fatalf("custom row should open custom input, got %#v", m.providerPicker)
	}
	m = updateProviderPickerKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("corp-gpt")})
	m = updateProviderPickerKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if got := agent.switchedModels; len(got) != 1 || got[0] != "corp-gpt" {
		t.Fatalf("switchedModels = %#v, want custom model", got)
	}
}

func TestProviderPicker_RenderShowsProviderStatusAndPlainModelRows(t *testing.T) {
	agent := &stubAgent{
		statusLine: "ready",
		providerCandidates: []providerpicker.ProviderCandidate{
			{Key: "openai", Label: "openai", Current: true, CredentialStatus: providerpicker.ProviderCredentialConfigured},
		},
		modelCandidates: map[string][]providerpicker.ModelCandidate{
			"openai": {
				{Name: "gpt-current", Current: true},
				{Name: "gpt-default", Default: true},
			},
		},
	}
	m := submitProviderPickerCommand(t, newSizedPromptTestModel(agent, 80, 24), "/provider")
	providerView := termtext.StripANSI(m.View())
	if !strings.Contains(providerView, "configured") {
		t.Fatalf("provider view should show credential status:\n%s", providerView)
	}

	m = updateProviderPickerKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	modelView := strings.ToLower(termtext.StripANSI(m.View()))
	for _, forbidden := range []string{"recommended", "fast", "latest"} {
		if strings.Contains(modelView, forbidden) {
			t.Fatalf("model view should not contain %q:\n%s", forbidden, modelView)
		}
	}
	if !strings.Contains(modelView, "gpt-current") || !strings.Contains(modelView, "current") {
		t.Fatalf("model view should show model name and minimal tags:\n%s", modelView)
	}
}

func submitProviderPickerCommand(t *testing.T, m Model, input string) Model {
	t.Helper()
	m.textInput.SetValue(input)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	return updated.(Model)
}

func updateProviderPickerKey(t *testing.T, m Model, key tea.KeyMsg) Model {
	t.Helper()
	updated, _ := m.Update(key)
	return updated.(Model)
}
