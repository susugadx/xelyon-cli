package tui

import (
	"fmt"
	"strings"
	"sync"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerpicker"
)

type providerModelSwitchCall struct {
	Provider string
	Model    string
}

type stubAgent struct {
	mu                        sync.RWMutex
	processing                bool
	cancelCalls               int
	cleanupCalls              int
	copyCalls                 int
	copyTexts                 []string
	chatInputs                []string
	chatImageInputs           []string
	handledInputs             []string
	handledCommands           map[string]bool
	statusLine                string
	saveStatusLine            string
	providerName              string
	providerConfigKey         string
	providerCandidates        []providerpicker.ProviderCandidate
	modelCandidates           map[string][]providerpicker.ModelCandidate
	azureCatalogModels        []providerpicker.ModelCandidate
	azureCatalogModelRequests []string
	switchedProviders         []providerModelSwitchCall
	switchedModels            []string
	configuredAzure           []azureDeploymentSetupCall
	switchProviderErr         error
	switchModelErr            error
	configureAzureErr         error
	saveErr                   error          // non-nil にすると SaveAndSyncConfig が失敗する
	lastSavedConfig           *config.Config // SaveAndSyncConfig で受け取った最後の Config
	projectConfig             *config.ProjectConfig
	projectLoadErr            error
	projectSaveErr            error
	projectCreateErr          error
	lastSavedProject          *config.ProjectConfig
	savedProjects             []*config.ProjectConfig
}

type azureDeploymentSetupCall struct {
	Deployment   string
	CatalogModel string
}

func (s *stubAgent) Chat(input string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.chatInputs = append(s.chatInputs, input)
}
func (s *stubAgent) ChatWithImagePath(input string, imagePath string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.chatImageInputs = append(s.chatImageInputs, input+"||"+imagePath)
}
func (s *stubAgent) HandleCommand(cmd string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handledInputs = append(s.handledInputs, cmd)
	return s.handledCommands[cmd]
}
func (s *stubAgent) GetStatusLine() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.statusLine
}
func (s *stubAgent) StatusSnapshot() StatusSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return StatusSnapshot{
		Mode:       s.statusLine,
		LegacyLine: s.statusLine,
	}
}
func (s *stubAgent) Cancel() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cancelCalls++
}
func (s *stubAgent) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupCalls++
}
func (s *stubAgent) IsProcessing() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.processing
}
func (s *stubAgent) CopyLastOutput() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.copyCalls++
	return "Copied 5 lines", nil
}
func (s *stubAgent) CopyText(text string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.copyCalls++
	s.copyTexts = append(s.copyTexts, text)
	return nil
}
func (s *stubAgent) LoadConfigForEdit() (*config.Config, error) {
	return config.DefaultConfig(), nil
}
func (s *stubAgent) SaveAndSyncConfig(cfg *config.Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastSavedConfig = config.CloneConfig(cfg)
	if s.saveErr == nil && s.saveStatusLine != "" {
		s.statusLine = s.saveStatusLine
	}
	return s.saveErr
}
func (s *stubAgent) LoadProjectForEdit() (*config.ProjectConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.projectLoadErr != nil {
		return nil, s.projectLoadErr
	}
	return config.CloneProjectConfig(s.projectConfig), nil
}
func (s *stubAgent) SaveProjectConfig(pc *config.ProjectConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastSavedProject = config.CloneProjectConfig(pc)
	s.savedProjects = append(s.savedProjects, config.CloneProjectConfig(pc))
	return s.projectSaveErr
}
func (s *stubAgent) CreateProjectConfigTemplate() (*config.ProjectConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.projectCreateErr != nil {
		return nil, s.projectCreateErr
	}
	s.projectConfig = &config.ProjectConfig{
		Context: "template context",
		Rules:   []string{"template rule"},
	}
	return config.CloneProjectConfig(s.projectConfig), nil
}
func (s *stubAgent) GetProviderName() string {
	if s.providerName != "" {
		return s.providerName
	}
	return "deepseek"
}
func (s *stubAgent) GetProviderConfigKey() string {
	if s.providerConfigKey != "" {
		return s.providerConfigKey
	}
	return s.GetProviderName()
}
func (s *stubAgent) ProviderCandidates() []providerpicker.ProviderCandidate {
	if s.providerCandidates != nil {
		return append([]providerpicker.ProviderCandidate(nil), s.providerCandidates...)
	}
	return []providerpicker.ProviderCandidate{
		{Key: "deepseek", Label: "deepseek", CredentialStatus: providerpicker.ProviderCredentialMissingKey},
		{Key: "openai", Label: "openai", Current: true, CredentialStatus: providerpicker.ProviderCredentialConfigured},
	}
}
func (s *stubAgent) ModelCandidates(provider string) []providerpicker.ModelCandidate {
	if s.modelCandidates != nil {
		if candidates, ok := s.modelCandidates[provider]; ok {
			return append([]providerpicker.ModelCandidate(nil), candidates...)
		}
	}
	return []providerpicker.ModelCandidate{
		{Name: "model-a", Current: true},
		{Name: "model-b", Default: true},
		{Name: "Custom model...", Custom: true},
	}
}
func (s *stubAgent) AzureCatalogModelCandidates(deployment string) []providerpicker.ModelCandidate {
	s.mu.Lock()
	s.azureCatalogModelRequests = append(s.azureCatalogModelRequests, deployment)
	s.mu.Unlock()
	if s.azureCatalogModels != nil {
		return append([]providerpicker.ModelCandidate(nil), s.azureCatalogModels...)
	}
	return []providerpicker.ModelCandidate{
		{Name: "gpt-5.3-codex"},
		{Name: "gpt-5.5-pro"},
		{Name: "Custom catalog model...", Custom: true},
	}
}
func (s *stubAgent) SwitchProviderModel(provider string, model string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.switchedProviders = append(s.switchedProviders, providerModelSwitchCall{
		Provider: provider,
		Model:    model,
	})
	return s.switchProviderErr
}
func (s *stubAgent) SwitchModelForCurrentProvider(model string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.switchedModels = append(s.switchedModels, model)
	return s.switchModelErr
}
func (s *stubAgent) ConfigureAzureDeployment(deployment string, catalogModel string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.configuredAzure = append(s.configuredAzure, azureDeploymentSetupCall{
		Deployment:   deployment,
		CatalogModel: catalogModel,
	})
	return s.configureAzureErr
}
func (s *stubAgent) ConfigureAndSwitchAzureDeployment(deployment string, catalogModel string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.configuredAzure = append(s.configuredAzure, azureDeploymentSetupCall{
		Deployment:   deployment,
		CatalogModel: catalogModel,
	})
	if s.configureAzureErr != nil {
		return s.configureAzureErr
	}
	if s.switchProviderErr != nil {
		return s.switchProviderErr
	}
	s.switchedProviders = append(s.switchedProviders, providerModelSwitchCall{
		Provider: "azure",
		Model:    deployment,
	})
	return nil
}
func (s *stubAgent) lastChatInput() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.chatInputs) == 0 {
		return ""
	}
	return s.chatInputs[len(s.chatInputs)-1]
}

func (s *stubAgent) lastImageChatInput() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.chatImageInputs) == 0 {
		return ""
	}
	return s.chatImageInputs[len(s.chatImageInputs)-1]
}

func (s *stubAgent) setProcessing(processing bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.processing = processing
}

func (s *stubAgent) setStatusLine(status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statusLine = status
}

func setModelRawLines(m *Model, count int) {
	lines := make([]string, count)
	for i := 0; i < count; i++ {
		lines[i] = fmt.Sprintf("line%d", i)
	}
	m.rawLines = append([]string(nil), lines...)
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
}

func newModelWithViewport(agent AgentInterface) Model {
	m := NewModel(agent, "")
	m.ready = true
	m.width = 80
	m.height = 30
	m.vp = lightViewport{width: 80, height: m.height - m.footerHeight()}
	m.padLineCache = strings.Repeat(" ", 80)
	return m
}
