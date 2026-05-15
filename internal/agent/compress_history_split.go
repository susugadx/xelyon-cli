package agent

import "github.com/susugadx/xelyon-cli/internal/api"

type compressionHistorySplit struct {
	toCompress    []api.Message
	toKeep        []api.Message
	toKeepPersist []api.Message
}

func splitHistoryForCompression(history, persistHistory []api.Message, keepRecent int) compressionHistorySplit {
	splitIdx := adjustSplitForFCPairs(history, len(history)-keepRecent)
	return compressionHistorySplit{
		toCompress:    persistHistory[:splitIdx],
		toKeep:        history[splitIdx:],
		toKeepPersist: persistHistory[splitIdx:],
	}
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
		// history[0] が FC ペアの場合、ペアごと toCompress に含める
		if len(history) > 0 && len(history[0].ToolCalls) > 0 {
			i := 1
			for i < len(history) && history[i].Role == "tool" {
				i++
			}
			if i >= len(history) {
				return 0 // 全体が1つの FC ペア → 分割不可
			}
			return i
		}
		return 1
	}
	return splitIdx
}
