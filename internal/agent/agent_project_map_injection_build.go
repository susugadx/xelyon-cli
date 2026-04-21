package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/token"
)

func buildProjectMapSection(agent *Agent, injectionCtx projectMapInjectionContext) (projectMapSectionBuild, bool) {
	baseSection := agent.projectMapBaseSection
	if injectionCtx.rebuilt || agent.projectMapBaseKey != injectionCtx.baseKey || strings.TrimSpace(baseSection) == "" {
		baseSection = injectionCtx.pm.GenerateManifest(nil)
	}

	focusSection := renderProjectMapFocusOverlay(injectionCtx.focusPaths)
	projectMapPrompt := composeProjectMapPromptSection(baseSection, focusSection)
	if projectMapPrompt != "" && token.EstimateTokenCount(projectMapPrompt) > injectionCtx.maxTokens {
		projectMapPrompt = composeProjectMapPromptSection(baseSection, "")
		focusSection = ""
	}
	if projectMapPrompt == "" {
		return projectMapSectionBuild{}, false
	}

	focusCount := countProjectMapFocusLines(focusSection)
	if focusCount > len(injectionCtx.focusPaths) {
		focusCount = len(injectionCtx.focusPaths)
	}

	return projectMapSectionBuild{
		baseSection:       baseSection,
		focusSection:      focusSection,
		projectMapPrompt:  projectMapPrompt,
		effectiveFocusKey: buildProjectMapFocusKey(injectionCtx.focusPaths[:focusCount]),
	}, true
}
