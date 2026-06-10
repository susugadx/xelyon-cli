// Package openairesponses は OpenAI Responses API 互換 payload と実行補助を提供する。
package openairesponses

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// ModelIdentity は API に送るモデル名と catalog 判定用モデル名を分離して保持する。
type ModelIdentity struct {
	RequestModel string
	CatalogModel string
}

// NewModelIdentity は request model と catalog model から ModelIdentity を作成する。
func NewModelIdentity(requestModel, catalogModel string) ModelIdentity {
	requestModel = strings.TrimSpace(requestModel)
	catalogModel = strings.TrimSpace(catalogModel)
	if catalogModel == "" {
		catalogModel = requestModel
	}
	return ModelIdentity{
		RequestModel: requestModel,
		CatalogModel: catalogModel,
	}
}

// RequestName は API payload の model に送る名前を返す。
func (m ModelIdentity) RequestName() string {
	return strings.TrimSpace(m.RequestModel)
}

// CatalogName は capability / pricing / token / model-family 判定に使う名前を返す。
func (m ModelIdentity) CatalogName() string {
	catalogModel := strings.TrimSpace(m.CatalogModel)
	if catalogModel != "" {
		return catalogModel
	}
	return m.RequestName()
}

// ReasoningConfig は OpenAI Responses API の reasoning 設定を表す。
type ReasoningConfig struct {
	Effort string `json:"effort"`
}

// Tool は Responses API 用のツール定義を表す。
type Tool struct {
	Type        string                 `json:"type"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Strict      bool                   `json:"strict,omitempty"`
}

// Request は Responses API リクエストを表す。
type Request struct {
	Model                                 string                     `json:"model"`
	Input                                 interface{}                `json:"input,omitempty"`
	PreviousResponseID                    string                     `json:"previous_response_id,omitempty"`
	ContextManagement                     []ContextManagementSetting `json:"context_management,omitempty"`
	Instructions                          string                     `json:"instructions,omitempty"`
	MaxOutputTokens                       int                        `json:"max_output_tokens,omitempty"`
	Stream                                bool                       `json:"stream,omitempty"`
	Store                                 bool                       `json:"store"`
	Reasoning                             *ReasoningConfig           `json:"reasoning,omitempty"`
	Tools                                 []Tool                     `json:"tools,omitempty"`
	ToolChoice                            interface{}                `json:"tool_choice,omitempty"`
	PromptCacheKey                        string                     `json:"prompt_cache_key,omitempty"`
	PromptCacheRetention                  string                     `json:"prompt_cache_retention,omitempty"`
	SkipLocalAutoCompressionAfterResponse bool                       `json:"-"`
}

// ContextManagementSetting は Responses API の context_management 設定項目を表す。
type ContextManagementSetting struct {
	Type             string `json:"type"`
	CompactThreshold int    `json:"compact_threshold,omitempty"`
}

// InputItem は api.InputItem のエイリアス。
type InputItem = api.InputItem

// InputContentPart は api.InputContentPart のエイリアス。
type InputContentPart = api.InputContentPart

// ResponseMetadata は Responses API のレスポンスメタデータを表す。
type ResponseMetadata struct {
	ID     string `json:"id"`
	Status string `json:"status,omitempty"`
	Model  string `json:"model,omitempty"`
	Usage  *Usage `json:"usage,omitempty"`
}

// Usage は Responses API の usage 情報を表す。
type Usage struct {
	InputTokens         int            `json:"input_tokens"`
	OutputTokens        int            `json:"output_tokens"`
	InputTokensDetails  *InputDetails  `json:"input_tokens_details,omitempty"`
	OutputTokensDetails *OutputDetails `json:"output_tokens_details,omitempty"`
}

// InputDetails は Responses API の入力トークン詳細を表す。
type InputDetails struct {
	CachedTokens int `json:"cached_tokens,omitempty"`
}

// OutputDetails は Responses API の出力トークン詳細を表す。
type OutputDetails struct {
	ReasoningTokens int `json:"reasoning_tokens,omitempty"`
}

// Error は Responses API のエラー情報を表す。
type Error struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message,omitempty"`
}

// StreamChunk は Responses API のストリーミングチャンクを表す。
type StreamChunk struct {
	Type        string            `json:"type"`
	Delta       string            `json:"delta,omitempty"`
	OutputIndex *int              `json:"output_index,omitempty"`
	Response    *ResponseMetadata `json:"response,omitempty"`
	Item        *Item             `json:"item,omitempty"`
	Usage       *Usage            `json:"usage,omitempty"`
	Error       *Error            `json:"error,omitempty"`
}

// Item は Responses API の output item を表す。
type Item struct {
	Type             string           `json:"type,omitempty"`
	ID               string           `json:"id,omitempty"`
	Status           string           `json:"status,omitempty"`
	Name             string           `json:"name,omitempty"`
	CallID           string           `json:"call_id,omitempty"`
	Arguments        string           `json:"arguments,omitempty"`
	Output           string           `json:"output,omitempty"`
	Content          interface{}      `json:"content,omitempty"`
	Summary          []map[string]any `json:"summary,omitempty"`
	EncryptedContent string           `json:"encrypted_content,omitempty"`
}

// Result は Responses API の抽出済み結果を表す。
type Result struct {
	Content    string
	ResponseID string
}
