package agent

import (
	"github.com/susugadx/xelyon-cli/internal/repomap"
)

type projectMapInjectionContext struct {
	pm          *repomap.ProjectMap
	rebuilt     bool
	maxTokens   int
	baseKey     string
	focusPaths  []string
	focusKey    string
	fileCount   int
	symbolCount int
}

type projectMapSectionBuild struct {
	baseSection       string
	focusSection      string
	projectMapPrompt  string
	effectiveFocusKey string
}

// injectProjectMap はプロジェクト構造マップをシステムプロンプトに注入する。
func injectProjectMap(agent *Agent, input string) {
	injectProjectMapWithOverrides(agent, input, projectMapInjectionOverrides{})
}

func injectProjectMapWithOverrides(agent *Agent, input string, overrides projectMapInjectionOverrides) {
	if agent == nil {
		return
	}

	resetProjectMapPromptSection(agent)

	injectionCtx, ok := prepareProjectMapInjectionWithOverrides(agent, input, overrides)
	if !ok {
		return
	}

	if applyProjectMapCachedSection(agent, injectionCtx) {
		return
	}

	build, ok := buildProjectMapSection(agent, injectionCtx)
	if !ok {
		resetProjectMapCachedSections(agent)
		agent.projectMapDirty = false
		return
	}

	applyProjectMapBuiltSection(agent, injectionCtx, build)
	if injectionCtx.rebuilt {
		green.Fprintf(agent.output(), "🗺️  Project map loaded (%d files, %d symbols)\n", agent.projectMapFileCount, agent.projectMapSymbolCount)
	}
}

func resetProjectMapPromptSection(agent *Agent) {
	if agent == nil {
		return
	}
	agent.SystemPrompt = stripProjectMapSection(agent.SystemPrompt)
	resetProjectMapRuntimeCounts(agent)
}

func resetProjectMapRuntimeCounts(agent *Agent) {
	if agent == nil {
		return
	}
	agent.projectMapFileCount = 0
	agent.projectMapSymbolCount = 0
}

func resetProjectMapCachedSections(agent *Agent) {
	agent.projectMapBaseSection = ""
	agent.projectMapFocusSection = ""
	agent.projectMapSection = ""
	agent.projectMapBaseKey = ""
	agent.projectMapFocusKey = ""
}
