package providerpickerscreen

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/providerpicker"
)

// Command は provider picker が root Model に要求する操作を表す。
type Command int

const (
	// CommandNone は root 側の操作が不要な入力処理を表す。
	CommandNone Command = iota
	// CommandClose は picker を閉じる要求を表す。
	CommandClose
	// CommandSelectProvider は provider 選択後に model 候補取得を要求する。
	CommandSelectProvider
	// CommandApplyModel は model/deployment の適用を要求する。
	CommandApplyModel
	// CommandBeginAzureCatalogModelSelection は Azure catalog model 候補取得を要求する。
	CommandBeginAzureCatalogModelSelection
	// CommandReturnToAzureDeploymentPicker は Azure deployment 候補への復帰を要求する。
	CommandReturnToAzureDeploymentPicker
	// CommandApplyAzureDeploymentSetup は Azure deployment/catalog_model 設定を要求する。
	CommandApplyAzureDeploymentSetup
	// CommandRequiredMessage は custom 入力の必須エラー表示を要求する。
	CommandRequiredMessage
)

// KeyResult はキー入力処理の結果を表す。
type KeyResult struct {
	Command      Command
	Provider     providerpicker.ProviderCandidate
	ProviderKey  string
	CurrentOnly  bool
	Model        string
	Deployment   string
	CatalogModel string
	Message      string
}

// HandleKey は provider picker のキー入力を処理する。
func (p *Screen) HandleKey(msg tea.KeyMsg) (KeyResult, tea.Cmd) {
	if p == nil {
		return KeyResult{}, nil
	}
	if p.mode == ModeCustom {
		return p.handleCustomKey(msg)
	}

	switch {
	case msg.Type == tea.KeyEsc || msg.Type == tea.KeyCtrlC:
		return KeyResult{Command: CommandClose}, nil
	case msg.Type == tea.KeyUp || (!p.filtering && msg.String() == "k"):
		p.moveSelection(-1)
	case msg.Type == tea.KeyDown || (!p.filtering && msg.String() == "j"):
		p.moveSelection(1)
	case msg.String() == "/":
		p.filtering = true
		p.filter = ""
		p.selected = 0
	case isBackspaceKey(msg):
		return p.handleBackspace(), nil
	case isEnterKey(msg):
		return p.submitSelection(), nil
	case p.filtering && msg.Type == tea.KeyRunes:
		p.filter += string(msg.Runes)
		p.selected = 0
	}

	p.clampSelection()
	return KeyResult{}, nil
}

func (p *Screen) handleBackspace() KeyResult {
	if p.filtering && p.filter != "" {
		runes := []rune(p.filter)
		p.filter = string(runes[:len(runes)-1])
		p.selected = 0
		p.clampSelection()
		return KeyResult{}
	}
	if p.mode != ModeModels {
		return KeyResult{}
	}
	if p.step == StepAzureCatalogModelSelect {
		return KeyResult{Command: CommandReturnToAzureDeploymentPicker}
	}
	if p.currentOnly {
		return KeyResult{Command: CommandClose}
	}
	p.mode = ModeProviders
	p.step = StepProviderSelect
	p.filter = ""
	p.filtering = false
	p.selected = initialProviderPickerSelection(p.providerRows())
	return KeyResult{}
}

func (p *Screen) submitSelection() KeyResult {
	switch p.mode {
	case ModeProviders:
		provider, ok := p.selectedProvider()
		if !ok {
			return KeyResult{}
		}
		return KeyResult{Command: CommandSelectProvider, Provider: provider}
	case ModeModels:
		model, ok := p.selectedModel()
		if !ok {
			return KeyResult{}
		}
		if model.Custom {
			if p.step == StepAzureCatalogModelSelect {
				p.beginCustomInput(StepAzureCatalogModelCustom)
			} else {
				p.beginCustomInput(customModelStep(p.provider, p.currentOnly))
			}
			return KeyResult{}
		}
		if p.step == StepAzureCatalogModelSelect {
			return KeyResult{
				Command:      CommandApplyAzureDeploymentSetup,
				Deployment:   p.azureDeployment,
				CatalogModel: model.Name,
			}
		}
		if p.isAzureSetupDeploymentSelection() {
			return KeyResult{
				Command:    CommandBeginAzureCatalogModelSelection,
				Deployment: model.Name,
			}
		}
		return KeyResult{
			Command:     CommandApplyModel,
			ProviderKey: p.provider,
			CurrentOnly: p.currentOnly,
			Model:       model.Name,
		}
	default:
		return KeyResult{}
	}
}

func (p *Screen) handleCustomKey(msg tea.KeyMsg) (KeyResult, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyEsc || msg.Type == tea.KeyCtrlC:
		p.cancelCustomInput()
		return KeyResult{}, nil
	case isEnterKey(msg):
		value := strings.TrimSpace(p.customInput.Value())
		if value == "" {
			return KeyResult{Command: CommandRequiredMessage, Message: requiredMessage(p.step)}, nil
		}
		switch p.step {
		case StepAzureDeploymentInput:
			return KeyResult{Command: CommandBeginAzureCatalogModelSelection, Deployment: value}, nil
		case StepAzureCatalogModelCustom:
			return KeyResult{
				Command:      CommandApplyAzureDeploymentSetup,
				Deployment:   p.azureDeployment,
				CatalogModel: value,
			}, nil
		}
		return KeyResult{
			Command:     CommandApplyModel,
			ProviderKey: p.provider,
			CurrentOnly: p.currentOnly,
			Model:       value,
		}, nil
	default:
		var cmd tea.Cmd
		p.customInput, cmd = p.customInput.Update(msg)
		return KeyResult{}, cmd
	}
}

func (p *Screen) cancelCustomInput() {
	switch p.step {
	case StepAzureCatalogModelCustom:
		p.mode = ModeModels
		p.step = StepAzureCatalogModelSelect
	default:
		p.mode = ModeModels
		p.step = StepModelSelect
	}
}

func requiredMessage(step Step) string {
	switch step {
	case StepAzureDeploymentInput:
		return "Deployment is required"
	case StepAzureCatalogModelCustom:
		return "Catalog model is required"
	default:
		return "Model is required"
	}
}

func customModelStep(provider string, currentOnly bool) Step {
	if !currentOnly && strings.TrimSpace(provider) == "azure" {
		return StepAzureDeploymentInput
	}
	return StepModelCustom
}

func (p *Screen) isAzureSetupDeploymentSelection() bool {
	return p != nil &&
		!p.currentOnly &&
		p.mode == ModeModels &&
		p.step == StepModelSelect &&
		strings.TrimSpace(p.provider) == "azure"
}

func isEnterKey(msg tea.KeyMsg) bool {
	if msg.Type == tea.KeyEnter {
		return true
	}
	s := msg.String()
	return s == "enter" || s == "\r" || s == "\n"
}

func isBackspaceKey(msg tea.KeyMsg) bool {
	if msg.Type == tea.KeyBackspace || msg.Type == tea.KeyCtrlH {
		return true
	}
	s := msg.String()
	return s == "backspace" || s == "ctrl+h"
}
