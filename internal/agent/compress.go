package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/prompt"
)

// CompressHistory は会話履歴を圧縮する
func (a *Agent) CompressHistory(keepRecent int) error {
	if len(a.History) <= keepRecent {
		return fmt.Errorf("履歴が短すぎます（%d件）。圧縮は%d件を超える場合のみ可能です", len(a.History), keepRecent)
	}

	// 圧縮前の統計
	beforeTokens := estimateTokens(a.History)

	// 圧縮対象のメッセージを抽出（FC ターンペアの分断を防止）
	splitIdx := adjustSplitForFCPairs(a.History, len(a.History)-keepRecent)
	toCompress := a.History[:splitIdx]
	toKeep := a.History[splitIdx:]

	if len(toCompress) == 0 {
		return fmt.Errorf("圧縮対象のメッセージがありません（FC ターン保護により分割不可）")
	}

	// サマリー生成プロンプト
	// api.Message を prompt.Message に変換
	promptMessages := make([]prompt.Message, len(toCompress))
	for i, m := range toCompress {
		promptMessages[i] = prompt.Message{Role: m.Role, Content: m.Content}
	}
	summaryPrompt := prompt.BuildSummaryPrompt(promptMessages, config.MessageTruncateLen)

	// LLMにサマリーを依頼
	cyan.Println("🗜️ Compressing history...")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	summary, err := a.CurrentProvider.ChatWithTools(ctx, "", []api.Message{
		{Role: "user", Content: summaryPrompt},
	}, a.CurrentModel)
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
	afterTokens := estimateTokens(a.History)

	// 結果表示
	fmt.Printf("   Before: %s tokens → After: %s tokens\n",
		formatNumber(beforeTokens), formatNumber(afterTokens))
	fmt.Println()

	return nil
}

// adjustSplitForFCPairs は FC ターン（assistant+ToolCalls → tool レスポンス）のペアが
// 分割点で分断されないように splitIdx を調整する。
// NOTE: パラレル FC（assistant → tool × N）の場合は完全対応していない。
// 現在の XELYON は write throttle で1ターン1ツールに制限しているため実害なし。
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

// estimateTokens は概算トークン数を計算（英語: 4文字/token、日本語: 2文字/token）
func estimateTokens(messages []api.Message) int {
	totalChars := 0
	for _, msg := range messages {
		totalChars += len(msg.Content)
	}

	// 簡易計算: 平均3文字/token として概算
	return totalChars / 3
}
