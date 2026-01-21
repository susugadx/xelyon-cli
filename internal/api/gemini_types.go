package api

// ===== Basic Gemini API structures =====

// GeminiPart はGeminiの parts 構造（テキストのみ）
type GeminiPart struct {
	Text string `json:"text"`
}

// GeminiMultimodalPart はマルチモーダル対応のparts構造
type GeminiMultimodalPart struct {
	Text       string            `json:"text,omitempty"`
	InlineData *GeminiInlineData `json:"inline_data,omitempty"`
}

// GeminiInlineData は画像データ
type GeminiInlineData struct {
	MimeType string `json:"mime_type"` // "image/png", "image/jpeg" etc
	Data     string `json:"data"`      // Base64エンコードされたデータ
}

// GeminiMultimodalContent はマルチモーダル対応のcontents構造
type GeminiMultimodalContent struct {
	Parts []GeminiMultimodalPart `json:"parts"`
	Role  string                 `json:"role,omitempty"` // "user" or "model"
}

// GeminiMultimodalRequest はマルチモーダルAPIリクエスト
type GeminiMultimodalRequest struct {
	Contents []interface{} `json:"contents"` // GeminiContent or GeminiMultimodalContent
}

// GeminiContent はGeminiの contents 構造
type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"` // "user" or "model"
}

// GeminiRequest はGemini APIリクエスト
type GeminiRequest struct {
	Contents []GeminiContent `json:"contents"`
}

// GeminiCandidate はレスポンスの候補
type GeminiCandidate struct {
	Content GeminiContent `json:"content"`
}

// GeminiResponse はGeminiレスポンス
type GeminiResponse struct {
	Candidates []GeminiCandidate `json:"candidates"`
}

// ===== Function Calling API structures =====

// GeminiFunctionPart はtext または functionCall を含むパート
type GeminiFunctionPart struct {
	Text         string              `json:"text,omitempty"`
	FunctionCall *GeminiFunctionCall `json:"functionCall,omitempty"`
}

// GeminiFunctionContent はFunction Calling対応のコンテンツ
type GeminiFunctionContent struct {
	Parts []GeminiFunctionPart `json:"parts"`
	Role  string               `json:"role,omitempty"`
}

// GeminiFunctionCandidate はFunction Calling対応の候補
type GeminiFunctionCandidate struct {
	Content GeminiFunctionContent `json:"content"`
}

// GeminiFunctionResponse はFunction Calling対応のレスポンス
type GeminiFunctionResponse struct {
	Candidates []GeminiFunctionCandidate `json:"candidates"`
}

// GeminiThinkingConfig は Extended Thinking の設定
type GeminiThinkingConfig struct {
	ThinkingBudget int `json:"thinkingBudget,omitempty"`
}

// GeminiGenerationConfig は生成設定
type GeminiGenerationConfig struct {
	ThinkingConfig *GeminiThinkingConfig `json:"thinkingConfig,omitempty"`
}

// GeminiRequestWithTools はtools を含むリクエスト
type GeminiRequestWithTools struct {
	Contents         []interface{}           `json:"contents"`
	Tools            []GeminiToolConfig      `json:"tools,omitempty"`
	GenerationConfig *GeminiGenerationConfig `json:"generationConfig,omitempty"`
}
