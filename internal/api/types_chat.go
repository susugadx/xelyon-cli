package api

// このファイルは OpenAI互換 Chat API で使用される共通レスポンス型を定義します。
// request payload は各 provider の owner パッケージで定義します。

// StreamOptions はストリーミング時のオプション
type StreamOptions struct {
	IncludeUsage bool `json:"include_usage"` // trueでusage情報を最終チャンクに含める
}

// Delta はストリームレスポンスの差分
type Delta struct {
	Content          string           `json:"content,omitempty"`
	ReasoningContent string           `json:"reasoning_content,omitempty"` // DeepSeek Reasoner の思考内容
	ToolCalls        []OpenAIToolCall `json:"tool_calls,omitempty"`        // Function Calling用
}

// StreamChoice はストリームの選択肢
type StreamChoice struct {
	Delta        Delta  `json:"delta"`
	FinishReason string `json:"finish_reason,omitempty"` // "stop", "tool_calls" など
}

// PromptTokensDetails はプロンプトトークンの詳細情報（キャッシュ等）
type PromptTokensDetails struct {
	CachedTokens int `json:"cached_tokens,omitempty"` // キャッシュから読み取ったトークン数
}

// CompletionTokensDetails は出力トークンの詳細情報（reasoning 等）。
type CompletionTokensDetails struct {
	ReasoningTokens int `json:"reasoning_tokens,omitempty"`
}

// StreamUsageInfo はストリーミングレスポンスの使用量情報
type StreamUsageInfo struct {
	PromptTokens            int                      `json:"prompt_tokens"`
	CompletionTokens        int                      `json:"completion_tokens"`
	PromptTokensDetails     *PromptTokensDetails     `json:"prompt_tokens_details,omitempty"`     // キャッシュ詳細（Groq, OpenAI等）
	CompletionTokensDetails *CompletionTokensDetails `json:"completion_tokens_details,omitempty"` // reasoning 詳細（OpenAI等）
}

// ToUsage は OpenAI 互換 usage を api.Usage の契約へ正規化する。
func (u StreamUsageInfo) ToUsage() Usage {
	cachedTokens := 0
	if u.PromptTokensDetails != nil {
		cachedTokens = u.PromptTokensDetails.CachedTokens
	}
	reasoningTokens := 0
	if u.CompletionTokensDetails != nil {
		reasoningTokens = u.CompletionTokensDetails.ReasoningTokens
	}
	return UsageFromOutputTokensIncludingThinking(u.PromptTokens, u.CompletionTokens, cachedTokens, reasoningTokens)
}

// StreamResponse はストリームレスポンス
type StreamResponse struct {
	Choices []StreamChoice   `json:"choices"`
	Usage   *StreamUsageInfo `json:"usage,omitempty"` // 最終チャンクに含まれる
}

// Choice は通常レスポンスの選択肢
type Choice struct {
	Message Message `json:"message"`
}

// ChatResponse は通常レスポンス
type ChatResponse struct {
	Choices []Choice `json:"choices"`
}
