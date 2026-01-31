package api

// このファイルは OpenAI互換 Chat API で使用される共通型を定義します。
// DeepSeek, Groq, Ollama などの OpenAI互換プロバイダーで共有されます。

// StreamOptions はストリーミング時のオプション
type StreamOptions struct {
	IncludeUsage bool `json:"include_usage"` // trueでusage情報を最終チャンクに含める
}

// ChatRequest はAPIリクエスト（OpenAI互換形式）
type ChatRequest struct {
	Model           string         `json:"model"`
	Messages        []Message      `json:"messages"`
	Stream          bool           `json:"stream"`
	StreamOptions   *StreamOptions `json:"stream_options,omitempty"`   // ストリーミング時のオプション
	ReasoningEffort string         `json:"reasoning_effort,omitempty"` // OpenAI Extended Thinking用
	Tools           []OpenAITool   `json:"tools,omitempty"`            // Function Calling用
	ToolChoice      string         `json:"tool_choice,omitempty"`      // "auto", "none", "required"
}

// Delta はストリームレスポンスの差分
type Delta struct {
	Content   string           `json:"content,omitempty"`
	ToolCalls []OpenAIToolCall `json:"tool_calls,omitempty"` // Function Calling用
}

// StreamChoice はストリームの選択肢
type StreamChoice struct {
	Delta        Delta  `json:"delta"`
	FinishReason string `json:"finish_reason,omitempty"` // "stop", "tool_calls" など
}

// StreamUsageInfo はストリーミングレスポンスの使用量情報
type StreamUsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
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
