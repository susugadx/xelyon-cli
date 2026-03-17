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

// CompressHistory は会話履歴を圧縮する
func (a *Agent) CompressHistory(keepRecent int) error {
	out := a.output()

	if len(a.History) <= keepRecent {
		return fmt.Errorf("履歴が短すぎます（%d件）。圧縮は%d件を超える場合のみ可能です", len(a.History), keepRecent)
	}

	// 圧縮前の統計
	beforeTokens := estimateTokens(a.CurrentModel, a.History)

	// 圧縮対象のメッセージを抽出（FC ターンペアの分断を防止）
	splitIdx := adjustSplitForFCPairs(a.History, len(a.History)-keepRecent)
	toCompress := a.History[:splitIdx]
	toKeep := a.History[splitIdx:]

	if len(toCompress) == 0 {
		return fmt.Errorf("圧縮対象のメッセージがありません（FC ターン保護により分割不可）")
	}

	// サマリー生成プロンプト
	// 古いツール結果を截断してトークン節約（サマリー生成の入力を削減）
	prunedCompress, metrics := CompactOldToolResults(toCompress, DefaultMaxLines, DefaultHeadLines, DefaultTailLines)
	a.addCompactionMetrics(metrics)
	// api.Message を prompt.Message に変換
	promptMessages := make([]prompt.Message, len(prunedCompress))
	for i, m := range prunedCompress {
		promptMessages[i] = prompt.Message{Role: m.Role, Content: m.Content}
	}
	summaryPrompt := prompt.BuildSummaryPrompt(promptMessages, config.MessageTruncateLen)

	// LLMにサマリーを依頼
	cyan.Fprintln(out, "🗜️ Compressing history...")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	compressModel := a.getCompressionModel()
	summary, err := a.CurrentProvider.ChatWithTools(a.requestContext(ctx), "", []api.Message{
		{Role: "user", Content: summaryPrompt},
	}, compressModel)
	if err != nil {
		return fmt.Errorf("サマリー生成に失敗しました: %w", err)
	}

	// 新しい履歴を構築
	newHistory := []api.Message{
		{
			Role:    "system",
			Content: fmt.Sprintf("[Summary of previous conversation]\n\n%s", summary),
		},
	}
	newHistory = append(newHistory, toKeep...)

	// 履歴を置き換え
	a.History = newHistory

	// 圧縮後の統計
	afterTokens := estimateTokens(a.CurrentModel, a.History)

	// 結果表示
	_, _ = fmt.Fprintf(out, "   Before: %s tokens → After: %s tokens\n",
		formatNumber(beforeTokens), formatNumber(afterTokens))
	_, _ = fmt.Fprintln(out)

	return nil
}

// adjustSplitForFCPairs は FC ターン（assistant+ToolCalls → tool レスポンス）のペアが
// 分割点で分断されないように splitIdx を調整する。
// パラレル FC（assistant → tool × N）にも対応: 連続する role:"tool" を巻き戻すループで
// 複数の tool レスポンスを1つの assistant メッセージと一緒に保持する。
func adjustSplitForFCPairs(history []api.Message, splitIdx int) int {
	if splitIdx <= 0 || splitIdx >= len(history) {
		return splitIdx
	}

	// 1. toKeep 側: 先頭が role:"tool" なら、対応する assistant まで巻き戻す
	for splitIdx > 0 && history[splitIdx].Role == "tool" {
		splitIdx--
	}
	// assistant(ToolCalls付き) も toKeep に含める
	if splitIdx > 0 && len(history[splitIdx].ToolCalls) > 0 {
		splitIdx--
	}

	// 2. toCompress 側: 末尾が assistant(ToolCalls付き) なら、
	//    対応する tool レスポンスが toKeep に入ってペアが分断されている。
	//    → assistant も toKeep に移す（splitIdx をさらに1つ前へ）
	if splitIdx > 0 && len(history[splitIdx-1].ToolCalls) > 0 {
		splitIdx--
	}

	if splitIdx <= 0 {
		return 1
	}
	return splitIdx
}

// estimateTokens は会話履歴の概算トークン数を計算する。
func estimateTokens(model string, messages []api.Message) int {
	total := 0
	for _, msg := range messages {
		total += token.EstimateTokenCountForModel(model, msg.Content)
	}
	return total
}
