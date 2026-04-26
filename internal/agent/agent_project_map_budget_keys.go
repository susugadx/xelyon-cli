package agent

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func buildProjectMapBaseKey(agent *Agent, cfg *config.Config, maxTokens, fileCount, symbolCount int) string {
	stateKey := ""
	if agent != nil {
		stateKey = agent.projectMapStateKey
	}
	contextWindow := 0
	if agent != nil {
		contextWindow = agent.currentModelTokenLimit(cfg)
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
	contextWindow := agent.currentModelTokenLimit(cfg)
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
