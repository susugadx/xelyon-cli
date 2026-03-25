package agent

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/tools/subagent"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

const (
	// defaultKeepRecent は /compress コマンドで保持するデフォルトのメッセージ数
	defaultKeepRecent = 10
)

func requestCacheMode(usage api.Usage) string {
	switch {
	case usage.CachedInputTokens > 0 && usage.CacheCreationTokens > 0:
		return "read + create"
	case usage.CachedInputTokens > 0:
		return "read"
	case usage.CacheCreationTokens > 0:
		return "create"
	default:
		return "none"
	}
}

func requestCacheHitRate(usage api.Usage) float64 {
	if usage.InputTokens <= 0 {
		return 0
	}
	return float64(usage.CachedInputTokens) / float64(usage.InputTokens) * 100.0
}

func requestUsageCost(provider, model string, usage api.Usage) float64 {
	return CalculateRequestCostWithCache(provider, model, usage) + usage.StorageCost
}

func buildLastRequestTable(provider, model string, usage *api.Usage, costOverride *float64) *ui.Table {
	if usage == nil {
		return nil
	}

	table := ui.NewTable().
		AddRow("Input", formatNumber(usage.InputTokens)+" tokens").
		AddRow("Cache Mode", requestCacheMode(*usage))

	if usage.CachedInputTokens > 0 || usage.CacheCreationTokens > 0 {
		table.AddRow("Cached", formatNumber(usage.CachedInputTokens)+" tokens").
			AddRow("Cache Creation", formatNumber(usage.CacheCreationTokens)+" tokens").
			AddRow("Hit Rate", fmt.Sprintf("%.1f%%", requestCacheHitRate(*usage)))
	}

	table.AddRow("Output", formatNumber(usage.OutputTokens)+" tokens")
	if usage.ThinkingTokens > 0 {
		table.AddRow("Thinking", formatNumber(usage.ThinkingTokens)+" tokens")
	}

	cost := requestUsageCost(provider, model, *usage)
	if costOverride != nil {
		cost = *costOverride
	}
	if cost > 0 {
		table.AddRow("Cost", fmt.Sprintf("$%.4f USD", cost))
	} else {
		table.AddRow("Cost", "Free (local)")
	}

	return table
}

func lastRequestUsageForStatus(stats *SessionStats) (*api.Usage, *float64) {
	if stats == nil {
		return nil, nil
	}
	if stats.LastTurnUsage != nil {
		cost := stats.LastTurnCost
		return stats.LastTurnUsage, &cost
	}
	return stats.LastUsage, nil
}

func getSessionFileInfo(agent *Agent) (string, int64) {
	sessionPath := ""
	sessionSize := int64(0)
	if agent.session != nil {
		sessionPath = fmt.Sprintf("~/.xelyon/history/%s.jsonl", agent.session.ID)
		if agent.storage != nil {
			homeDir, err := os.UserHomeDir()
			if err == nil {
				fullPath := fmt.Sprintf("%s/.xelyon/history/%s.jsonl", homeDir, agent.session.ID)
				if size, err := GetSessionFileSize(fullPath); err == nil {
					sessionSize = size
				}
			}
		}
	}
	return sessionPath, sessionSize
}

func sessionCacheHitRate(stats *SessionStats) float64 {
	if stats.InputTokens <= 0 {
		return 0
	}
	return float64(stats.CachedInputTokens) / float64(stats.InputTokens) * 100.0
}

func buildSessionOverviewTable(agent *Agent, stats *SessionStats) *ui.Table {
	sessionPath, sessionSize := getSessionFileInfo(agent)
	table := ui.NewTable().
		AddRow("Elapsed", stats.FormatElapsedTime()).
		AddRow("User Messages", fmt.Sprintf("%d", stats.UserMessages)).
		AddRow("Assistant Messages", fmt.Sprintf("%d", stats.AssistantMessages)).
		AddRow("Total Messages", fmt.Sprintf("%d", stats.TotalMessages())).
		AddRow("Tool Executions", fmt.Sprintf("%d", stats.TotalToolExecutions()))

	if sessionPath != "" {
		table.AddRow("Session File", sessionPath)
		if sessionSize > 0 {
			table.AddRow("Session Size", FormatFileSize(sessionSize))
		}
	}
	return table
}

func buildSessionTokenTable(agent *Agent, stats *SessionStats, subSummary *subagent.SubAgentSummary) *ui.Table {
	hasSubAgents := subSummary != nil && subSummary.TotalSpawned > 0
	if stats.TotalTokens() <= 0 && !hasSubAgents {
		return nil
	}

	tokenTable := ui.NewTable()
	currentTokens := agent.EstimateTokens()
	limit := token.GetModelTokenLimit(agent.CurrentModel)
	if limit > 0 {
		contextPct := float64(currentTokens) / float64(limit) * 100
		tokenTable.AddRow(tokenRowLabel(hasSubAgents, "Parent", "Context"), fmt.Sprintf("%s / %s (%.1f%%)", formatNumber(currentTokens), formatNumber(limit), contextPct))
	}

	cost := stats.EstimatedCost()
	if stats.TotalTokens() > 0 {
		tokenTable.AddRow(tokenRowLabel(hasSubAgents, "Parent", "Input"), formatNumber(stats.InputTokens)+" tokens")

		if stats.CachedInputTokens > 0 || stats.CacheCreationTokens > 0 {
			tokenTable.AddRow(tokenRowLabel(hasSubAgents, "Parent", "Cached"), formatNumber(stats.CachedInputTokens)+" tokens").
				AddRow(tokenRowLabel(hasSubAgents, "Parent", "Cache Creation"), formatNumber(stats.CacheCreationTokens)+" tokens").
				AddRow(tokenRowLabel(hasSubAgents, "Parent", "Hit Rate"), fmt.Sprintf("%.1f%%", sessionCacheHitRate(stats)))
		}

		tokenTable.AddRow(tokenRowLabel(hasSubAgents, "Parent", "Output"), formatNumber(stats.OutputTokens)+" tokens")
		if stats.ThinkingTokens > 0 {
			tokenTable.AddRow(tokenRowLabel(hasSubAgents, "Parent", "Thinking"), formatNumber(stats.ThinkingTokens)+" tokens")
		}

		tokenTable.AddRow(tokenRowLabel(hasSubAgents, "Parent", "Total"), formatNumber(stats.TotalTokens())+" tokens")
	}

	if hasSubAgents {
		tokenTable.AddRow("Sub-agent Input", formatNumber(subSummary.TotalInput)+" tokens")
		if subSummary.TotalCached > 0 {
			tokenTable.AddRow("Sub-agent Cached", formatNumber(subSummary.TotalCached)+" tokens")
			if subSummary.TotalInput > 0 {
				tokenTable.AddRow("Sub-agent Hit Rate", fmt.Sprintf("%.1f%%", subAgentCacheHitRate(*subSummary)))
			}
		}
		tokenTable.AddRow("Sub-agent Output", formatNumber(subSummary.TotalOutput)+" tokens")
		if subSummary.TotalThinking > 0 {
			tokenTable.AddRow("Sub-agent Thinking", formatNumber(subSummary.TotalThinking)+" tokens")
		}
		tokenTable.AddRow("Sub-agent Total", formatNumber(subAgentTotalTokens(*subSummary))+" tokens")
	}

	if hasSubAgents {
		tokenTable.AddRow("Parent Cost", formatParentCost(agent.ProviderName, cost)).
			AddRow("Sub-agent Cost", formatUSDWithSuffix(subSummary.TotalCost)).
			AddRow("Total Cost", formatUSDWithSuffix(cost+subSummary.TotalCost))
	} else if cost > 0 {
		tokenTable.AddRow("Cost", formatUSDWithSuffix(cost))
	} else {
		tokenTable.AddRow("Cost", "Free (local)")
	}
	return tokenTable
}

func tokenRowLabel(hasSubAgents bool, scope, label string) string {
	if !hasSubAgents {
		return label
	}
	return scope + " " + label
}

func subAgentCacheHitRate(summary subagent.SubAgentSummary) float64 {
	if summary.TotalInput <= 0 {
		return 0
	}
	return float64(summary.TotalCached) / float64(summary.TotalInput) * 100.0
}

func subAgentTotalTokens(summary subagent.SubAgentSummary) int {
	return summary.TotalInput + summary.TotalOutput + summary.TotalThinking
}

func printSessionSections(agent *Agent) {
	out := agent.output()
	if agent.Stats == nil {
		dim.Fprintln(out, "  Statistics not available")
		return
	}

	stats := agent.Stats
	var subSummary *subagent.SubAgentSummary
	if manager := agent.subAgentManager(); manager != nil {
		summary := manager.GetSummary()
		if summary.TotalSpawned > 0 {
			subSummary = &summary
		}
	}

	_, _ = fmt.Fprintln(out)
	green.Fprintln(out, "📚 Session")
	_, _ = fmt.Fprint(out, buildSessionOverviewTable(agent, stats).RenderCompact())

	_, _ = fmt.Fprintln(out)
	green.Fprintln(out, "🔧 Tool Executions")
	if stats.TotalToolExecutions() > 0 {
		toolTable := ui.NewTable()
		for tool, count := range stats.ToolExecutions {
			toolTable.AddRow(tool, fmt.Sprintf("%d", count))
		}
		toolTable.AddRow("Total", fmt.Sprintf("%d", stats.TotalToolExecutions()))
		_, _ = fmt.Fprint(out, toolTable.RenderCompact())
	} else {
		dim.Fprintln(out, "  No tools executed yet")
	}

	_, _ = fmt.Fprintln(out)
	green.Fprintln(out, "💰 Session Tokens & Cost")
	if tokenTable := buildSessionTokenTable(agent, stats, subSummary); tokenTable != nil {
		_, _ = fmt.Fprint(out, tokenTable.RenderCompact())
	} else {
		dim.Fprintln(out, "  No token usage data available")
	}

	if subSummary != nil {
		printSubAgentStats(out, *subSummary)
	}

	printToolObservabilitySection(out, stats)
	printSavingsSection(out, stats)

	_, _ = fmt.Fprintln(out)
	green.Fprintln(out, "⚡ Optimizations")
	opt := stats.Optimizations
	if opt.hasAny() {
		optTable := ui.NewTable()
		if opt.NegativeCacheHits > 0 {
			optTable.AddRow("Negative cache", fmt.Sprintf("%d hits", opt.NegativeCacheHits))
		}
		if opt.ErrorCompressions > 0 {
			optTable.AddRow("Error compression", fmt.Sprintf("%d times", opt.ErrorCompressions))
		}
		if opt.FailedPairCompressions > 0 {
			optTable.AddRow("Failed-pair compression", fmt.Sprintf("%d times", opt.FailedPairCompressions))
		}
		if opt.OutlineFirstCount > 0 {
			optTable.AddRow("Outline-first mode", fmt.Sprintf("%d times", opt.OutlineFirstCount))
		}
		if opt.CompactionCount > 0 {
			optTable.AddRow("Auto-compress", fmt.Sprintf("%d times", opt.CompactionCount))
		}
		if opt.CostAwareCompressions > 0 {
			optTable.AddRow("Cost-aware auto-compress", fmt.Sprintf("%d times", opt.CostAwareCompressions))
		}
		_, _ = fmt.Fprint(out, optTable.RenderCompact())
	} else {
		dim.Fprintln(out, "  No optimizations triggered yet")
	}
}

func formatUSD(value float64) string {
	return fmt.Sprintf("$%.4f", value)
}

func formatUSDWithSuffix(value float64) string {
	return fmt.Sprintf("%s USD", formatUSD(value))
}

func formatParentCost(providerName string, cost float64) string {
	if strings.EqualFold(providerName, "ollama") && cost == 0 {
		return "Free (local)"
	}
	return formatUSDWithSuffix(cost)
}

func formatSubAgentNumber(value int, pending bool) string {
	if pending {
		return "-"
	}
	return formatNumber(value)
}

func formatSubAgentCost(cost float64, pending bool) string {
	if pending {
		return "-"
	}
	return formatUSD(cost)
}

func formatSubAgentError(status, message string) string {
	if status == "running" {
		return ""
	}
	if status != "error" {
		return ""
	}
	message = firstStatusLine(strings.TrimSpace(message))
	if message == "" {
		return "unknown error"
	}
	return truncateStatusText(message, 120)
}

func firstStatusLine(s string) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return s[:idx]
	}
	return s
}

func truncateStatusText(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func printSubAgentStats(out io.Writer, summary subagent.SubAgentSummary) {
	if summary.TotalSpawned == 0 {
		return
	}

	_, _ = fmt.Fprintln(out)
	green.Fprintln(out, "🤖 Sub-agents")

	table := ui.NewTable().SetHeaders("ID", "Model", "Status", "Input", "Cached", "Output", "Thinking", "Cost", "Tools", "Error")
	for _, agentStats := range summary.Agents {
		pending := agentStats.Status == "running"
		model := agentStats.Model
		if model == "" {
			model = "-"
		}
		table.AddRow(
			agentStats.ID,
			model,
			agentStats.Status,
			formatSubAgentNumber(agentStats.InputTokens, pending),
			formatSubAgentNumber(agentStats.CachedTokens, pending),
			formatSubAgentNumber(agentStats.OutputTokens, pending),
			formatSubAgentNumber(agentStats.ThinkingTokens, pending),
			formatSubAgentCost(agentStats.Cost, pending),
			formatSubAgentNumber(agentStats.ToolExecutions, pending),
			formatSubAgentError(agentStats.Status, agentStats.ErrorMessage),
		)
	}

	table.AddRow(
		"Total",
		formatNumber(summary.TotalSpawned),
		"",
		formatNumber(summary.TotalInput),
		formatNumber(summary.TotalCached),
		formatNumber(summary.TotalOutput),
		formatNumber(summary.TotalThinking),
		formatUSD(summary.TotalCost),
		formatNumber(summary.TotalTools),
		"",
	)

	_, _ = fmt.Fprint(out, table.RenderCompact())

	// ツール内訳（ToolBreakdown がある場合のみ表示）
	hasBreakdown := false
	for _, agentStats := range summary.Agents {
		if len(agentStats.ToolBreakdown) > 0 {
			hasBreakdown = true
			break
		}
	}
	if hasBreakdown {
		_, _ = fmt.Fprintln(out)
		green.Fprintln(out, "🔧 Sub-agent Tool Breakdown")
		for _, agentStats := range summary.Agents {
			if len(agentStats.ToolBreakdown) == 0 {
				continue
			}
			_, _ = fmt.Fprintf(out, "  %s (%s):\n", agentStats.ID, agentStats.Model)
			bdTable := ui.NewTable().SetHeaders("Tool", "✓", "✗", "Total")
			for _, entry := range agentStats.ToolBreakdown {
				total := entry.Success + entry.Failures
				failStr := fmt.Sprintf("%d", entry.Failures)
				if entry.Failures > 0 {
					failStr = red.Sprintf("%d", entry.Failures)
				}
				bdTable.AddRow(
					entry.Tool,
					fmt.Sprintf("%d", entry.Success),
					failStr,
					fmt.Sprintf("%d", total),
				)
			}
			_, _ = fmt.Fprint(out, bdTable.RenderCompact())
		}
	}
}

// printToolObservabilitySection はツール選択のobservabilityセクションを表示する。
func printToolObservabilitySection(out io.Writer, stats *SessionStats) {
	obs := stats.ToolObs

	_, _ = fmt.Fprintln(out)
	green.Fprintln(out, "📈 Tool Selection")
	selTable := ui.NewTable()
	selTable.AddRow("read_file(batch)", fmt.Sprintf("%d", obs.ReadFileBatchCalls))
	selTable.AddRow("search_code(multi)", fmt.Sprintf("%d", obs.SearchCodeMultiPatternCalls))
	selTable.AddRow("search_code(batch merge)", fmt.Sprintf("%d", obs.SearchCodeBatchMerges))
	selTable.AddRow("read_file(batch merge)", fmt.Sprintf("%d", obs.ReadFileBatchMerges))
	selTable.AddRow("read_file empty-path errors", fmt.Sprintf("%d", obs.ReadFileEmptyPathsErrors))
	_, _ = fmt.Fprint(out, selTable.RenderCompact())
}

// printSavingsSection は API 入力トークン削減の推定量を表示する。
// batch merge は tool_call / tool result が個別に履歴に残るため含めない。
// 含むのは: same-turn duplicate の result サイズ差、compaction による圧縮分。
func printSavingsSection(out io.Writer, stats *SessionStats) {
	sav := stats.Savings
	if !sav.hasAny() {
		return
	}

	_, _ = fmt.Fprintln(out)
	green.Fprintln(out, "💰 Estimated Savings (API input)")
	savTable := ui.NewTable()
	if sav.SavedCalls > 0 {
		savTable.AddRow("Executions skipped", fmt.Sprintf("%d", sav.SavedCalls))
	}
	if sav.EstimatedInputTokensSaved > 0 {
		savTable.AddRow("~Input tokens saved", fmt.Sprintf("~%s", formatNumber(sav.EstimatedInputTokensSaved)))
	}
	if sav.EstimatedCostSaved > 0 {
		savTable.AddRow("~Cost saved", fmt.Sprintf("~$%.4f USD", sav.EstimatedCostSaved))
	}
	_, _ = fmt.Fprint(out, savTable.RenderCompact())
	dim.Fprintln(out, "  (~ = estimated, dedup result diff + compaction)")
}

// handleStatsCommand は /status の互換エイリアス
func handleStatsCommand(agent *Agent) bool {
	return handleStatusCommand(agent)
}

// formatNumber はカンマ区切りの数値を返す
func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	return fmt.Sprintf("%s,%03d", formatNumber(n/1000), n%1000)
}

// handleCopyCommand は最後のAI出力をクリップボードにコピー
func handleCopyCommand(agent *Agent, args []string) bool {
	out := agent.output()

	// 引数解析（ロック不要な部分を先に処理）
	codeOnly := false
	requestedN := 0 // 0 = デフォルト（最後の出力）
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "code":
			codeOnly = true
		case "-n":
			if i+1 < len(args) {
				n, err := strconv.Atoi(args[i+1])
				if err != nil {
					red.Fprintf(out, "Invalid number: %s\n", args[i+1])
					return true
				}
				requestedN = n
				i++
			} else {
				red.Fprintln(out, "Missing value for -n flag")
				return true
			}
		default:
			yellow.Fprintf(out, "Unknown argument: %s\n", arg)
			yellow.Fprintln(out, "Usage: /copy [code] [-n <number>]")
			return true
		}
	}

	// lastOutputs を1回のロックでスナップショット取得（TOCTOU 回避）
	agent.historyMu.Lock()
	outputCount := len(agent.lastOutputs)
	if outputCount == 0 {
		agent.historyMu.Unlock()
		yellow.Fprintln(out, "No AI output to copy yet")
		return true
	}
	outputIndex := outputCount - 1
	if requestedN > 0 {
		if requestedN < 1 || requestedN > outputCount {
			agent.historyMu.Unlock()
			red.Fprintf(out, "Index out of range (1-%d): %d\n", outputCount, requestedN)
			return true
		}
		outputIndex = outputCount - requestedN
	}
	output := agent.lastOutputs[outputIndex]
	agent.historyMu.Unlock()

	// コードブロックのみ抽出
	if codeOnly {
		codeBlocks := extractCodeBlocks(output)
		if len(codeBlocks) == 0 {
			yellow.Fprintln(out, "No code blocks found in output")
			return true
		}
		output = strings.Join(codeBlocks, "\n\n")
	}

	// クリップボードにコピー
	if err := clipboardWriteAll(output); err != nil {
		red.Fprintf(out, "Failed to copy to clipboard: %v\n", err)
		errText := strings.ToLower(err.Error())
		if strings.Contains(errText, "xclip") || strings.Contains(errText, "xsel") || strings.Contains(errText, "wl-copy") {
			yellow.Fprintln(out, "\nLinux requires a clipboard utility:")
			yellow.Fprintln(out, "  Ubuntu/Debian: sudo apt-get install xclip")
			yellow.Fprintln(out, "  Fedora/RHEL:   sudo dnf install xclip")
			yellow.Fprintln(out, "  Arch:          sudo pacman -S xclip")
		}
		return true
	}

	// 成功メッセージ
	lines := strings.Count(output, "\n") + 1
	chars := len(output)
	green.Fprintf(out, "✅ Copied to clipboard (%d lines, %d chars", lines, chars)
	if codeOnly {
		_, _ = fmt.Fprint(out, ", code blocks only")
	}
	_, _ = fmt.Fprintln(out, ")")

	return true
}

// extractCodeBlocks は ```で囲まれたコードブロックを抽出
func extractCodeBlocks(text string) []string {
	// 正規表現: ```language\n...```
	re := regexp.MustCompile("(?s)```\\w*\\n(.*?)```")
	matches := re.FindAllStringSubmatch(text, -1)

	blocks := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			blocks = append(blocks, strings.TrimSpace(match[1]))
		}
	}

	return blocks
}

// handleCompressCommand は会話履歴を圧縮
func handleCompressCommand(agent *Agent, args []string) bool {
	out := agent.output()

	// フラグ解析
	useCompactAPI := false
	keepRecent := defaultKeepRecent

	remainingArgs := []string{}
	for _, arg := range args {
		if arg == "--compact" || arg == "-c" {
			useCompactAPI = true
		} else {
			remainingArgs = append(remainingArgs, arg)
		}
	}

	// Compact API モード
	if useCompactAPI {
		return handleCompactAPICompress(agent)
	}

	// 引数解析
	if len(remainingArgs) > 0 {
		n, err := strconv.Atoi(remainingArgs[0])
		if err != nil {
			red.Fprintf(out, "Invalid number: %s\n", remainingArgs[0])
			yellow.Fprintln(out, "Usage: /compress [keep_recent] [--compact|-c]")
			return true
		}
		if n < 1 {
			red.Fprintln(out, "keep_recent must be at least 1")
			return true
		}
		keepRecent = n
	}

	// 確認プロンプト
	printCommandHeaderToWriter(out, "Compress History / 会話履歴を圧縮")
	_, _ = fmt.Fprintf(out, "現在の履歴: %d messages\n", len(agent.History))
	_, _ = fmt.Fprintf(out, "保持する最新件数: %d messages\n", keepRecent)
	_, _ = fmt.Fprintf(out, "圧縮対象: %d messages\n", len(agent.History)-keepRecent)
	yellow.Fprintln(out, "\n⚠️  Warning: 圧縮後、古いメッセージはサマリーに置き換わります")

	// 確認
	if !promptConfirmWithRuntime(agent.ui(), "\nContinue? (y/n): ") {
		yellow.Fprintln(out, "Cancelled")
		return true
	}

	// 圧縮実行
	if err := agent.CompressHistory(keepRecent); err != nil {
		red.Fprintf(out, "圧縮に失敗しました: %v\n", err)
		return true
	}

	return true
}

// handleCompactAPICompress は OpenAI Compact API で圧縮
func handleCompactAPICompress(agent *Agent) bool {
	out := agent.output()

	// Compact API 対応チェック
	compactProvider, ok := agent.CurrentProvider.(api.CompactCapable)
	if !ok {
		red.Fprintln(out, "❌ Current provider does not support Compact API")
		yellow.Fprintln(out, "💡 Compact API is only available for OpenAI Responses API models")
		return true
	}

	if !compactProvider.SupportsCompact() {
		red.Fprintln(out, "❌ Current model does not support Compact API")
		return true
	}

	// 確認プロンプト
	printCommandHeaderToWriter(out, "Compress with OpenAI Compact API")
	_, _ = fmt.Fprintf(out, "現在の履歴: %d messages\n", len(agent.History))
	yellow.Fprintln(out, "\n💡 Compact API uses OpenAI's lossy compression")
	yellow.Fprintln(out, "   User messages are preserved verbatim")
	yellow.Fprintln(out, "   Assistant responses are replaced with encrypted data")

	// 確認
	if !promptConfirmWithRuntime(agent.ui(), "\nContinue? (y/n): ") {
		yellow.Fprintln(out, "Cancelled")
		return true
	}

	// Compact API 実行
	ctx := context.Background()
	if err := agent.CompressWithCompactAPI(ctx); err != nil {
		red.Fprintf(out, "❌ Compact API failed: %v\n", err)
		return true
	}

	green.Fprintln(out, "✅ History compressed with Compact API")
	return true
}

// handleTokensCommand はトークン使用量を表示
func handleTokensCommand(agent *Agent) bool {
	cfg := agent.cfg()
	out := agent.output()

	// トークン推定
	totalTokens := agent.EstimateTokens()
	systemTokens := agent.EstimateSystemPromptTokens()
	historyTokens := agent.EstimateHistoryTokens()
	limit := token.GetModelTokenLimit(agent.CurrentModel)
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
	if cfg.Compression.AutoCompress {
		thresholdPercent := cfg.Compression.ThresholdPercent
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

// isCodexModel は Codex モデルかどうかを判定
// Codex モデルは reasoning が必須（"none" 非サポート）
func isCodexModel(model string) bool {
	return strings.Contains(strings.ToLower(model), "codex")
}

// handleThinkCommand は Extended Thinking モードの切り替え
func handleThinkCommand(agent *Agent, args []string) bool {
	cfg := agent.cfg()
	isCodex := agent != nil && isCodexModel(agent.CurrentModel)
	out := agent.output()

	if len(args) == 0 {
		// 現在の状態を表示
		status := "OFF"
		if cfg.Thinking.Enabled {
			status = fmt.Sprintf("ON (level: %s)", cfg.Thinking.Level)
		} else if isCodex {
			status = "low (Codex minimum)"
		}
		_, _ = fmt.Fprintf(out, "🧠 Thinking Mode: %s\n", status)
		return true
	}

	isDeepSeek := agent != nil && strings.ToLower(agent.ProviderName) == "deepseek"

	switch args[0] {
	case "on":
		cfg.Thinking.Enabled = true
		// DeepSeek: モデル名で思考が決まるため reasoner に切り替え
		if isDeepSeek && agent != nil {
			agent.CurrentModel = "deepseek-reasoner"
		}
		green.Fprintf(out, "🧠 Thinking Mode: ON (level: %s)\n", cfg.Thinking.Level)
		if isDeepSeek {
			green.Fprintf(out, "   Model: %s\n", agent.CurrentModel)
		}
	case "off":
		if isCodex {
			// Codexモデルは reasoning 必須のため "low" にフォールバック
			cfg.Thinking.Enabled = false
			cfg.Thinking.Level = "low"
			yellow.Fprintln(out, "⚠️  Codexモデルは reasoning 必須のため low に設定しました")
			green.Fprintln(out, "🧠 Thinking Mode: low (Codex minimum)")
		} else {
			cfg.Thinking.Enabled = false
			// DeepSeek: reasoner → chat にフォールバック
			if isDeepSeek && agent != nil && agent.CurrentModel == "deepseek-reasoner" {
				agent.CurrentModel = "deepseek-chat"
			}
			green.Fprintln(out, "🧠 Thinking Mode: OFF")
			if isDeepSeek {
				green.Fprintf(out, "   Model: %s\n", agent.CurrentModel)
			}
		}
	case "low", "medium", "high", "xhigh":
		cfg.Thinking.Enabled = true
		cfg.Thinking.Level = args[0]
		// DeepSeek: モデル名で思考が決まるため reasoner に切り替え
		if isDeepSeek && agent != nil {
			agent.CurrentModel = "deepseek-reasoner"
		}
		green.Fprintf(out, "🧠 Thinking Mode: ON (level: %s)\n", args[0])
		if isDeepSeek {
			green.Fprintf(out, "   Model: %s\n", agent.CurrentModel)
		}
	default:
		yellow.Fprintln(out, "Usage: /think [on|off|low|medium|high|xhigh]")
	}

	if isDeepSeek && agent != nil {
		agent.syncSessionModel()
	}
	return true
}

// handlePlanCommand は Plan Mode の切り替え
func handlePlanCommand(agent *Agent, args []string) bool {
	out := agent.output()

	if len(args) > 0 {
		switch args[0] {
		case "on":
			agent.PlanModeEnabled = true
			green.Fprintln(out, "✅ Plan Mode ON - 調査→計画→承認→実行")
			return true
		case "off":
			agent.PlanModeEnabled = false
			green.Fprintln(out, "✅ Plan Mode OFF - 通常モード")
			return true
		case "status":
			if agent.PlanModeEnabled {
				cyan.Fprintln(out, "📋 Plan Mode: ON")
				_, _ = fmt.Fprintln(out, "   調査 → 計画 → 承認 → 実行 のフローで処理")
			} else {
				cyan.Fprintln(out, "📋 Plan Mode: OFF")
				_, _ = fmt.Fprintln(out, "   通常モード（ツール個別確認）")
			}
			return true
		}
	}

	// 引数なし：現在のステータスを表示
	if agent.PlanModeEnabled {
		cyan.Fprintln(out, "📋 Plan Mode: ON")
		_, _ = fmt.Fprintln(out, "   調査 → 計画 → 承認 → 実行 のフローで処理")
	} else {
		cyan.Fprintln(out, "📋 Plan Mode: OFF")
		_, _ = fmt.Fprintln(out, "   通常モード（ツール個別確認）")
	}
	return true
}
