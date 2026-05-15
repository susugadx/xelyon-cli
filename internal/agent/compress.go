package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/prompt"
)

const (
	manualCompressMaxLines  = 50
	manualCompressHeadLines = 20
	manualCompressTailLines = 5
)

type compressHistoryOptions struct {
	// Plan Mode の retry など、呼び出し側が restore 前の一時履歴を保存したくない場合だけ true。
	skipPersistenceOnSuccess bool
	displayReason            string
	suppressTUIDisplay       bool
}

// CompressHistory は会話履歴を圧縮する
func (a *Agent) CompressHistory(keepRecent int) error {
	return a.compressHistory(keepRecent, compressHistoryOptions{})
}

func (a *Agent) compressHistory(keepRecent int, opts compressHistoryOptions) error {
	out := a.output()

	if len(a.History) <= keepRecent {
		return fmt.Errorf("履歴が短すぎます（%d件）。圧縮は%d件を超える場合のみ可能です", len(a.History), keepRecent)
	}

	// 圧縮前の統計
	beforeTokens := estimateTokens(a.CurrentModel, a.History)

	persistHistory := a.persistableHistoryForCompression()

	// 圧縮対象のメッセージを抽出（FC ターンペアの分断を防止）
	split := splitHistoryForCompression(a.History, persistHistory, keepRecent)
	toCompress := split.toCompress
	toKeep := split.toKeep
	toKeepPersist := split.toKeepPersist

	if len(toCompress) == 0 {
		return fmt.Errorf("圧縮対象のメッセージがありません（FC ターン保護により分割不可）")
	}

	display := compressionDisplayOperation{}
	if !opts.suppressTUIDisplay {
		display = a.beginCompressionDisplay(
			compressionDisplayModeHistory,
			opts.displayReason,
			keepRecent,
			beforeTokens,
		)
	}

	// サマリー生成プロンプト
	// 古いツール結果を截断してトークン節約（サマリー生成の入力を削減）
	prunedCompress, metrics := CompactOldToolResults(toCompress, manualCompressMaxLines, manualCompressHeadLines, manualCompressTailLines)
	a.addCompactionMetrics(metrics)
	// api.Message を prompt.Message に変換
	promptMessages := make([]prompt.Message, len(prunedCompress))
	for i, m := range prunedCompress {
		promptMessages[i] = prompt.Message{Role: m.Role, Content: m.Content}
	}
	summaryPrompt := prompt.BuildSummaryPrompt(promptMessages, config.MessageTruncateLen)

	// LLMにサマリーを依頼
	if a.shouldPrintCompressionOutput() {
		cyan.Fprintln(out, "🗜️ Compressing history...")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	compressModel := a.getCompressionModel()
	finishResponseContext := a.suspendResponseContinuationForLocalCompression(!opts.skipPersistenceOnSuccess)
	summary, err := a.CurrentProvider.ChatWithTools(a.compressionRequestContext(ctx), "", []api.Message{
		{Role: "user", Content: summaryPrompt},
	}, compressModel)
	if err != nil {
		finishResponseContext(false, nil)
		wrapped := fmt.Errorf("サマリー生成に失敗しました: %w", err)
		a.finishCompressionDisplay(display, 0, wrapped)
		return wrapped
	}

	// 新しい履歴を構築
	summaryMessage := api.Message{
		Role:    "system",
		Content: fmt.Sprintf("[Summary of previous conversation]\n\n%s", summary),
	}
	newHistory := []api.Message{summaryMessage}
	newHistory = append(newHistory, toKeep...)
	persistedHistory := []api.Message{summaryMessage}
	persistedHistory = append(persistedHistory, toKeepPersist...)

	// 履歴を置き換え
	a.History = newHistory
	finishResponseContext(true, persistedHistory)

	// 圧縮後の統計
	afterTokens := estimateTokens(a.CurrentModel, a.History)

	// 結果表示
	if a.shouldPrintCompressionOutput() {
		_, _ = fmt.Fprintf(out, "   Before: %s tokens → After: %s tokens\n",
			formatNumber(beforeTokens), formatNumber(afterTokens))
		_, _ = fmt.Fprintln(out)
	}
	a.finishCompressionDisplay(display, afterTokens, nil)

	return nil
}

func (a *Agent) compressionRequestContext(ctx context.Context) context.Context {
	return api.WithAssistantUpdateMode(a.requestContext(ctx), api.AssistantUpdatesOff)
}

// estimateTokens は会話履歴の概算トークン数を計算する。
func estimateTokens(model string, messages []api.Message) int {
	total := 0
	for _, msg := range messages {
		total += token.EstimateTokenCountForModel(model, msg.Content)
	}
	return total
}
