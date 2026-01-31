package api

// Usage はAPIレスポンスのトークン使用量
type Usage struct {
	InputTokens  int
	OutputTokens int
}

// UsageCallback は usage 受信時に呼ばれるコールバック
type UsageCallback func(usage Usage)
