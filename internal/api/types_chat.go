package api

// このファイルは OpenAI互換 Chat API で使用される共通型を定義します。
// DeepSeek, Groq, Ollama などの OpenAI互換プロバイダーで共有されます。

// ChatRequest はAPIリクエスト（OpenAI互換形式）
type ChatRequest struct {
	Model           string    `json:"model"`
	Messages        []Message `json:"messages"`
	Stream          bool      `json:"stream"`
	ReasoningEffort string    `json:"reasoning_effort,omitempty"` // OpenAI Extended Thinking用
}

// Delta はストリームレスポンスの差分
type Delta struct {
	Content string `json:"content"`
}

// StreamChoice はストリームの選択肢
type StreamChoice struct {
	Delta Delta `json:"delta"`
}

// StreamResponse はストリームレスポンス
type StreamResponse struct {
	Choices []StreamChoice `json:"choices"`
}

// Choice は通常レスポンスの選択肢
type Choice struct {
	Message Message `json:"message"`
}

// ChatResponse は通常レスポンス
type ChatResponse struct {
	Choices []Choice `json:"choices"`
}
