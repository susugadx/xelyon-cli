package agent

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

const (
	// defaultKeepRecent は /compress コマンドで保持するデフォルトのメッセージ数
	defaultKeepRecent = 10
)

// handleStatsCommand はセッション統計を表示
func handleStatsCommand(agent *Agent) bool {
	if agent.Stats == nil {
		yellow.Println("Statistics not available")
		return true
	}

	stats := agent.Stats

	// セッションファイルパスとサイズを取得
	sessionPath := ""
	sessionSize := int64(0)
	if agent.session != nil {
		sessionPath = fmt.Sprintf("~/.xelyon/sessions/%s.json", agent.session.ID)
		if agent.storage != nil {
			// セッションファイルの実際のパスを構築
			homeDir, err := os.UserHomeDir()
			if err == nil {
				fullPath := fmt.Sprintf("%s/.xelyon/sessions/%s.json", homeDir, agent.session.ID)
				if size, err := GetSessionFileSize(fullPath); err == nil {
					sessionSize = size
				}
			}
		}
	}

	// 統計情報を表示
	printCommandHeader("Session Statistics / セッション統計")

	fmt.Println()
	green.Println("⏱️  Time / 経過時間")
	fmt.Printf("  Elapsed: %s\n", stats.FormatElapsedTime())

	fmt.Println()
	green.Println("💬 Messages / メッセージ数")
	fmt.Printf("  User:      %d\n", stats.UserMessages)
	fmt.Printf("  Assistant: %d\n", stats.AssistantMessages)
	fmt.Printf("  Total:     %d\n", stats.TotalMessages())

	fmt.Println()
	green.Println("🔧 Tool Executions / ツール実行回数")
	if stats.TotalToolExecutions() > 0 {
		fmt.Printf("  Total: %d\n", stats.TotalToolExecutions())
		fmt.Println("  Breakdown:")
		for tool, count := range stats.ToolExecutions {
			fmt.Printf("    - %-15s: %d\n", tool, count)
		}
	} else {
		fmt.Println("  No tools executed yet")
	}

	fmt.Println()
	green.Println("🤖 Provider / プロバイダー")
	fmt.Printf("  Name: %s\n", stats.Provider)
	fmt.Printf("  Model: %s\n", agent.CurrentModel)

	fmt.Println()
	green.Println("💰 Token Usage & Cost / トークン使用量とコスト")
	if stats.TotalTokens() > 0 {
		fmt.Printf("  Input:  %s tokens\n", formatNumber(stats.InputTokens))
		fmt.Printf("  Output: %s tokens\n", formatNumber(stats.OutputTokens))
		fmt.Printf("  Total:  %s tokens\n", formatNumber(stats.TotalTokens()))
		cost := stats.EstimatedCost()
		if cost > 0 {
			fmt.Printf("  Estimated Cost: $%.4f USD\n", cost)
		} else {
			fmt.Println("  Cost: Free (local model)")
		}
	} else {
		yellow.Println("  No token usage data available")
		yellow.Println("  (Token tracking requires API support)")
	}

	fmt.Println()
	green.Println("📁 Session File / セッションファイル")
	if sessionPath != "" {
		fmt.Printf("  Path: %s\n", sessionPath)
		if sessionSize > 0 {
			fmt.Printf("  Size: %s\n", FormatFileSize(sessionSize))
		}
	} else {
		yellow.Println("  No session file")
	}

	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	return true
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
	limit := GetModelTokenLimit(agent.CurrentModel)
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

// handleThinkCommand は Extended Thinking モードの切り替え
func handleThinkCommand(agent *Agent, args []string) bool {
	cfg := config.GetGlobalConfig()

	if len(args) == 0 {
		// 現在の状態を表示
		status := "OFF"
		if cfg.Thinking.Enabled {
			status = fmt.Sprintf("ON (level: %s)", cfg.Thinking.Level)
		}
		fmt.Printf("🧠 Thinking Mode: %s\n", status)
		return true
	}

	switch args[0] {
	case "on":
		cfg.Thinking.Enabled = true
		green.Printf("🧠 Thinking Mode: ON (level: %s)\n", cfg.Thinking.Level)
	case "off":
		cfg.Thinking.Enabled = false
		green.Println("🧠 Thinking Mode: OFF")
	case "low", "medium", "high", "xhigh":
		cfg.Thinking.Enabled = true
		cfg.Thinking.Level = args[0]
		green.Printf("🧠 Thinking Mode: ON (level: %s)\n", args[0])
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
