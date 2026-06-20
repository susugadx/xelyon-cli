package agent

import (
	"context"
	"fmt"
	"time"

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
	baseContext              context.Context
	displayReason            string
	suppressTUIDisplay       bool
	onSummaryStart           func()
}

type compressionHistoryPlan struct {
	split                           compressionHistorySplit
	beforeTokens                    int
	displayKeepRecent               int
	compressedCurrentTurnStartIndex int
}

func (p compressionHistoryPlan) hasCompressibleHistory() bool {
	return len(p.split.toCompress) > 0
}

// CompressHistory は会話履歴を圧縮する
func (a *Agent) CompressHistory(keepRecent int) error {
	return a.compressHistory(keepRecent, compressHistoryOptions{})
}

func (a *Agent) compressHistory(keepRecent int, opts compressHistoryOptions) error {
	if len(a.History) <= keepRecent {
		return fmt.Errorf("履歴が短すぎます（%d件）。圧縮は%d件を超える場合のみ可能です", len(a.History), keepRecent)
	}

	return a.compressHistoryWithPlan(a.compressionHistoryPlanForKeepRecent(keepRecent), opts)
}

func (a *Agent) compressionHistoryPlanForKeepRecent(keepRecent int) compressionHistoryPlan {
	beforeTokens := estimateTokens(a.CurrentModel, a.History)
	persistHistory := a.persistableHistoryForCompression()
	return compressionHistoryPlan{
		split:             splitHistoryForCompression(a.History, persistHistory, keepRecent),
		beforeTokens:      beforeTokens,
		displayKeepRecent: keepRecent,
	}
}

func (a *Agent) compressionHistoryPlanForInTurn(currentTurnStartIndex, keepRecent int) compressionHistoryPlan {
	return a.compressionHistoryPlanForInTurnWithPersistHistory(currentTurnStartIndex, keepRecent, nil)
}

func (a *Agent) compressionHistoryPlanForInTurnWithPersistHistory(currentTurnStartIndex, keepRecent int, persistHistory []api.Message) compressionHistoryPlan {
	beforeTokens := estimateTokens(a.CurrentModel, a.History)
	if len(persistHistory) != len(a.History) {
		persistHistory = a.persistableHistoryForCompression()
	}
	if len(persistHistory) != len(a.History) {
		persistHistory = a.History
	}
	split := splitHistoryForInTurnCompression(a.History, persistHistory, currentTurnStartIndex, keepRecent)
	return compressionHistoryPlan{
		split:                           split,
		beforeTokens:                    beforeTokens,
		displayKeepRecent:               keepRecent,
		compressedCurrentTurnStartIndex: compressedCurrentTurnStartIndex(a.History, split, currentTurnStartIndex),
	}
}

func compressedCurrentTurnStartIndex(history []api.Message, split compressionHistorySplit, currentTurnStartIndex int) int {
	if currentTurnStartIndex < 0 {
		currentTurnStartIndex = 0
	}
	if currentTurnStartIndex > len(history) {
		currentTurnStartIndex = len(history)
	}
	currentTurnTailLen := len(history) - currentTurnStartIndex
	keptBeforeCurrentTurn := len(split.toKeep) - currentTurnTailLen
	if keptBeforeCurrentTurn < 0 {
		keptBeforeCurrentTurn = 0
	}
	return 1 + keptBeforeCurrentTurn
}

func (a *Agent) compressHistoryWithPlan(plan compressionHistoryPlan, opts compressHistoryOptions) error {
	return a.compressHistoryWithSplit(
		plan.split.toCompress,
		plan.split.toKeep,
		plan.split.toKeepPersist,
		plan.beforeTokens,
		plan.displayKeepRecent,
		opts,
	)
}

func (a *Agent) compressHistoryWithSplit(
	toCompress []api.Message,
	toKeep []api.Message,
	toKeepPersist []api.Message,
	beforeTokens int,
	displayKeepRecent int,
	opts compressHistoryOptions,
) error {
	out := a.output()

	if len(toCompress) == 0 {
		return fmt.Errorf("圧縮対象のメッセージがありません（FC ターン保護により分割不可）")
	}

	display := compressionDisplayOperation{}
	if !opts.suppressTUIDisplay {
		display = a.beginCompressionDisplay(
			compressionDisplayModeHistory,
			opts.displayReason,
			displayKeepRecent,
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
	baseCtx := opts.baseContext
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	ctx, cancel := context.WithTimeout(baseCtx, 2*time.Minute)
	defer cancel()

	compressModel := a.getCompressionModel()
	finishResponseContext := a.suspendResponseContinuationForLocalCompression(!opts.skipPersistenceOnSuccess)
	if opts.onSummaryStart != nil {
		opts.onSummaryStart()
	}
	summary, err := a.CurrentProvider.ChatWithTools(a.compressionRequestContext(ctx), prompt.BuildSummarySystemPrompt(), []api.Message{
		{Role: "user", Content: summaryPrompt},
	}, compressModel)
	if err != nil {
		finishResponseContext(false, nil)
		wrapped := fmt.Errorf("サマリー生成に失敗しました: %w", err)
		a.finishCompressionDisplay(display, 0, wrapped)
		return wrapped
	}
	continuation, err := prompt.ParseSummaryContinuation(summary)
	if err != nil {
		finishResponseContext(false, nil)
		wrapped := fmt.Errorf("サマリー生成結果が不正です: %w", err)
		a.finishCompressionDisplay(display, 0, wrapped)
		return wrapped
	}

	// 新しい履歴を構築
	summaryMessage := api.Message{
		Role:    "assistant",
		Content: prompt.FormatSummaryContinuationMessage(continuation),
	}
	newHistory := []api.Message{summaryMessage}
	newHistory = append(newHistory, toKeep...)
	persistedHistory := []api.Message{summaryMessage}
	persistedHistory = append(persistedHistory, toKeepPersist...)

	// 履歴を置き換え
	a.History = newHistory
	a.resetProviderFacingTaskLedger()
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
	return api.WithAssistantUpdateMode(a.requestContextWithoutActiveContext(ctx), api.AssistantUpdatesOff)
}
