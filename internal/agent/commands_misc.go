package agent

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
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

func buildLastRequestTable(provider, model string, usage *api.Usage) *ui.Table {
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
	if cost > 0 {
		table.AddRow("Cost", fmt.Sprintf("$%.4f USD", cost))
	} else {
		table.AddRow("Cost", "Free (local)")
	}

	return table
}

func getSessionFileInfo(agent *Agent) (string, int64) {
	sessionPath := ""
	sessionSize := int64(0)
	if agent.session != nil {
		sessionPath = fmt.Sprintf("~/.xelyon/sessions/%s.json", agent.session.ID)
		if agent.storage != nil {
			homeDir, err := os.UserHomeDir()
			if err == nil {
				fullPath := fmt.Sprintf("%s/.xelyon/sessions/%s.json", homeDir, agent.session.ID)
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

func buildSessionTokenTable(agent *Agent, stats *SessionStats) *ui.Table {
	if stats.TotalTokens() <= 0 {
		return nil
	}

	tokenTable := ui.NewTable()
	currentTokens := agent.EstimateTokens()
	limit := token.GetModelTokenLimit(agent.CurrentModel)
	if limit > 0 {
		contextPct := float64(currentTokens) / float64(limit) * 100
		tokenTable.AddRow("Context", fmt.Sprintf("%s / %s (%.1f%%)", formatNumber(currentTokens), formatNumber(limit), contextPct))
	}

	tokenTable.AddRow("Input", formatNumber(stats.InputTokens)+" tokens")

	if stats.CachedInputTokens > 0 || stats.CacheCreationTokens > 0 {
		tokenTable.AddRow("Cached", formatNumber(stats.CachedInputTokens)+" tokens").
			AddRow("Cache Creation", formatNumber(stats.CacheCreationTokens)+" tokens").
			AddRow("Hit Rate", fmt.Sprintf("%.1f%%", sessionCacheHitRate(stats)))
	}

	tokenTable.AddRow("Output", formatNumber(stats.OutputTokens)+" tokens")
	if stats.ThinkingTokens > 0 {
		tokenTable.AddRow("Thinking", formatNumber(stats.ThinkingTokens)+" tokens")
	}

	tokenTable.AddRow("Total", formatNumber(stats.TotalTokens())+" tokens")

	cost := stats.EstimatedCost()
	if cost > 0 {
		tokenTable.AddRow("Cost", fmt.Sprintf("$%.4f USD", cost))
	} else {
		tokenTable.AddRow("Cost", "Free (local)")
	}
	return tokenTable
}

func printSessionSections(agent *Agent) {
	if agent.Stats == nil {
		dim.Println("  Statistics not available")
		return
	}

	stats := agent.Stats

	fmt.Println()
	green.Println("📚 Session")
	fmt.Print(buildSessionOverviewTable(agent, stats).RenderCompact())

	fmt.Println()
	green.Println("🔧 Tool Executions")
	if stats.TotalToolExecutions() > 0 {
		toolTable := ui.NewTable()
		for tool, count := range stats.ToolExecutions {
			toolTable.AddRow(tool, fmt.Sprintf("%d", count))
		}
		toolTable.AddRow("Total", fmt.Sprintf("%d", stats.TotalToolExecutions()))
		fmt.Print(toolTable.RenderCompact())
	} else {
		dim.Println("  No tools executed yet")
	}

	fmt.Println()
	green.Println("💰 Session Tokens & Cost")
	if tokenTable := buildSessionTokenTable(agent, stats); tokenTable != nil {
		fmt.Print(tokenTable.RenderCompact())
	} else {
		dim.Println("  No token usage data available")
	}

	fmt.Println()
	green.Println("⚡ Optimizations")
	opt := stats.Optimizations
	if opt.hasAny() {
		optTable := ui.NewTable()
		if opt.DeduplicateCount > 0 {
			optTable.AddRow("Cache-hit dedup", fmt.Sprintf("%d times (~%s tokens saved)", opt.DeduplicateCount, formatNumber(opt.DeduplicateTokensSaved)))
		}
		if opt.NegativeCacheHits > 0 {
			optTable.AddRow("Negative cache", fmt.Sprintf("%d hits", opt.NegativeCacheHits))
		}
		if opt.ErrorCompressions > 0 {
			optTable.AddRow("Error compression", fmt.Sprintf("%d times", opt.ErrorCompressions))
		}
		if opt.FailedPairCompressions > 0 {
			optTable.AddRow("Failed-pair compression", fmt.Sprintf("%d times", opt.FailedPairCompressions))
		}
		if opt.TruncationCount > 0 {
			optTable.AddRow("Graduated truncate", fmt.Sprintf("%d times", opt.TruncationCount))
		}
		if opt.OutlineFirstCount > 0 {
			optTable.AddRow("Outline-first mode", fmt.Sprintf("%d times", opt.OutlineFirstCount))
		}
		if opt.MilestoneDetections > 0 {
			optTable.AddRow("Milestone triggers", fmt.Sprintf("%d times", opt.MilestoneDetections))
		}
		if opt.ToolRatioDetections > 0 {
			optTable.AddRow("Tool-ratio triggers", fmt.Sprintf("%d times", opt.ToolRatioDetections))
		}
		if opt.CompactionCount > 0 {
			optTable.AddRow("Auto-compress", fmt.Sprintf("%d times", opt.CompactionCount))
		}
		fmt.Print(optTable.RenderCompact())
	} else {
		dim.Println("  No optimizations triggered yet")
	}
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
	if len(agent.lastOutputs) == 0 {
		yellow.Println("No AI output to copy yet")
		return true
	}

	// デフォルト: 最後の出力
	outputIndex := len(agent.lastOutputs) - 1
	codeOnly := false

	// 引数解析
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "code":
			codeOnly = true
		case "-n":
			if i+1 < len(args) {
				n, err := strconv.Atoi(args[i+1])
				if err != nil {
					red.Printf("Invalid number: %s\n", args[i+1])
					return true
				}
				if n < 1 || n > len(agent.lastOutputs) {
					red.Printf("Index out of range (1-%d): %d\n", len(agent.lastOutputs), n)
					return true
				}
				outputIndex = len(agent.lastOutputs) - n
				i++ // skip next arg
			} else {
				red.Println("Missing value for -n flag")
				return true
			}
		default:
			yellow.Printf("Unknown argument: %s\n", arg)
			yellow.Println("Usage: /copy [code] [-n <number>]")
			return true
		}
	}

	output := agent.lastOutputs[outputIndex]

	// コードブロックのみ抽出
	if codeOnly {
		codeBlocks := extractCodeBlocks(output)
		if len(codeBlocks) == 0 {
			yellow.Println("No code blocks found in output")
			return true
		}
		output = strings.Join(codeBlocks, "\n\n")
	}

	// クリップボードにコピー
	if err := clipboard.WriteAll(output); err != nil {
		red.Printf("Failed to copy to clipboard: %v\n", err)
		if strings.Contains(err.Error(), "xclip") || strings.Contains(err.Error(), "xsel") {
			yellow.Println("\nLinux requires xclip or xsel:")
			yellow.Println("  Ubuntu/Debian: sudo apt-get install xclip")
			yellow.Println("  Fedora/RHEL:   sudo dnf install xclip")
			yellow.Println("  Arch:          sudo pacman -S xclip")
		}
		return true
	}

	// 成功メッセージ
	lines := strings.Count(output, "\n") + 1
	chars := len(output)
	green.Printf("✅ Copied to clipboard (%d lines, %d chars", lines, chars)
	if codeOnly {
		fmt.Printf(", code blocks only")
	}
	fmt.Println(")")

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
			red.Printf("Invalid number: %s\n", remainingArgs[0])
			yellow.Println("Usage: /compress [keep_recent] [--compact|-c]")
			return true
		}
		if n < 1 {
			red.Println("keep_recent must be at least 1")
			return true
		}
		keepRecent = n
	}

	// 確認プロンプト
	printCommandHeader("Compress History / 会話履歴を圧縮")
	fmt.Printf("現在の履歴: %d messages\n", len(agent.History))
	fmt.Printf("保持する最新件数: %d messages\n", keepRecent)
	fmt.Printf("圧縮対象: %d messages\n", len(agent.History)-keepRecent)
	yellow.Println("\n⚠️  Warning: 圧縮後、古いメッセージはサマリーに置き換わります")

	// 確認
	if !promptConfirm("\nContinue? (y/n): ") {
		yellow.Println("Cancelled")
		return true
	}

	// 圧縮実行
	if err := agent.CompressHistory(keepRecent); err != nil {
		red.Printf("圧縮に失敗しました: %v\n", err)
		return true
	}

	return true
}

// handleCompactAPICompress は OpenAI Compact API で圧縮
func handleCompactAPICompress(agent *Agent) bool {
	// Compact API 対応チェック
	compactProvider, ok := agent.CurrentProvider.(api.CompactCapable)
	if !ok {
		red.Println("❌ Current provider does not support Compact API")
		yellow.Println("💡 Compact API is only available for OpenAI Responses API models")
		return true
	}

	if !compactProvider.SupportsCompact() {
		red.Println("❌ Current model does not support Compact API")
		return true
	}

	// 確認プロンプト
	printCommandHeader("Compress with OpenAI Compact API")
	fmt.Printf("現在の履歴: %d messages\n", len(agent.History))
	yellow.Println("\n💡 Compact API uses OpenAI's lossy compression")
	yellow.Println("   User messages are preserved verbatim")
	yellow.Println("   Assistant responses are replaced with encrypted data")

	// 確認
	if !promptConfirm("\nContinue? (y/n): ") {
		yellow.Println("Cancelled")
		return true
	}

	// Compact API 実行
	ctx := context.Background()
	if err := agent.CompressWithCompactAPI(ctx); err != nil {
		red.Printf("❌ Compact API failed: %v\n", err)
		return true
	}

	green.Println("✅ History compressed with Compact API")
	return true
}

// handleTokensCommand はトークン使用量を表示
func handleTokensCommand(agent *Agent) bool {
	cfg := config.GetGlobalConfig()

	// トークン推定
	totalTokens := agent.EstimateTokens()
	systemTokens := agent.EstimateSystemPromptTokens()
	historyTokens := agent.EstimateHistoryTokens()
	limit := token.GetModelTokenLimit(agent.CurrentModel)
	percentage := float64(totalTokens) / float64(limit) * 100

	// 表示
	printCommandHeader("Token Usage / トークン使用量")
	fmt.Println()

	// 使用量バー表示
	barWidth := 30
	filled := int(percentage / 100 * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	// 色分け
	if percentage > 90 {
		red.Printf("  [%s] %.1f%%\n", bar, percentage)
	} else if percentage > 80 {
		yellow.Printf("  [%s] %.1f%%\n", bar, percentage)
	} else {
		green.Printf("  [%s] %.1f%%\n", bar, percentage)
	}

	fmt.Println()
	fmt.Printf("  Current: %s / %s tokens\n", formatNumber(totalTokens), formatNumber(limit))

	fmt.Println()
	green.Println("📋 Breakdown:")
	fmt.Printf("    System Prompt: %s tokens (%.1f%%)\n",
		formatNumber(systemTokens), float64(systemTokens)/float64(limit)*100)

	// ツール定義トークン
	toolTokens := 0
	if agent.CurrentProvider != nil && agent.CurrentProvider.IsFunctionCallingEnabled() {
		toolTokens = estimateToolDefinitionTokens()
	}
	builtinCount, mcpCount := countToolsByType()
	toolLabel := fmt.Sprintf("%d", builtinCount)
	if mcpCount > 0 {
		toolLabel = fmt.Sprintf("%d+%d MCP", builtinCount, mcpCount)
	}
	fmt.Printf("    Tools (%s):   %s tokens (%.1f%%)\n",
		toolLabel, formatNumber(toolTokens), float64(toolTokens)/float64(limit)*100)

	fmt.Printf("    History:       %s tokens (%.1f%%)  [%d messages]\n",
		formatNumber(historyTokens), float64(historyTokens)/float64(limit)*100, len(agent.History))

	fmt.Println()
	green.Println("🤖 Model:")
	fmt.Printf("    %s (context: %s tokens)\n", agent.CurrentModel, formatNumber(limit))

	fmt.Println()
	green.Println("⚙️  Auto-compress:")
	if cfg.Compression.AutoCompress {
		threshold := cfg.Compression.ThresholdPercent
		if threshold == 0 {
			threshold = 80
		}
		fmt.Printf("    ON (threshold: %d%%)\n", threshold)
	} else {
		fmt.Println("    OFF")
	}

	// 警告
	if percentage > 90 {
		fmt.Println()
		red.Println("⚠️  Token usage is very high! Consider using /compress")
	} else if percentage > 80 {
		fmt.Println()
		yellow.Println("💡 Token usage is high. /compress available if needed")
	}

	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	return true
}

// isCodexModel は Codex モデルかどうかを判定
// Codex モデルは reasoning が必須（"none" 非サポート）
func isCodexModel(model string) bool {
	return strings.Contains(strings.ToLower(model), "codex")
}

// handleThinkCommand は Extended Thinking モードの切り替え
func handleThinkCommand(agent *Agent, args []string) bool {
	cfg := config.GetGlobalConfig()
	isCodex := agent != nil && isCodexModel(agent.CurrentModel)

	if len(args) == 0 {
		// 現在の状態を表示
		status := "OFF"
		if cfg.Thinking.Enabled {
			status = fmt.Sprintf("ON (level: %s)", cfg.Thinking.Level)
		} else if isCodex {
			status = "low (Codex minimum)"
		}
		fmt.Printf("🧠 Thinking Mode: %s\n", status)
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
		green.Printf("🧠 Thinking Mode: ON (level: %s)\n", cfg.Thinking.Level)
		if isDeepSeek {
			green.Printf("   Model: %s\n", agent.CurrentModel)
		}
	case "off":
		if isCodex {
			// Codexモデルは reasoning 必須のため "low" にフォールバック
			cfg.Thinking.Enabled = false
			cfg.Thinking.Level = "low"
			yellow.Println("⚠️  Codexモデルは reasoning 必須のため low に設定しました")
			green.Println("🧠 Thinking Mode: low (Codex minimum)")
		} else {
			cfg.Thinking.Enabled = false
			// DeepSeek: reasoner → chat にフォールバック
			if isDeepSeek && agent != nil && agent.CurrentModel == "deepseek-reasoner" {
				agent.CurrentModel = "deepseek-chat"
			}
			green.Println("🧠 Thinking Mode: OFF")
			if isDeepSeek {
				green.Printf("   Model: %s\n", agent.CurrentModel)
			}
		}
	case "low", "medium", "high", "xhigh":
		cfg.Thinking.Enabled = true
		cfg.Thinking.Level = args[0]
		// DeepSeek: モデル名で思考が決まるため reasoner に切り替え
		if isDeepSeek && agent != nil {
			agent.CurrentModel = "deepseek-reasoner"
		}
		green.Printf("🧠 Thinking Mode: ON (level: %s)\n", args[0])
		if isDeepSeek {
			green.Printf("   Model: %s\n", agent.CurrentModel)
		}
	default:
		yellow.Println("Usage: /think [on|off|low|medium|high|xhigh]")
	}
	return true
}

// handlePlanCommand は Plan Mode の切り替え
func handlePlanCommand(agent *Agent, args []string) bool {
	if len(args) > 0 {
		switch args[0] {
		case "on":
			agent.PlanModeEnabled = true
			green.Println("✅ Plan Mode ON - 調査→計画→承認→実行")
			return true
		case "off":
			agent.PlanModeEnabled = false
			green.Println("✅ Plan Mode OFF - 通常モード")
			return true
		case "status":
			if agent.PlanModeEnabled {
				cyan.Println("📋 Plan Mode: ON")
				fmt.Println("   調査 → 計画 → 承認 → 実行 のフローで処理")
			} else {
				cyan.Println("📋 Plan Mode: OFF")
				fmt.Println("   通常モード（ツール個別確認）")
			}
			return true
		}
	}

	// 引数なし：現在のステータスを表示
	if agent.PlanModeEnabled {
		cyan.Println("📋 Plan Mode: ON")
		fmt.Println("   調査 → 計画 → 承認 → 実行 のフローで処理")
	} else {
		cyan.Println("📋 Plan Mode: OFF")
		fmt.Println("   通常モード（ツール個別確認）")
	}
	return true
}
