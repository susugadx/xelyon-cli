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

type providerPickerState struct {
	mode          providerPickerMode
	provider      string
	providerLabel string
	providers     []providerpicker.ProviderCandidate
	models        []providerpicker.ModelCandidate
	selected      int
	filter        string
	filtering     bool
	currentOnly   bool
	customInput   textinput.Model
}

func newProviderPickerState(candidates []providerpicker.ProviderCandidate) *providerPickerState {
	state := &providerPickerState{
		mode:      providerPickerProviders,
		providers: append([]providerpicker.ProviderCandidate(nil), candidates...),
	}
	state.selected = initialProviderPickerSelection(state.providerRows())
	return state
}

func newModelPickerState(provider string, candidates []providerpicker.ModelCandidate, currentOnly bool) *providerPickerState {
	state := &providerPickerState{
		mode:        providerPickerModels,
		provider:    provider,
		models:      append([]providerpicker.ModelCandidate(nil), candidates...),
		currentOnly: currentOnly,
	}
	state.providerLabel = provider
	state.selected = initialModelPickerSelection(state.modelRows())
	return state
}

func (p *providerPickerState) beginCustomInput() {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = "model or deployment name"
	input.CharLimit = 0
	input.Width = 80
	input.Focus()
	p.mode = providerPickerCustom
	p.customInput = input
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
