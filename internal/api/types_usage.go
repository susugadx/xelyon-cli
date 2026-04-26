package api

// Usage はAPIレスポンスのトークン使用量
type Usage struct {
	InputTokens  int
	OutputTokens int

	// Thinking 関連（Extended Thinking 対応プロバイダー用）
	ThinkingTokens int // 思考トークン数（出力レートで課金、OutputTokens には含まれない）

	// キャッシュ関連（プロバイダーにより対応状況が異なる）
	CachedInputTokens   int     // キャッシュから読み取ったトークン数（割引対象）
	CacheCreationTokens int     // キャッシュ作成に使用したトークン数（Claude: 1.25x課金）
	StorageCost         float64 // キャッシュストレージ料金（USD、Gemini用）
}

// UsageCallback は usage 受信時に呼ばれるコールバック
type UsageCallback func(usage Usage)

// UsageFromOutputTokensIncludingThinking は、provider の出力合計が thinking/reasoning
// tokens を内包している usage を api.Usage の契約に正規化する。
func UsageFromOutputTokensIncludingThinking(inputTokens, outputTokensIncludingThinking, cachedInputTokens, thinkingTokens int) Usage {
	if thinkingTokens < 0 {
		thinkingTokens = 0
	}
	outputTokens := outputTokensIncludingThinking - thinkingTokens
	if outputTokens < 0 {
		outputTokens = 0
	}
	return Usage{
		InputTokens:       inputTokens,
		OutputTokens:      outputTokens,
		ThinkingTokens:    thinkingTokens,
		CachedInputTokens: cachedInputTokens,
	}
}
