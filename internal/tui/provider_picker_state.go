package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/susugadx/xelyon-cli/internal/providerpicker"
)

type providerPickerMode int

const (
	providerPickerProviders providerPickerMode = iota
	providerPickerModels
	providerPickerCustom
)

type providerPickerStep int

const (
	providerPickerStepProviderSelect providerPickerStep = iota
	providerPickerStepModelSelect
	providerPickerStepModelCustom
	providerPickerStepAzureDeploymentInput
	providerPickerStepAzureCatalogModelSelect
	providerPickerStepAzureCatalogModelCustom
)

type providerPickerState struct {
	mode            providerPickerMode
	step            providerPickerStep
	provider        string
	providerLabel   string
	providers       []providerpicker.ProviderCandidate
	models          []providerpicker.ModelCandidate
	selected        int
	filter          string
	filtering       bool
	currentOnly     bool
	azureDeployment string
	customInput     textinput.Model
}

func newProviderPickerState(candidates []providerpicker.ProviderCandidate) *providerPickerState {
	state := &providerPickerState{
		mode:      providerPickerProviders,
		step:      providerPickerStepProviderSelect,
		providers: append([]providerpicker.ProviderCandidate(nil), candidates...),
	}
	state.selected = initialProviderPickerSelection(state.providerRows())
	return state
}

func newModelPickerState(provider string, candidates []providerpicker.ModelCandidate, currentOnly bool) *providerPickerState {
	state := &providerPickerState{
		mode:        providerPickerModels,
		step:        providerPickerStepModelSelect,
		provider:    provider,
		models:      append([]providerpicker.ModelCandidate(nil), candidates...),
		currentOnly: currentOnly,
	}
	state.providerLabel = provider
	state.selected = initialModelPickerSelection(state.modelRows())
	return state
}

func (p *providerPickerState) beginCustomInput(step providerPickerStep) {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = providerPickerCustomPlaceholder(step)
	input.CharLimit = 0
	input.Width = 80
	input.Focus()
	p.mode = providerPickerCustom
	p.step = step
	p.customInput = input
}

func providerPickerCustomPlaceholder(step providerPickerStep) string {
	switch step {
	case providerPickerStepAzureDeploymentInput:
		return "deployment name"
	case providerPickerStepAzureCatalogModelCustom:
		return "catalog model"
	default:
		return "model or deployment name"
	}
}

func (p providerPickerState) providerRows() []providerpicker.ProviderCandidate {
	if strings.TrimSpace(p.filter) == "" {
		return append([]providerpicker.ProviderCandidate(nil), p.providers...)
	}
	filter := strings.ToLower(strings.TrimSpace(p.filter))
	rows := make([]providerpicker.ProviderCandidate, 0, len(p.providers))
	for _, candidate := range p.providers {
		haystack := strings.ToLower(candidate.Key + " " + candidate.Label + " " + string(candidate.CredentialStatus))
		if strings.Contains(haystack, filter) {
			rows = append(rows, candidate)
		}
	}
	return rows
}

func (p providerPickerState) modelRows() []providerpicker.ModelCandidate {
	if strings.TrimSpace(p.filter) == "" {
		return append([]providerpicker.ModelCandidate(nil), p.models...)
	}
	filter := strings.ToLower(strings.TrimSpace(p.filter))
	rows := make([]providerpicker.ModelCandidate, 0, len(p.models))
	for _, candidate := range p.models {
		if strings.Contains(strings.ToLower(candidate.Name), filter) {
			rows = append(rows, candidate)
		}
	}
	return rows
}

func (p *providerPickerState) visibleRowCount() int {
	if p == nil {
		return 0
	}
	if p.mode == providerPickerProviders {
		return len(p.providerRows())
	}
	return len(p.modelRows())
}

func (p *providerPickerState) moveSelection(delta int) {
	if p == nil {
		return
	}
	count := p.visibleRowCount()
	if count == 0 {
		p.selected = 0
		return
	}
	p.selected += delta
	if p.selected < 0 {
		p.selected = 0
	}
	if p.selected >= count {
		p.selected = count - 1
	}
}

func (p *providerPickerState) clampSelection() {
	if p == nil {
		return
	}
	count := p.visibleRowCount()
	if count <= 0 {
		p.selected = 0
		return
	}
	if p.selected >= count {
		p.selected = count - 1
	}
	if p.selected < 0 {
		p.selected = 0
	}
}

func (p providerPickerState) selectedProvider() (providerpicker.ProviderCandidate, bool) {
	rows := p.providerRows()
	if p.selected < 0 || p.selected >= len(rows) {
		return providerpicker.ProviderCandidate{}, false
	}
	return rows[p.selected], true
}

func (p providerPickerState) selectedModel() (providerpicker.ModelCandidate, bool) {
	rows := p.modelRows()
	if p.selected < 0 || p.selected >= len(rows) {
		return providerpicker.ModelCandidate{}, false
	}
	return rows[p.selected], true
}

func initialProviderPickerSelection(rows []providerpicker.ProviderCandidate) int {
	for i, row := range rows {
		if row.Current {
			return i
		}
	}
	return 0
}

func initialModelPickerSelection(rows []providerpicker.ModelCandidate) int {
	for i, row := range rows {
		if row.Current {
			return i
		}
	}
	for i, row := range rows {
		if row.Default {
			return i
		}
	}
	return 0
}

func (p providerPickerMode) String() string {
	switch p {
	case providerPickerProviders:
		return "providers"
	case providerPickerModels:
		return "models"
	case providerPickerCustom:
		return "custom"
	default:
		return fmt.Sprintf("providerPickerMode(%d)", int(p))
	}
}

func (p providerPickerStep) String() string {
	switch p {
	case providerPickerStepProviderSelect:
		return "provider_select"
	case providerPickerStepModelSelect:
		return "model_select"
	case providerPickerStepModelCustom:
		return "model_custom"
	case providerPickerStepAzureDeploymentInput:
		return "azure_deployment_input"
	case providerPickerStepAzureCatalogModelSelect:
		return "azure_catalog_model_select"
	case providerPickerStepAzureCatalogModelCustom:
		return "azure_catalog_model_custom"
	default:
		return fmt.Sprintf("providerPickerStep(%d)", int(p))
	}
}
