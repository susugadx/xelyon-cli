package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/tui/providerpickerscreen"
)

func (m Model) openProviderPicker() (Model, tea.Cmd) {
	m.switchToComposerInput()
	m.providerPicker = providerpickerscreen.NewProvider(m.providerModels.ProviderCandidates())
	m.clearSlashSuggestions()
	m.chromeDirty = true
	return m, nil
}

func (m Model) openCurrentProviderModelPicker() (Model, tea.Cmd) {
	provider := strings.TrimSpace(m.configAgent.GetProviderConfigKey())
	if provider == "" {
		provider = strings.TrimSpace(m.configAgent.GetProviderName())
	}
	if provider == "" {
		m.setTransientStatus("No current provider")
		return m, nil
	}
	m.switchToComposerInput()
	m.providerPicker = providerpickerscreen.NewModel(provider, m.providerModels.ModelCandidates(provider), true)
	m.clearSlashSuggestions()
	m.chromeDirty = true
	return m, nil
}

func (m Model) updateWithProviderPickerOpen(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleProviderPickerKeyMsg(msg)
	case tea.MouseMsg:
		if isPromptBackgroundWheelMsg(msg) {
			return m.forwardMessageBehindProviderPicker(msg)
		}
		return m, nil
	default:
		return m.forwardMessageBehindProviderPicker(msg)
	}
}

func (m Model) forwardMessageBehindProviderPicker(msg tea.Msg) (Model, tea.Cmd) {
	active := m.providerPicker
	m.providerPicker = nil

	updated, cmd := m.Update(msg)
	next := updated.(Model)
	next.providerPicker = active
	return next, cmd
}

func (m Model) handleProviderPickerKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.providerPicker == nil {
		return m, nil
	}

	result, cmd := m.providerPicker.HandleKey(msg)
	switch result.Command {
	case providerpickerscreen.CommandClose:
		m.closeProviderPicker("Selection cancelled")
	case providerpickerscreen.CommandSelectProvider:
		m.providerPicker.ShowModels(result.Provider, m.providerModels.ModelCandidates(result.Provider.Key))
	case providerpickerscreen.CommandApplyModel:
		return m.applyProviderPickerModel(result.ProviderKey, result.CurrentOnly, result.Model)
	case providerpickerscreen.CommandBeginAzureCatalogModelSelection:
		return m.beginAzureCatalogModelSelection(result.Deployment)
	case providerpickerscreen.CommandReturnToAzureDeploymentPicker:
		return m.returnToAzureDeploymentPicker()
	case providerpickerscreen.CommandApplyAzureDeploymentSetup:
		return m.applyAzureDeploymentSetup(result.Deployment, result.CatalogModel)
	case providerpickerscreen.CommandRequiredMessage:
		m.setTransientStatus(result.Message)
	}

	m.chromeDirty = true
	return m, cmd
}

func (m Model) beginAzureCatalogModelSelection(deployment string) (Model, tea.Cmd) {
	deployment = strings.TrimSpace(deployment)
	if deployment == "" {
		m.setTransientStatus("Deployment is required")
		return m, nil
	}
	m.providerPicker.BeginAzureCatalogModelSelection(deployment, m.providerModels.AzureCatalogModelCandidates(deployment))
	m.chromeDirty = true
	return m, nil
}

func (m Model) returnToAzureDeploymentPicker() (Model, tea.Cmd) {
	m.providerPicker.ReturnToAzureDeploymentPicker(m.providerModels.ModelCandidates(m.providerPicker.Provider()))
	m.chromeDirty = true
	return m, nil
}

func (m Model) applyProviderPickerModel(provider string, currentOnly bool, model string) (Model, tea.Cmd) {
	provider = strings.TrimSpace(provider)
	currentProviderOnly := currentOnly || provider == ""
	m.providerPicker = nil
	m.chromeDirty = true

	var err error
	if currentProviderOnly {
		err = m.providerModels.SwitchModelForCurrentProvider(model)
	} else {
		err = m.providerModels.SwitchProviderModel(provider, model)
	}
	if err != nil {
		m.setTransientStatus(err.Error())
	} else {
		m.setTransientStatus("Selection applied")
	}
	m.refreshStatusLine()
	return m, nil
}

func (m Model) applyAzureDeploymentSetup(deployment string, catalogModel string) (Model, tea.Cmd) {
	deployment = strings.TrimSpace(deployment)
	catalogModel = strings.TrimSpace(catalogModel)
	m.providerPicker = nil
	m.chromeDirty = true

	if err := m.providerModels.ConfigureAndSwitchAzureDeployment(deployment, catalogModel); err != nil {
		m.setTransientStatus(err.Error())
	} else {
		m.setTransientStatus("Selection applied")
	}
	m.refreshStatusLine()
	return m, nil
}

func (m *Model) closeProviderPicker(status string) {
	m.providerPicker = nil
	m.setTransientStatus(status)
	m.chromeDirty = true
}
