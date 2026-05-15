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
	_, err := a.compressWithCompactAPI(ctx, compressCompactOptions{})
	return err
}

type compressCompactOptions struct {
	displayReason      string
	suppressTUIDisplay bool
}

func (a *Agent) compressWithCompactAPI(ctx context.Context, opts compressCompactOptions) (*api.CompactResponse, error) {
	out := a.output()

	// CompactCapable インターフェースをチェック
	compactProvider, ok := a.CurrentProvider.(api.CompactCapable)
	if !ok {
		return nil, fmt.Errorf("current provider does not support Compact API")
	}

	if !compactProvider.SupportsCompact() {
		return nil, fmt.Errorf("compact API is not supported for this model")
	}

	// 履歴が空の場合はスキップ
	if len(a.History) == 0 {
		return nil, fmt.Errorf("no history to compress")
	}

	beforeTokens := a.EstimateTokens()
	display := compressionDisplayOperation{}
	if !opts.suppressTUIDisplay {
		display = a.beginCompressionDisplay(
			compressionDisplayModeCompactAPI,
			opts.displayReason,
			0,
			beforeTokens,
		)
	}

	// フル会話ウィンドウを構築
	input := a.buildFullInputItems()

	// Compact API 呼び出し
	compactModel := a.getCompressionModel()
	finishResponseContext := a.suspendResponseContinuationForLocalCompression(true)
	result, err := compactProvider.CompactHistory(a.compressionRequestContext(ctx), input, compactModel, a.SystemPrompt)
	if err != nil {
		finishResponseContext(false, nil)
		wrapped := fmt.Errorf("compact API failed: %w", err)
		a.finishCompressionDisplay(display, 0, wrapped)
		return nil, wrapped
	}

	// 圧縮結果を保存
	a.compactedItems = result.Output
	a.isCompactedMode = true

	// 元の履歴をクリア（圧縮済みデータに置き換え）
	a.History = nil
	finishResponseContext(true, nil)

	// 統計情報を表示
	if a.shouldPrintCompressionOutput() {
		if result.Usage != nil {
			yellow.Fprintf(out, "📦 History compacted: %d → %d tokens\n",
				result.Usage.InputTokens,
				result.Usage.OutputTokens)
		} else {
			yellow.Fprintln(out, "📦 History compacted successfully")
		}
	}

	a.finishCompressionDisplay(display, compactResultOutputTokens(result), nil)
	return result, nil
}

// buildFullInputItems は History から完全な InputItem リストを構築
// Compact API に送るためのフル会話ウィンドウ
func (a *Agent) buildFullInputItems() []api.InputItem {
	history := a.persistableHistoryForCompression()

	// 既に圧縮モードの場合は、圧縮済み state の後ろに現在の履歴を継ぎ足す。
	if a.isCompactedMode && len(a.compactedItems) > 0 {
		input := append([]api.InputItem(nil), a.compactedItems...)
		if len(history) == 0 {
			return input
		}
		pruned, metrics := CompactOldToolResults(history, compactAPIPreprocessMaxLines, compactAPIPreprocessHeadLines, compactAPIPreprocessTailLines)
		a.addCompactionMetrics(metrics)
		return append(input, api.ConvertHistoryToInputItems(pruned)...)
	}

	// 古いツール結果を截断してから InputItem に変換（トークン節約）
	pruned, metrics := CompactOldToolResults(history, compactAPIPreprocessMaxLines, compactAPIPreprocessHeadLines, compactAPIPreprocessTailLines)
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
		a.compactedItems = api.CloneInputItems(session.CompactedItems)
		a.isCompactedMode = true
	}
}

// convertToHistoryCompactedItems は api.InputItem を保存用に defensive copy する。
func convertToHistoryCompactedItems(items []api.InputItem) []history.CompactedItem {
	return api.CloneInputItems(items)
}

// convertFromHistoryCompactedItems は保存済み input item を API 用に defensive copy する。
func convertFromHistoryCompactedItems(items []history.CompactedItem) []api.InputItem {
	return api.CloneInputItems(items)
}

func compactResultOutputTokens(result *api.CompactResponse) int {
	if result == nil || result.Usage == nil {
		return 0
	}
	return result.Usage.OutputTokens
}
