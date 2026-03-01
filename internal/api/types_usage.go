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
