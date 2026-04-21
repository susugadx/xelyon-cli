package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/repomap"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
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
	if agent == nil {
		return
	}

	resetProjectMapPromptSection(agent)

	injectionCtx, ok := prepareProjectMapInjection(agent, input)
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
	agent.SystemPrompt = stripProjectMapSection(agent.SystemPrompt)
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

func prepareProjectMapInjection(agent *Agent, input string) (projectMapInjectionContext, bool) {
	cfg := agent.cfg()
	if !cfg.ProjectMap.Enabled {
		return projectMapInjectionContext{}, false
	}
	if !common.IsRipgrepAvailable() {
		return projectMapInjectionContext{}, false
	}

	cwd, err := os.Getwd()
	if err != nil {
		return projectMapInjectionContext{}, false
	}

	pc := loadProjectConfig()
	rootPath := cwd
	if pc != nil && strings.TrimSpace(pc.FilePath) != "" {
		rootPath = filepath.Dir(pc.FilePath)
	}
	ignorePatterns := config.ResolveSharedIgnorePatterns(cfg, pc)
	ignoreKey := strings.Join(ignorePatterns, "\x00")

	pm, rebuilt := ensureProjectMap(agent, rootPath, ignorePatterns, ignoreKey)
	if pm == nil {
		return projectMapInjectionContext{}, false
	}

	fileCount := pm.GetFileCount()
	symbolCount := pm.GetSymbolCount()
	maxTokens := calcProjectMapBudget(agent, cfg, fileCount, symbolCount)
	pm.MaxTokens = maxTokens

	focusPaths := extractProjectMapFocusPaths(cwd, rootPath, input, projectMapFocusMaxPaths)
	return projectMapInjectionContext{
		pm:          pm,
		rebuilt:     rebuilt,
		maxTokens:   maxTokens,
		baseKey:     buildProjectMapBaseKey(agent, cfg, maxTokens, fileCount, symbolCount),
		focusPaths:  focusPaths,
		focusKey:    buildProjectMapFocusKey(focusPaths),
		fileCount:   fileCount,
		symbolCount: symbolCount,
	}, true
}

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

func buildProjectMapBaseKey(agent *Agent, cfg *config.Config, maxTokens, fileCount, symbolCount int) string {
	stateKey := ""
	if agent != nil {
		stateKey = agent.projectMapStateKey
	}
	contextWindow := 0
	if agent != nil {
		contextWindow = token.GetModelTokenLimit(agent.CurrentModel)
	}
	if contextWindow <= 0 {
		contextWindow = 128000
	}

	ratio := config.NormalizeProjectMapContextRatio(0)
	if cfg != nil {
		ratio = config.NormalizeProjectMapContextRatio(cfg.ProjectMap.ContextRatio)
	}
	effectiveRatio := effectiveProjectMapContextRatio(ratio, fileCount, symbolCount)

	return fmt.Sprintf("%s\x00budget:%d\x00ctx:%d\x00ratio:%.6f", stateKey, maxTokens, contextWindow, effectiveRatio)
}

func buildProjectMapFocusKey(paths []string) string {
	return strings.Join(dedupeProjectMapPriorityPaths(paths), "\x00")
}

func calcProjectMapBudget(agent *Agent, cfg *config.Config, fileCount, symbolCount int) int {
	// コンテキストウィンドウサイズを取得
	contextWindow := token.GetModelTokenLimit(agent.CurrentModel)
	if contextWindow <= 0 {
		contextWindow = 128000 // フォールバック
	}

	ratio := effectiveProjectMapContextRatio(cfg.ProjectMap.ContextRatio, fileCount, symbolCount)
	budgetCap := int(float64(contextWindow) * ratio)

	if budgetCap < 1 {
		return 1
	}

	return budgetCap
}

func effectiveProjectMapContextRatio(baseRatio float64, fileCount, symbolCount int) float64 {
	ratio := config.NormalizeProjectMapContextRatio(baseRatio)

	switch {
	case fileCount >= 400 || symbolCount >= 4000:
		if ratio < 0.04 {
			return 0.04
		}
	case fileCount >= 200 || symbolCount >= 2000:
		if ratio < 0.03 {
			return 0.03
		}
	}

	return ratio
}
