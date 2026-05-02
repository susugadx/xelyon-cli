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

type projectMapSourceResolveOptions struct {
	allowBundleLoad bool
}

func prepareProjectMapInjection(agent *Agent, input string) (projectMapInjectionContext, bool) {
	sources, _, ok := resolveProjectMapInjectionSources(agent, projectMapSourceResolveOptions{
		allowBundleLoad: true,
	})
	if !ok {
		return projectMapInjectionContext{}, false
	}
	return buildProjectMapInjectionContext(agent, input, sources)
}

func resolveProjectMapInjectionSources(agent *Agent, opts projectMapSourceResolveOptions) (projectMapInjectionSources, projectPromptRefreshReason, bool) {
	if agent == nil {
		return projectMapInjectionSources{}, refreshReasonNoAgent, false
	}
	cfg := agent.cfg()
	if cfg == nil || !cfg.ProjectMap.Enabled {
		return projectMapInjectionSources{}, refreshReasonProjectMapDisabled, false
	}
	if !common.IsRipgrepAvailable() {
		return projectMapInjectionSources{}, refreshReasonRipgrepUnavailable, false
	}

	cwd, ok := resolveProjectMapSourceCWD()
	if !ok {
		return projectMapInjectionSources{}, refreshReasonCWDUnavailable, false
	}

	bundle := agent.projectInstructionBundleIfLoaded()
	if bundle == nil && opts.allowBundleLoad {
		bundle = agent.loadProjectInstructionBundleCached(false)
	}
	rootPath := resolveProjectMapSourceRootPath(cwd, bundle)
	if bundle == nil && strings.TrimSpace(agent.projectMapRootPath) != "" {
		rootPath = agent.projectMapRootPath
	}

	var projectCfg *config.ProjectConfig
	if bundle != nil {
		projectCfg = bundle.ProjectConfig
	}
	ignorePatterns := config.ResolveSharedIgnorePatterns(cfg, projectCfg)

	return projectMapInjectionSources{
		cfg:            cfg,
		cwd:            cwd,
		rootPath:       rootPath,
		ignorePatterns: ignorePatterns,
		ignoreKey:      strings.Join(ignorePatterns, "\x00"),
	}, refreshReasonNoChange, true
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
