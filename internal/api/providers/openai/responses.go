package openai

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	openaicompat "github.com/susugadx/xelyon-cli/internal/api/providers/openai_compat"
	openairesponses "github.com/susugadx/xelyon-cli/internal/api/providers/openai_responses"
)

const defaultOpenAIResponsesURL = "https://api.openai.com/v1/responses"

// isCodexModel は Codex モデルかどうかを判定
// Codex モデルは reasoning が必須（"none" 非サポート）
func isCodexModel(model string) bool {
	return strings.Contains(strings.ToLower(model), "codex")
}

// convertHistoryToResponsesInput は api.ConvertHistoryToInputItems のラッパー。
func convertHistoryToResponsesInput(history []api.Message) []InputItem {
	return api.ConvertHistoryToInputItems(history)
}

// ReasoningConfig は OpenAI Extended Thinking の設定
type ReasoningConfig = openairesponses.ReasoningConfig

// ResponsesTool は Responses API 用のツール定義を表す。
type ResponsesTool = openairesponses.Tool

// ResponsesRequest は Responses API リクエストを表す。
type ResponsesRequest = openairesponses.Request

// InputItem は api.InputItem のエイリアス。
type InputItem = openairesponses.InputItem

// InputContentPart は api.InputContentPart のエイリアス。
type InputContentPart = openairesponses.InputContentPart

// ResponseMetadata は Responses API のレスポンスメタデータを表す。
type ResponseMetadata = openairesponses.ResponseMetadata

// ResponsesUsage は Responses API の usage 情報を表す。
type ResponsesUsage = openairesponses.Usage

// ResponsesInputDetails は Responses API の入力トークン詳細を表す。
type ResponsesInputDetails = openairesponses.InputDetails

// ResponsesOutputDetails は Responses API の出力トークン詳細を表す。
type ResponsesOutputDetails = openairesponses.OutputDetails

// ResponsesError は Responses API のエラー情報を表す。
type ResponsesError = openairesponses.Error

// ResponsesStreamChunk は Responses API のストリーミングチャンクを表す。
type ResponsesStreamChunk = openairesponses.StreamChunk

// ResponsesItem は Responses API の output item を表す。
type ResponsesItem = openairesponses.Item

// ResponsesResult は Responses API の抽出済み結果を表す。
type ResponsesResult = openairesponses.Result

// chatWithResponses は Responses API でチャット
// previous_response_id を使用してキャッシュを活用
func (p *Provider) chatWithResponses(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	errOut := api.ErrorWriterFromContext(ctx)
	if os.Getenv("XELYON_DEBUG_OPENAI") == "1" {
		fmt.Fprintf(errOut, "[DEBUG OpenAI] chatWithResponses called, model=%s\n", model)
	}

	content, responseID, err := openairesponses.RunStreaming(ctx, openairesponses.StreamingRunOptions{
		URL:          resolveResponsesAPIURL(),
		BuildRequest: func() openairesponses.Request { return p.buildChatResponsesRequest(ctx, systemPrompt, history, model) },
		NewRequest: func(ctx context.Context, url string, payload []byte) (*http.Request, error) {
			return openaicompat.NewBearerJSONBytesRequest(ctx, url, p.APIKey, payload)
		},
		ExecuteRequest: p.ExecuteRequest,
		StreamHandler:  p.handleResponsesStreaming,
		ProviderName:   p.Name(),
		DebugName:      "OpenAI",
		Debug:          os.Getenv("XELYON_DEBUG_OPENAI") == "1",
		DebugWriter:    errOut,
		HasPreviousResponseID: func() bool {
			return p.lastResponseID != ""
		},
		ClearPreviousResponseID: func() {
			p.lastResponseID = ""
		},
	})
	if err == nil && responseID != "" {
		p.lastResponseID = responseID
	}
	return content, err
}

// chatWithImageResponses は Responses API で画像付きメッセージを処理
// NOTE: 画像付きの場合は previous_response_id を使用しない（キャッシュ動作が不明瞭なため）
func (p *Provider) chatWithImageResponses(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	content, responseID, err := openairesponses.RunStreaming(ctx, openairesponses.StreamingRunOptions{
		URL: resolveResponsesAPIURL(),
		BuildRequest: func() openairesponses.Request {
			return p.buildImageResponsesRequest(ctx, systemPrompt, history, userMessage, image, model)
		},
		NewRequest: func(ctx context.Context, url string, payload []byte) (*http.Request, error) {
			return openaicompat.NewBearerJSONBytesRequest(ctx, url, p.APIKey, payload)
		},
		ExecuteRequest: p.ExecuteRequest,
		StreamHandler:  p.handleResponsesStreaming,
		ProviderName:   p.Name(),
		DebugName:      "OpenAI",
		Debug:          os.Getenv("XELYON_DEBUG_OPENAI") == "1",
		DebugWriter:    api.ErrorWriterFromContext(ctx),
	})
	if err == nil && responseID != "" {
		// 画像メッセージ後も responseID を保存（次回テキストのみの場合に使用可能）
		p.lastResponseID = responseID
	}
	return content, err
}
