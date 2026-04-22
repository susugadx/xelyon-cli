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

func prepareProjectMapInjection(agent *Agent, input string) (projectMapInjectionContext, bool) {
	sources, ok := resolveProjectMapInjectionSources(agent)
	if !ok {
		return projectMapInjectionContext{}, false
	}
	return buildProjectMapInjectionContext(agent, input, sources)
}

func resolveProjectMapInjectionSources(agent *Agent) (projectMapInjectionSources, bool) {
	cfg := agent.cfg()
	if !cfg.ProjectMap.Enabled {
		return projectMapInjectionSources{}, false
	}
	if !common.IsRipgrepAvailable() {
		return projectMapInjectionSources{}, false
	}

	cwd, ok := resolveProjectMapSourceCWD()
	if !ok {
		return projectMapInjectionSources{}, false
	}

	pc := loadProjectConfig()
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
