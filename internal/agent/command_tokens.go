package agent

import (
	"fmt"
	"strings"
)

// handleTokensCommand はトークン使用量を表示
func handleTokensCommand(agent *Agent) bool {
	cfg := agent.cfg()
	out := agent.output()

	// トークン推定
	totalTokens := agent.EstimateTokens()
	systemTokens := agent.EstimateSystemPromptTokens()
	historyTokens := agent.EstimateHistoryTokens()
	limit := agent.currentModelTokenLimit(cfg)
	percentage := float64(totalTokens) / float64(limit) * 100

	// 表示
	printCommandHeaderToWriter(out, "Token Usage / トークン使用量")
	_, _ = fmt.Fprintln(out)

	// 使用量バー表示
	barWidth := 30
	filled := int(percentage / 100 * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	// 色分け
	if percentage > 90 {
		red.Fprintf(out, "  [%s] %.1f%%\n", bar, percentage)
	} else if percentage > 80 {
		yellow.Fprintf(out, "  [%s] %.1f%%\n", bar, percentage)
	} else {
		green.Fprintf(out, "  [%s] %.1f%%\n", bar, percentage)
	}

	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintf(out, "  Current: %s / %s tokens\n", formatNumber(totalTokens), formatNumber(limit))

	_, _ = fmt.Fprintln(out)
	green.Fprintln(out, "📋 Breakdown:")
	_, _ = fmt.Fprintf(out, "    System Prompt: %s tokens (%.1f%%)\n",
		formatNumber(systemTokens), float64(systemTokens)/float64(limit)*100)

	// ツール定義トークン
	toolTokens := 0
	if agent.CurrentProvider != nil && agent.CurrentProvider.IsFunctionCallingEnabled() {
		toolTokens = agent.estimateToolDefinitionTokens()
	}
	builtinCount, mcpCount := agent.countToolsByType()
	toolLabel := fmt.Sprintf("%d", builtinCount)
	if mcpCount > 0 {
		toolLabel = fmt.Sprintf("%d+%d MCP", builtinCount, mcpCount)
	}
	_, _ = fmt.Fprintf(out, "    Tools (%s):   %s tokens (%.1f%%)\n",
		toolLabel, formatNumber(toolTokens), float64(toolTokens)/float64(limit)*100)

	_, _ = fmt.Fprintf(out, "    History:       %s tokens (%.1f%%)  [%d messages]\n",
		formatNumber(historyTokens), float64(historyTokens)/float64(limit)*100, len(agent.History))

	_, _ = fmt.Fprintln(out)
	green.Fprintln(out, "🤖 Model:")
	_, _ = fmt.Fprintf(out, "    %s (context: %s tokens)\n", agent.CurrentModel, formatNumber(limit))

	_, _ = fmt.Fprintln(out)
	green.Fprintln(out, "⚙️  Auto-compress:")
	if cfg.Compression.Enabled {
		thresholdPercent := cfg.Compression.TriggerPercent
		if thresholdPercent == 0 {
			thresholdPercent = 80
		}
		customThreshold := "disabled"
		if cfg.Compression.TokenThreshold > 0 {
			customThreshold = formatNumber(cfg.Compression.TokenThreshold) + " tokens"
		}
		_, _ = fmt.Fprintf(out, "    ON (custom: %s, percent: %d%%, model: %s)\n",
			customThreshold, thresholdPercent, agent.getCompressionModel())
	} else {
		_, _ = fmt.Fprintln(out, "    OFF")
	}

	// 警告
	if percentage > 90 {
		_, _ = fmt.Fprintln(out)
		red.Fprintln(out, "⚠️  Token usage is very high! Consider using /compress")
	} else if percentage > 80 {
		_, _ = fmt.Fprintln(out)
		yellow.Fprintln(out, "💡 Token usage is high. /compress available if needed")
	}

	cyan.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	return true
}
