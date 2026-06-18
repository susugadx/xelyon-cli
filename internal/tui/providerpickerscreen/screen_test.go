package providerpickerscreen

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/providerpicker"
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
)

func TestHandleKeySelectsProviderThenAppliesModel(t *testing.T) {
	screen := NewProvider([]providerpicker.ProviderCandidate{
		{Key: "openai", Label: "OpenAI", Current: true},
		{Key: "gemini", Label: "Gemini"},
	})

	screen.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	result, _ := screen.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if result.Command != CommandSelectProvider || result.Provider.Key != "gemini" {
		t.Fatalf("provider result = %#v, want gemini selection", result)
	}

	screen.ShowModels(result.Provider, []providerpicker.ModelCandidate{
		{Name: "gemini-current", Current: true},
		{Name: "gemini-next"},
	})
	screen.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	result, _ = screen.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if result.Command != CommandApplyModel {
		t.Fatalf("model command = %d, want apply model", result.Command)
	}
	if result.ProviderKey != "gemini" || result.Model != "gemini-next" || result.CurrentOnly {
		t.Fatalf("model result = %#v, want provider gemini model gemini-next", result)
	}
}

func TestHandleKeyAzureCustomDeploymentRequestsCatalogSelection(t *testing.T) {
	screen := NewModel("azure", []providerpicker.ModelCandidate{{Name: "Custom deployment...", Custom: true}}, false)

	result, _ := screen.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if result.Command != CommandNone || screen.Snapshot().Step != StepAzureDeploymentInput {
		t.Fatalf("custom deployment should enter deployment input, result=%#v snapshot=%#v", result, screen.Snapshot())
	}

	screen.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("corp-deployment")})
	result, _ = screen.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if result.Command != CommandBeginAzureCatalogModelSelection || result.Deployment != "corp-deployment" {
		t.Fatalf("deployment result = %#v, want catalog selection request", result)
	}

	screen.BeginAzureCatalogModelSelection(result.Deployment, []providerpicker.ModelCandidate{{Name: "gpt-catalog"}})
	result, _ = screen.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if result.Command != CommandApplyAzureDeploymentSetup {
		t.Fatalf("catalog result command = %d, want azure setup", result.Command)
	}
	if result.Deployment != "corp-deployment" || result.CatalogModel != "gpt-catalog" {
		t.Fatalf("catalog result = %#v, want deployment/catalog pair", result)
	}
}

func TestPanelLinesShowsPlainProviderAndModelRows(t *testing.T) {
	screen := NewProvider([]providerpicker.ProviderCandidate{{
		Key:              "openai",
		Label:            "OpenAI\nProvider",
		Current:          true,
		CredentialStatus: providerpicker.ProviderCredentialConfigured,
	}})
	plain := termtext.StripANSI(strings.Join(screen.PanelLines(80, 24), "\n"))
	for _, want := range []string{"Provider", "OpenAI Provider (openai)", "configured", "current"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("provider panel missing %q:\n%s", want, plain)
		}
	}
	if strings.Contains(plain, "OpenAI\nProvider") {
		t.Fatalf("provider panel contains unsanitized newline:\n%s", plain)
	}
}
