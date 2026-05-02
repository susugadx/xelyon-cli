package agent

import (
	"fmt"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/prompt"
	promptplan "github.com/susugadx/xelyon-cli/internal/prompt/plan"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

type PromptManager struct {
	agent *Agent
}

func newPromptManager(agent *Agent) *PromptManager {
	return &PromptManager{agent: agent}
}

func (a *Agent) promptManager() *PromptManager {
	return newPromptManager(a)
}

func (m *PromptManager) RebuildSystemPromptForCurrentProvider() {
	a := m.agent
	if a == nil || a.CurrentProvider == nil {
		return
	}

	planningPrompt := promptplan.BuildPlanningPrompt()
	hadPlanPrompt := strings.Contains(a.SystemPrompt, planningPrompt)

	providerName := a.ProviderName
	if providerName == "" {
		providerName = providerRuntimeNameFromProvider(a.CurrentProvider)
	}
	systemPrompt := prompt.GetSystemPromptForProviderWithConfig(providerName, a.CurrentModel, a.cfg())
	if a.mcpManager != nil && len(a.mcpManager.GetTools()) > 0 {
		systemPrompt += buildMCPToolsPrompt(a.mcpManager)
	}
	systemPrompt = prompt.BuildProviderSystemPromptWithConfig(systemPrompt, providerName, a.CurrentModel, a.cfg())

	if bundle := loadProjectInstructionBundle(a.cfg()); bundle != nil {
		systemPrompt = injectProjectInstructionBundle(systemPrompt, bundle, "")
	}

	if hadPlanPrompt {
		systemPrompt += api.SystemPromptCacheBoundary + planningPrompt
	}

	a.SystemPrompt = systemPrompt
	injectProjectMap(a, "")
}

func (m *PromptManager) RefreshProjectPrompt(input string) {
	a := m.agent
	if a == nil {
		return
	}

	bundle := loadProjectInstructionBundle(a.cfg())
	newConfigBlock := buildProjectInstructionBlock(bundle, input)

	oldConfigBlock := prompt.ExtractProjectConfigBlock(a.SystemPrompt)
	if strings.TrimSpace(newConfigBlock) != strings.TrimSpace(oldConfigBlock) {
		systemPrompt := stripProjectMapSection(prompt.StripProjectConfigSections(a.SystemPrompt))
		if newConfigBlock != "" {
			systemPrompt = prompt.InjectProjectConfigBlock(systemPrompt, newConfigBlock)
		}
		a.SystemPrompt = systemPrompt
	} else {
		a.SystemPrompt = stripProjectMapSection(a.SystemPrompt)
	}
	injectProjectMap(a, input)
}

func (m *PromptManager) RefreshProjectPromptIfDirty(input string) {
	if m == nil || !m.ShouldRefreshProjectPrompt(input) {
		return
	}
	m.RefreshProjectPrompt(input)
}

func (m *PromptManager) ShouldRefreshProjectPrompt(input string) bool {
	a := m.agent
	if a == nil {
		return false
	}
	if a.projectMapDirty {
		return true
	}

	cfg := a.cfg()
	if cfg == nil || !cfg.ProjectMap.Enabled || !common.IsRipgrepAvailable() {
		return false
	}

	cwd, err := os.Getwd()
	if err != nil {
		return false
	}

	bundle := loadProjectInstructionBundle(a.cfg())
	rootPath := resolveProjectMapSourceRootPath(cwd, bundle)

	if stateKey := currentProjectMapStateKey(a, rootPath); stateKey != "" && stateKey != a.projectMapStateKey {
		return true
	}

	baseKey := buildProjectMapBaseKey(a, cfg, calcProjectMapBudget(a, cfg, a.projectMapFileCount, a.projectMapSymbolCount), a.projectMapFileCount, a.projectMapSymbolCount)
	if a.projectMapBaseKey != baseKey {
		return true
	}

	focusPaths := extractProjectMapFocusPaths(cwd, rootPath, input, projectMapFocusMaxPaths)
	if a.projectMapFocusKey != buildProjectMapFocusKey(focusPaths) {
		return true
	}

	return a.projectMap == nil || a.projectMapBaseSection == "" || a.projectMapSection == ""
}

func (m *PromptManager) InvalidateProjectMap() {
	a := m.agent
	if a == nil {
		return
	}

	a.projectMap = nil
	a.projectMapRootPath = ""
	a.projectMapIgnoreKey = ""
	a.projectMapStateKey = ""
	a.projectMapWatchDirs = nil
	a.projectMapBaseSection = ""
	a.projectMapFocusSection = ""
	a.projectMapSection = ""
	a.projectMapBaseKey = ""
	a.projectMapFocusKey = ""
	a.projectMapDirty = true
}

func (m *PromptManager) DebugString() string {
	a := m.agent
	if a == nil {
		return "<nil>"
	}
	return fmt.Sprintf("PromptManager(model=%s, dirty=%t, files=%d, symbols=%d)", a.CurrentModel, a.projectMapDirty, a.projectMapFileCount, a.projectMapSymbolCount)
}
