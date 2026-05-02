package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

type projectMapInjectionSources struct {
	cfg            *config.Config
	cwd            string
	rootPath       string
	ignorePatterns []string
	ignoreKey      string
}

type projectMapInjectionOverrides struct {
	invocationCWD string
	projectConfig *config.ProjectConfig
}

func prepareProjectMapInjectionWithOverrides(agent *Agent, input string, overrides projectMapInjectionOverrides) (projectMapInjectionContext, bool) {
	sources, ok := resolveProjectMapInjectionSourcesWithOverrides(agent, overrides)
	if !ok {
		return projectMapInjectionContext{}, false
	}
	return buildProjectMapInjectionContext(agent, input, sources)
}

func resolveProjectMapInjectionSourcesWithOverrides(agent *Agent, overrides projectMapInjectionOverrides) (projectMapInjectionSources, bool) {
	cfg := agent.cfg()
	if !cfg.ProjectMap.Enabled {
		return projectMapInjectionSources{}, false
	}
	if !common.IsRipgrepAvailable() {
		return projectMapInjectionSources{}, false
	}

	cwd, ok := resolveProjectMapSourceCWD(agent, overrides.invocationCWD)
	if !ok {
		return projectMapInjectionSources{}, false
	}

	pc := overrides.projectConfig
	if pc == nil {
		pc = agent.loadProjectConfig()
	}
	rootPath := resolveProjectMapSourceRootPath(cwd, pc)
	ignorePatterns := config.ResolveSharedIgnorePatterns(cfg, pc)

	return projectMapInjectionSources{
		cfg:            cfg,
		cwd:            cwd,
		rootPath:       rootPath,
		ignorePatterns: ignorePatterns,
		ignoreKey:      strings.Join(ignorePatterns, "\x00"),
	}, true
}

func buildProjectMapInjectionContext(agent *Agent, input string, sources projectMapInjectionSources) (projectMapInjectionContext, bool) {
	pm, rebuilt := ensureProjectMap(agent, sources.rootPath, sources.ignorePatterns, sources.ignoreKey)
	if pm == nil {
		return projectMapInjectionContext{}, false
	}

	fileCount := pm.GetFileCount()
	symbolCount := pm.GetSymbolCount()
	maxTokens := calcProjectMapBudget(agent, sources.cfg, fileCount, symbolCount)
	pm.MaxTokens = maxTokens

	focusPaths := extractProjectMapFocusPaths(sources.cwd, sources.rootPath, input, projectMapFocusMaxPaths)
	return projectMapInjectionContext{
		pm:          pm,
		rebuilt:     rebuilt,
		maxTokens:   maxTokens,
		baseKey:     buildProjectMapBaseKey(agent, sources.cfg, maxTokens, fileCount, symbolCount),
		focusPaths:  focusPaths,
		focusKey:    buildProjectMapFocusKey(focusPaths),
		fileCount:   fileCount,
		symbolCount: symbolCount,
	}, true
}
