package agent

import (
	"context"
	"fmt"
	"strconv"

	"github.com/susugadx/xelyon-cli/internal/api"
)

const (
	// defaultKeepRecent は /compress コマンドで保持するデフォルトのメッセージ数
	defaultKeepRecent = 10
)

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
	if agent.shouldPrintCompressionOutput() {
		printCommandHeaderToWriter(out, "Compress History / 会話履歴を圧縮")
		_, _ = fmt.Fprintf(out, "現在の履歴: %d messages\n", len(agent.History))
		_, _ = fmt.Fprintf(out, "保持する最新件数: %d messages\n", keepRecent)
		_, _ = fmt.Fprintf(out, "圧縮対象: %d messages\n", len(agent.History)-keepRecent)
		yellow.Fprintln(out, "\n⚠️  Warning: 圧縮後、古いメッセージはサマリーに置き換わります")
	}

	// 確認
	if !promptConfirmWithRuntime(agent.ui(), "\nContinue? (y/n): ") {
		if agent.shouldPrintCompressionOutput() {
			yellow.Fprintln(out, "Cancelled")
		}
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
	if agent.shouldPrintCompressionOutput() {
		printCommandHeaderToWriter(out, "Compress with OpenAI Compact API")
		_, _ = fmt.Fprintf(out, "現在の履歴: %d messages\n", len(agent.History))
		yellow.Fprintln(out, "\n💡 Compact API uses OpenAI's lossy compression")
		yellow.Fprintln(out, "   User messages are preserved verbatim")
		yellow.Fprintln(out, "   Assistant responses are replaced with encrypted data")
	}

	// 確認
	if !promptConfirmWithRuntime(agent.ui(), "\nContinue? (y/n): ") {
		if agent.shouldPrintCompressionOutput() {
			yellow.Fprintln(out, "Cancelled")
		}
		return true
	}

	// Compact API 実行
	ctx := context.Background()
	if err := agent.CompressWithCompactAPI(ctx); err != nil {
		red.Fprintf(out, "❌ Compact API failed: %v\n", err)
		return true
	}

	if agent.shouldPrintCompressionOutput() {
		green.Fprintln(out, "✅ History compressed with Compact API")
	}
	return true
}
