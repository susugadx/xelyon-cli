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
	beforeMessages := len(a.History)
	beforeTokens := estimateTokens(a.History)

	// 圧縮対象のメッセージを抽出
	toCompress := a.History[:len(a.History)-keepRecent]
	toKeep := a.History[len(a.History)-keepRecent:]

	// サマリー生成プロンプト
	// api.Message を prompt.Message に変換
	promptMessages := make([]prompt.Message, len(toCompress))
	for i, m := range toCompress {
		promptMessages[i] = prompt.Message{Role: m.Role, Content: m.Content}
	}
	summaryPrompt := prompt.BuildSummaryPrompt(promptMessages, config.MessageTruncateLen)

	// LLMにサマリーを依頼
	yellow.Println("🗜️  会話を圧縮中...")
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
	afterMessages := len(a.History)
	afterTokens := estimateTokens(a.History)

	// 削減率計算
	reductionRate := 0.0
	if beforeTokens > 0 {
		reductionRate = float64(beforeTokens-afterTokens) / float64(beforeTokens) * 100.0
	}

	// 結果表示
	fmt.Println()
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Printf("🗜️  Compression Results / 圧縮結果\n")
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Printf("圧縮前: %s tokens (%d messages)\n", formatNumber(beforeTokens), beforeMessages)
	fmt.Printf("圧縮後: %s tokens (%d messages + 1 summary)\n", formatNumber(afterTokens), afterMessages-1)
	fmt.Printf("削減率: %.1f%%\n", reductionRate)
	fmt.Println()
	green.Println("✅ 圧縮完了")
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	return nil
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
