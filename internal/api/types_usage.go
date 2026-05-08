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
	StorageCost         float64 // トークン料金とは別枠の固定料金（USD、Gemini cache/Kimi web search など）

	// Web search 関連（Kimi built-in $web_search 用、検索結果 tokens は InputTokens に二重加算しない）
	WebSearchCalls        int // built-in web search の呼び出し回数
	WebSearchResultTokens int // provider が返した検索結果 token 観測値
}

// Add は provider から分割して届く usage 観測値を同じ集計単位へ加算する。
func (u *Usage) Add(other Usage) {
	u.InputTokens += other.InputTokens
	u.OutputTokens += other.OutputTokens
	u.ThinkingTokens += other.ThinkingTokens
	u.CachedInputTokens += other.CachedInputTokens
	u.CacheCreationTokens += other.CacheCreationTokens
	u.StorageCost += other.StorageCost
	u.WebSearchCalls += other.WebSearchCalls
	u.WebSearchResultTokens += other.WebSearchResultTokens
}

// HasTokenOrWebSearchObservation は token usage または native web search 観測があるかを返す。
func (u Usage) HasTokenOrWebSearchObservation() bool {
	return u.HasTokenObservation() ||
		u.WebSearchCalls > 0 ||
		u.WebSearchResultTokens > 0
}

// HasTokenObservation は provider endpoint 由来の token/cache usage 観測があるかを返す。
func (u Usage) HasTokenObservation() bool {
	return u.InputTokens > 0 ||
		u.OutputTokens > 0 ||
		u.ThinkingTokens > 0 ||
		u.CachedInputTokens > 0 ||
		u.CacheCreationTokens > 0
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
