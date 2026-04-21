package agent

import "github.com/susugadx/xelyon-cli/internal/agent/token"

func applyProjectMapCachedSection(agent *Agent, injectionCtx projectMapInjectionContext) bool {
	if injectionCtx.rebuilt ||
		agent.projectMapBaseSection == "" ||
		agent.projectMapBaseKey != injectionCtx.baseKey ||
		agent.projectMapFocusKey != injectionCtx.focusKey ||
		agent.projectMapSection == "" ||
		token.EstimateTokenCount(agent.projectMapBaseSection) > injectionCtx.maxTokens ||
		token.EstimateTokenCount(agent.projectMapSection) > injectionCtx.maxTokens {
		return false
	}

	agent.SystemPrompt = appendProjectMapSection(agent.SystemPrompt, agent.projectMapSection)
	agent.projectMapFileCount = injectionCtx.fileCount
	agent.projectMapSymbolCount = injectionCtx.symbolCount
	agent.projectMapDirty = false
	return true
}

func applyProjectMapBuiltSection(agent *Agent, injectionCtx projectMapInjectionContext, build projectMapSectionBuild) {
	agent.SystemPrompt = appendProjectMapSection(agent.SystemPrompt, build.projectMapPrompt)
	agent.projectMapFileCount = injectionCtx.fileCount
	agent.projectMapSymbolCount = injectionCtx.symbolCount
	agent.projectMapBaseSection = build.baseSection
	agent.projectMapFocusSection = build.focusSection
	agent.projectMapSection = build.projectMapPrompt
	agent.projectMapBaseKey = injectionCtx.baseKey
	agent.projectMapFocusKey = build.effectiveFocusKey
	agent.projectMapDirty = false
}
