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

type projectMapInjector struct {
	agent *Agent
	input string
}

// injectProjectMap はプロジェクト構造マップをシステムプロンプトに注入する。
func injectProjectMap(agent *Agent, input string) {
	injector := projectMapInjector{
		agent: agent,
		input: input,
	}
	injector.inject()
}

func (i projectMapInjector) inject() {
	if i.agent == nil {
		return
	}

	resetProjectMapPromptSection(i.agent)

	injectionCtx, ok := prepareProjectMapInjection(i.agent, i.input)
	if !ok {
		return
	}

	if applyProjectMapCachedSection(i.agent, injectionCtx) {
		return
	}

	build, ok := buildProjectMapSection(i.agent, injectionCtx)
	if !ok {
		resetProjectMapCachedSections(i.agent)
		i.agent.projectMapDirty = false
		return
	}

	applyProjectMapBuiltSection(i.agent, injectionCtx, build)
	if injectionCtx.rebuilt {
		green.Fprintf(i.agent.output(), "🗺️  Project map loaded (%d files, %d symbols)\n", i.agent.projectMapFileCount, i.agent.projectMapSymbolCount)
	}
}

func resetProjectMapPromptSection(agent *Agent) {
	agent.promptManager().StripProjectMapSection()
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
