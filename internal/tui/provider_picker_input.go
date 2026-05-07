package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) openProviderPicker() (Model, tea.Cmd) {
	m.switchToComposerInput()
	m.providerPicker = newProviderPickerState(m.providerModels.ProviderCandidates())
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
	m.providerPicker = newModelPickerState(provider, m.providerModels.ModelCandidates(provider), true)
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
	if m.providerPicker.mode == providerPickerCustom {
		return m.handleProviderPickerCustomKeyMsg(msg)
	}

	switch {
	case msg.Type == tea.KeyEsc || msg.Type == tea.KeyCtrlC:
		m.closeProviderPicker("Selection cancelled")
		return m, nil
	case msg.Type == tea.KeyUp || (!m.providerPicker.filtering && msg.String() == "k"):
		m.providerPicker.moveSelection(-1)
	case msg.Type == tea.KeyDown || (!m.providerPicker.filtering && msg.String() == "j"):
		m.providerPicker.moveSelection(1)
	case msg.String() == "/":
		m.providerPicker.filtering = true
		m.providerPicker.filter = ""
		m.providerPicker.selected = 0
	case isBackspaceKey(msg):
		return m.handleProviderPickerBackspace()
	case isEnterKey(msg):
		return m.submitProviderPickerSelection()
	case m.providerPicker.filtering && msg.Type == tea.KeyRunes:
		m.providerPicker.filter += string(msg.Runes)
		m.providerPicker.selected = 0
	}

	m.providerPicker.clampSelection()
	m.chromeDirty = true
	return m, nil
}

func (m Model) handleProviderPickerBackspace() (Model, tea.Cmd) {
	if m.providerPicker.filtering && m.providerPicker.filter != "" {
		runes := []rune(m.providerPicker.filter)
		m.providerPicker.filter = string(runes[:len(runes)-1])
		m.providerPicker.selected = 0
		m.providerPicker.clampSelection()
		m.chromeDirty = true
		return m, nil
	}
	if m.providerPicker.mode == providerPickerModels {
		if m.providerPicker.step == providerPickerStepAzureCatalogModelSelect {
			return m.returnToAzureDeploymentPicker()
		}
		if m.providerPicker.currentOnly {
			m.closeProviderPicker("Selection cancelled")
			return m, nil
		}
		m.providerPicker.mode = providerPickerProviders
		m.providerPicker.step = providerPickerStepProviderSelect
		m.providerPicker.filter = ""
		m.providerPicker.filtering = false
		m.providerPicker.selected = initialProviderPickerSelection(m.providerPicker.providerRows())
		m.chromeDirty = true
	}
	return m, nil
}

func (m Model) submitProviderPickerSelection() (Model, tea.Cmd) {
	switch m.providerPicker.mode {
	case providerPickerProviders:
		provider, ok := m.providerPicker.selectedProvider()
		if !ok {
			return m, nil
		}
		m.providerPicker.mode = providerPickerModels
		m.providerPicker.step = providerPickerStepModelSelect
		m.providerPicker.provider = provider.Key
		m.providerPicker.providerLabel = provider.Label
		m.providerPicker.models = m.providerModels.ModelCandidates(provider.Key)
		m.providerPicker.filter = ""
		m.providerPicker.filtering = false
		m.providerPicker.selected = initialModelPickerSelection(m.providerPicker.modelRows())
		m.chromeDirty = true
		return m, nil
	case providerPickerModels:
		model, ok := m.providerPicker.selectedModel()
		if !ok {
			return m, nil
		}
		if model.Custom {
			if m.providerPicker.step == providerPickerStepAzureCatalogModelSelect {
				m.providerPicker.beginCustomInput(providerPickerStepAzureCatalogModelCustom)
			} else {
				m.providerPicker.beginCustomInput(providerPickerCustomModelStep(m.providerPicker.provider, m.providerPicker.currentOnly))
			}
			m.chromeDirty = true
			return m, nil
		}
		if m.providerPicker.step == providerPickerStepAzureCatalogModelSelect {
			return m.applyAzureDeploymentSetup(m.providerPicker.azureDeployment, model.Name)
		}
		if m.providerPicker.isAzureSetupDeploymentSelection() {
			return m.beginAzureCatalogModelSelection(model.Name)
		}
		return m.applyProviderPickerModel(model.Name)
	default:
		return m, nil
	}
}

func (m Model) handleProviderPickerCustomKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyEsc || msg.Type == tea.KeyCtrlC:
		return m.cancelProviderPickerCustomInput()
	case isEnterKey(msg):
		value := strings.TrimSpace(m.providerPicker.customInput.Value())
		if value == "" {
			m.setTransientStatus(providerPickerRequiredMessage(m.providerPicker.step))
			return m, nil
		}
		switch m.providerPicker.step {
		case providerPickerStepAzureDeploymentInput:
			return m.beginAzureCatalogModelSelection(value)
		case providerPickerStepAzureCatalogModelCustom:
			return m.applyAzureDeploymentSetup(m.providerPicker.azureDeployment, value)
		}
		return m.applyProviderPickerModel(value)
	default:
		var cmd tea.Cmd
		m.providerPicker.customInput, cmd = m.providerPicker.customInput.Update(msg)
		m.chromeDirty = true
		return m, cmd
	}
}

func (m Model) cancelProviderPickerCustomInput() (Model, tea.Cmd) {
	switch m.providerPicker.step {
	case providerPickerStepAzureCatalogModelCustom:
		m.providerPicker.mode = providerPickerModels
		m.providerPicker.step = providerPickerStepAzureCatalogModelSelect
	default:
		m.providerPicker.mode = providerPickerModels
		m.providerPicker.step = providerPickerStepModelSelect
	}
	m.chromeDirty = true
	return m, nil
}

func providerPickerRequiredMessage(step providerPickerStep) string {
	switch step {
	case providerPickerStepAzureDeploymentInput:
		return "Deployment is required"
	case providerPickerStepAzureCatalogModelCustom:
		return "Catalog model is required"
	default:
		return "Model is required"
	}
}

func providerPickerCustomModelStep(provider string, currentOnly bool) providerPickerStep {
	if !currentOnly && strings.TrimSpace(provider) == "azure" {
		return providerPickerStepAzureDeploymentInput
	}
	return providerPickerStepModelCustom
}

func (p *providerPickerState) isAzureSetupDeploymentSelection() bool {
	return p != nil &&
		!p.currentOnly &&
		p.mode == providerPickerModels &&
		p.step == providerPickerStepModelSelect &&
		strings.TrimSpace(p.provider) == "azure"
}

func (m Model) beginAzureCatalogModelSelection(deployment string) (Model, tea.Cmd) {
	deployment = strings.TrimSpace(deployment)
	if deployment == "" {
		m.setTransientStatus("Deployment is required")
		return m, nil
	}
	m.providerPicker.mode = providerPickerModels
	m.providerPicker.step = providerPickerStepAzureCatalogModelSelect
	m.providerPicker.azureDeployment = deployment
	m.providerPicker.models = m.providerModels.AzureCatalogModelCandidates(deployment)
	m.providerPicker.filter = ""
	m.providerPicker.filtering = false
	m.providerPicker.selected = initialModelPickerSelection(m.providerPicker.modelRows())
	m.chromeDirty = true
	return m, nil
}

func (m Model) returnToAzureDeploymentPicker() (Model, tea.Cmd) {
	m.providerPicker.mode = providerPickerModels
	m.providerPicker.step = providerPickerStepModelSelect
	m.providerPicker.azureDeployment = ""
	m.providerPicker.models = m.providerModels.ModelCandidates(m.providerPicker.provider)
	m.providerPicker.filter = ""
	m.providerPicker.filtering = false
	m.providerPicker.selected = initialModelPickerSelection(m.providerPicker.modelRows())
	m.chromeDirty = true
	return m, nil
}

func (m Model) applyProviderPickerModel(model string) (Model, tea.Cmd) {
	provider := strings.TrimSpace(m.providerPicker.provider)
	currentProviderOnly := m.providerPicker.currentOnly || provider == ""
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
