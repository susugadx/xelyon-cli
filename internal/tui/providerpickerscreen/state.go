package providerpickerscreen

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/susugadx/xelyon-cli/internal/providerpicker"
)

// Mode は provider picker の表示モードを表す。
type Mode int

const (
	// ModeProviders は provider 一覧を表す。
	ModeProviders Mode = iota
	// ModeModels は model/deployment/catalog 一覧を表す。
	ModeModels
	// ModeCustom は custom 入力を表す。
	ModeCustom
)

// Step は provider picker の現在の選択段階を表す。
type Step int

const (
	// StepProviderSelect は provider 選択を表す。
	StepProviderSelect Step = iota
	// StepModelSelect は model/deployment 選択を表す。
	StepModelSelect
	// StepModelCustom は custom model/deployment 入力を表す。
	StepModelCustom
	// StepAzureDeploymentInput は Azure deployment 入力を表す。
	StepAzureDeploymentInput
	// StepAzureCatalogModelSelect は Azure catalog model 選択を表す。
	StepAzureCatalogModelSelect
	// StepAzureCatalogModelCustom は Azure catalog model 入力を表す。
	StepAzureCatalogModelCustom
)

// Screen は provider/model picker の UI state/input/render を保持する。
type Screen struct {
	mode            Mode
	step            Step
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

// Snapshot は root tests が picker の公開状態を確認するための読み取り専用状態。
type Snapshot struct {
	Mode         Mode
	Step         Step
	Provider     string
	ProviderRows []providerpicker.ProviderCandidate
	ModelRows    []providerpicker.ModelCandidate
	CurrentOnly  bool
}

// NewProvider は provider 選択 picker を構築する。
func NewProvider(candidates []providerpicker.ProviderCandidate) *Screen {
	state := &Screen{
		mode:      ModeProviders,
		step:      StepProviderSelect,
		providers: append([]providerpicker.ProviderCandidate(nil), candidates...),
	}
	state.selected = initialProviderPickerSelection(state.providerRows())
	return state
}

// NewModel は現在 provider の model 選択 picker を構築する。
func NewModel(provider string, candidates []providerpicker.ModelCandidate, currentOnly bool) *Screen {
	state := &Screen{
		mode:        ModeModels,
		step:        StepModelSelect,
		provider:    provider,
		models:      append([]providerpicker.ModelCandidate(nil), candidates...),
		currentOnly: currentOnly,
	}
	state.providerLabel = provider
	state.selected = initialModelPickerSelection(state.modelRows())
	return state
}

// Snapshot は picker の公開状態を返す。
func (p *Screen) Snapshot() Snapshot {
	if p == nil {
		return Snapshot{}
	}
	return Snapshot{
		Mode:         p.mode,
		Step:         p.step,
		Provider:     p.provider,
		ProviderRows: p.providerRows(),
		ModelRows:    p.modelRows(),
		CurrentOnly:  p.currentOnly,
	}
}

// Provider は現在選択中 provider key を返す。
func (p *Screen) Provider() string {
	if p == nil {
		return ""
	}
	return p.provider
}

// ShowModels は選択 provider の model/deployment 候補を表示する。
func (p *Screen) ShowModels(provider providerpicker.ProviderCandidate, candidates []providerpicker.ModelCandidate) {
	p.mode = ModeModels
	p.step = StepModelSelect
	p.provider = provider.Key
	p.providerLabel = provider.Label
	p.models = append([]providerpicker.ModelCandidate(nil), candidates...)
	p.filter = ""
	p.filtering = false
	p.selected = initialModelPickerSelection(p.modelRows())
}

// BeginAzureCatalogModelSelection は deployment 選択後の catalog model 候補を表示する。
func (p *Screen) BeginAzureCatalogModelSelection(deployment string, candidates []providerpicker.ModelCandidate) {
	p.mode = ModeModels
	p.step = StepAzureCatalogModelSelect
	p.azureDeployment = strings.TrimSpace(deployment)
	p.models = append([]providerpicker.ModelCandidate(nil), candidates...)
	p.filter = ""
	p.filtering = false
	p.selected = initialModelPickerSelection(p.modelRows())
}

// ReturnToAzureDeploymentPicker は catalog model 選択から deployment 選択へ戻す。
func (p *Screen) ReturnToAzureDeploymentPicker(candidates []providerpicker.ModelCandidate) {
	p.mode = ModeModels
	p.step = StepModelSelect
	p.azureDeployment = ""
	p.models = append([]providerpicker.ModelCandidate(nil), candidates...)
	p.filter = ""
	p.filtering = false
	p.selected = initialModelPickerSelection(p.modelRows())
}

func (p *Screen) beginCustomInput(step Step) {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = providerPickerCustomPlaceholder(step)
	input.CharLimit = 0
	input.Width = 80
	input.Focus()
	p.mode = ModeCustom
	p.step = step
	p.customInput = input
}

func providerPickerCustomPlaceholder(step Step) string {
	switch step {
	case StepAzureDeploymentInput:
		return "deployment name"
	case StepAzureCatalogModelCustom:
		return "catalog model"
	default:
		return "model or deployment name"
	}
}

func (p Screen) providerRows() []providerpicker.ProviderCandidate {
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

func (p Screen) modelRows() []providerpicker.ModelCandidate {
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

func (p *Screen) visibleRowCount() int {
	if p == nil {
		return 0
	}
	if p.mode == ModeProviders {
		return len(p.providerRows())
	}
	return len(p.modelRows())
}

func (p *Screen) moveSelection(delta int) {
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

func (p *Screen) clampSelection() {
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

func (p Screen) selectedProvider() (providerpicker.ProviderCandidate, bool) {
	rows := p.providerRows()
	if p.selected < 0 || p.selected >= len(rows) {
		return providerpicker.ProviderCandidate{}, false
	}
	return rows[p.selected], true
}

func (p Screen) selectedModel() (providerpicker.ModelCandidate, bool) {
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

func (p Mode) String() string {
	switch p {
	case ModeProviders:
		return "providers"
	case ModeModels:
		return "models"
	case ModeCustom:
		return "custom"
	default:
		return fmt.Sprintf("providerPickerMode(%d)", int(p))
	}
}

func (p Step) String() string {
	switch p {
	case StepProviderSelect:
		return "provider_select"
	case StepModelSelect:
		return "model_select"
	case StepModelCustom:
		return "model_custom"
	case StepAzureDeploymentInput:
		return "azure_deployment_input"
	case StepAzureCatalogModelSelect:
		return "azure_catalog_model_select"
	case StepAzureCatalogModelCustom:
		return "azure_catalog_model_custom"
	default:
		return fmt.Sprintf("providerPickerStep(%d)", int(p))
	}
}
