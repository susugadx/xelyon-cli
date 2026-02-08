package gemini

import "github.com/susugadx/xelyon-cli/internal/api"

// ===== Basic Gemini API structures =====

// GeminiSystemInstruction は system_instruction フィールド用構造体
// user/model ペアではなく専用フィールドで送信することで SP 遵守率を向上させる
type GeminiSystemInstruction struct {
	Parts []GeminiPart `json:"parts"`
}

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
	SystemInstruction *GeminiSystemInstruction `json:"system_instruction,omitempty"`
	Contents          []interface{}            `json:"contents"` // GeminiContent or GeminiMultimodalContent
	Tools             []api.GeminiToolConfig   `json:"tools,omitempty"`
	ToolConfig        *GeminiToolConfigWrapper `json:"tool_config,omitempty"`
	GenerationConfig  *GeminiGenerationConfig  `json:"generationConfig,omitempty"`
}

// GeminiContent はGeminiの contents 構造
type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"` // "user" or "model"
}

// GeminiRequest はGemini APIリクエスト
type GeminiRequest struct {
	SystemInstruction *GeminiSystemInstruction `json:"system_instruction,omitempty"`
	Contents          []GeminiContent          `json:"contents"`
	GenerationConfig  *GeminiGenerationConfig  `json:"generationConfig,omitempty"`
}

// GeminiCandidate はレスポンスの候補
type GeminiCandidate struct {
	Content GeminiContent `json:"content"`
}

// GeminiUsageMetadata はトークン使用量
type GeminiUsageMetadata struct {
	PromptTokenCount        int `json:"promptTokenCount"`
	CandidatesTokenCount    int `json:"candidatesTokenCount"`
	CachedContentTokenCount int `json:"cachedContentTokenCount,omitempty"`
}

// GeminiResponse はGeminiレスポンス
type GeminiResponse struct {
	Candidates    []GeminiCandidate    `json:"candidates"`
	UsageMetadata *GeminiUsageMetadata `json:"usageMetadata,omitempty"`
}

// ===== Function Calling API structures =====

// GeminiFunctionPart はtext または functionCall を含むパート
type GeminiFunctionPart struct {
	Text             string                  `json:"text,omitempty"`
	FunctionCall     *api.GeminiFunctionCall `json:"functionCall,omitempty"`
	ThoughtSignature string                  `json:"thoughtSignature,omitempty"` // Gemini 3: 暗号化された思考プロセス
	Thought          bool                    `json:"thought,omitempty"`          // Gemini 3: thinking summary フラグ
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
	Candidates    []GeminiFunctionCandidate `json:"candidates"`
	UsageMetadata *GeminiUsageMetadata      `json:"usageMetadata,omitempty"`
}

// GeminiThinkingConfig は Extended Thinking の設定
// Gemini 2.5: thinkingBudget (int) を使用
// Gemini 3:   thinkingLevel (string: "low", "high") を使用
// 両方同時に指定すると 400 エラーになるため、モデルに応じてどちらか一方のみ設定すること
type GeminiThinkingConfig struct {
	ThinkingBudget int    `json:"thinkingBudget,omitempty"` // Gemini 2.5 用
	ThinkingLevel  string `json:"thinkingLevel,omitempty"`  // Gemini 3 用: "low", "medium"(Flash), "high"
}

// GeminiGenerationConfig は生成設定
type GeminiGenerationConfig struct {
	ThinkingConfig  *GeminiThinkingConfig `json:"thinkingConfig,omitempty"`
	MaxOutputTokens int                   `json:"maxOutputTokens,omitempty"` // 最大出力トークン数
	Temperature     *float32              `json:"temperature,omitempty"`     // Gemini 3: 1.0 推奨
}

// GeminiFunctionCallingConfig は Function Calling の動作モード設定
type GeminiFunctionCallingConfig struct {
	Mode string `json:"mode"` // "AUTO", "ANY", "NONE"
}

// GeminiToolConfigWrapper は tool_config フィールド用ラッパー
type GeminiToolConfigWrapper struct {
	FunctionCallingConfig GeminiFunctionCallingConfig `json:"function_calling_config"`
}

// ===== Function Calling history structures =====

// GeminiFunctionCallPart は functionCall を含む parts 要素
type GeminiFunctionCallPart struct {
	FunctionCall GeminiFunctionCallData `json:"functionCall"`
}

// GeminiFunctionCallData は functionCall のデータ
type GeminiFunctionCallData struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

// GeminiFunctionResponsePart は functionResponse を含む parts 要素
type GeminiFunctionResponsePart struct {
	FunctionResponse GeminiFunctionResponseData `json:"functionResponse"`
}

// GeminiFunctionResponseData は functionResponse のデータ
type GeminiFunctionResponseData struct {
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
}

// GeminiGenericContent は任意の parts を含む contents 要素
// functionCall, functionResponse, text を混在させるため interface{} を使用
type GeminiGenericContent struct {
	Parts []interface{} `json:"parts"`
	Role  string        `json:"role,omitempty"`
}

// GeminiRequestWithTools はtools を含むリクエスト
type GeminiRequestWithTools struct {
	SystemInstruction *GeminiSystemInstruction `json:"system_instruction,omitempty"`
	Contents          []interface{}            `json:"contents"`
	Tools             []api.GeminiToolConfig   `json:"tools,omitempty"`
	ToolConfig        *GeminiToolConfigWrapper `json:"tool_config,omitempty"`
	GenerationConfig  *GeminiGenerationConfig  `json:"generationConfig,omitempty"`
}
