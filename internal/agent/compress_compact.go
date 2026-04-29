package agent

import (
	"context"
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/history"
)

const (
	compactAPIPreprocessMaxLines  = 50
	compactAPIPreprocessHeadLines = 20
	compactAPIPreprocessTailLines = 5
)

// CompressWithCompactAPI は OpenAI Compact API で会話履歴を圧縮
func (a *Agent) CompressWithCompactAPI(ctx context.Context) error {
	out := a.output()

	// CompactCapable インターフェースをチェック
	compactProvider, ok := a.CurrentProvider.(api.CompactCapable)
	if !ok {
		return fmt.Errorf("current provider does not support Compact API")
	}

	if !compactProvider.SupportsCompact() {
		return fmt.Errorf("compact API is not supported for this model")
	}

	// 履歴が空の場合はスキップ
	if len(a.History) == 0 {
		return fmt.Errorf("no history to compress")
	}

	// フル会話ウィンドウを構築
	input := a.buildFullInputItems()

	// Compact API 呼び出し
	compactModel := a.getCompressionModel()
	result, err := compactProvider.CompactHistory(a.requestContext(ctx), input, compactModel, a.SystemPrompt)
	if err != nil {
		return fmt.Errorf("compact API failed: %w", err)
	}

	// 圧縮結果を保存
	a.compactedItems = result.Output
	a.isCompactedMode = true

	// セッションにも保存（永続化用）
	if a.session != nil {
		a.session.CompactedItems = convertToHistoryCompactedItems(result.Output)
		a.session.IsCompactedMode = true
	}

	// 元の履歴をクリア（圧縮済みデータに置き換え）
	a.History = nil

	// 統計情報を表示
	if result.Usage != nil {
		yellow.Fprintf(out, "📦 History compacted: %d → %d tokens\n",
			result.Usage.InputTokens,
			result.Usage.OutputTokens)
	} else {
		yellow.Fprintln(out, "📦 History compacted successfully")
	}

	return nil
}

// buildFullInputItems は History から完全な InputItem リストを構築
// Compact API に送るためのフル会話ウィンドウ
func (a *Agent) buildFullInputItems() []api.InputItem {
	// 既に圧縮モードの場合は、圧縮済み state の後ろに現在の履歴を継ぎ足す。
	if a.isCompactedMode && len(a.compactedItems) > 0 {
		input := append([]api.InputItem(nil), a.compactedItems...)
		if len(a.History) == 0 {
			return input
		}
		pruned, metrics := CompactOldToolResults(a.History, compactAPIPreprocessMaxLines, compactAPIPreprocessHeadLines, compactAPIPreprocessTailLines)
		a.addCompactionMetrics(metrics)
		return append(input, api.ConvertHistoryToInputItems(pruned)...)
	}

	// 古いツール結果を截断してから InputItem に変換（トークン節約）
	pruned, metrics := CompactOldToolResults(a.History, compactAPIPreprocessMaxLines, compactAPIPreprocessHeadLines, compactAPIPreprocessTailLines)
	a.addCompactionMetrics(metrics)
	return api.ConvertHistoryToInputItems(pruned)
}

// GetCompactedItems は API 用の圧縮済みアイテムを返す
func (a *Agent) GetCompactedItems() []api.InputItem {
	return a.compactedItems
}

// IsCompactedMode は圧縮モードかどうかを返す
func (a *Agent) IsCompactedMode() bool {
	return a.isCompactedMode
}

// RestoreCompactedState はセッションから圧縮状態を復元
func (a *Agent) RestoreCompactedState(session *history.Session) {
	a.compactedItems = nil
	a.isCompactedMode = false
	if session == nil {
		return
	}

	if session.IsCompactedMode && len(session.CompactedItems) > 0 {
		a.compactedItems = convertFromHistoryCompactedItems(session.CompactedItems)
		a.isCompactedMode = true
	}
}

// convertToHistoryCompactedItems は api.InputItem を history.CompactedItem に変換
func convertToHistoryCompactedItems(items []api.InputItem) []history.CompactedItem {
	result := make([]history.CompactedItem, len(items))
	for i, item := range items {
		result[i] = history.CompactedItem{
			Type:    item.Type,
			Role:    item.Role,
			Content: item.Content,
			ID:      item.ID,
			Status:  item.Status,
			Data:    item.Data,
		}
	}
	return result
}

// convertFromHistoryCompactedItems は history.CompactedItem を api.InputItem に変換
func convertFromHistoryCompactedItems(items []history.CompactedItem) []api.InputItem {
	result := make([]api.InputItem, len(items))
	for i, item := range items {
		result[i] = api.InputItem{
			Type:    item.Type,
			Role:    item.Role,
			Content: item.Content,
			ID:      item.ID,
			Status:  item.Status,
			Data:    item.Data,
		}
	}
	return result
}
