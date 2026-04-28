package azure

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

const (
	apiKeyEnv  = "AZURE_OPENAI_API_KEY"
	baseURLEnv = "AZURE_OPENAI_BASE_URL"
)

func init() {
	api.RegisterProvider("azure", func(apiKey string) (api.Provider, error) {
		if strings.TrimSpace(apiKey) == "" {
			return nil, fmt.Errorf("%s not set", apiKeyEnv)
		}
		if strings.TrimSpace(os.Getenv(baseURLEnv)) == "" {
			return nil, fmt.Errorf("%s not set", baseURLEnv)
		}
		return New(apiKey), nil
	})
}

// Provider は Azure OpenAI Responses API の provider 実装。
type Provider struct {
	api.BaseProvider
	lastResponseID string
	mcpTools       []api.ToolDefinition
	usageCallback  api.UsageCallback
	toolChoice     *string
}

// New は Azure OpenAI provider を作成する。
func New(apiKey string) *Provider {
	base := api.NewBaseProvider("Azure OpenAI", apiKey, "", "")
	base.APIURL = normalizeBaseURL(os.Getenv(baseURLEnv))
	return &Provider{BaseProvider: base}
}

// RuntimeProviderName は session / catalog / stats 用の canonical provider identity を返す。
func (p *Provider) RuntimeProviderName() string {
	return "azure"
}

// ProviderConfigKey は provider_models の owner key を返す。
func (p *Provider) ProviderConfigKey() string {
	return "azure"
}

// SupportsImages は画像入力対応を返す。
func (p *Provider) SupportsImages() bool {
	return true
}

// IsFunctionCallingEnabled は Function Calling が有効かを返す。
func (p *Provider) IsFunctionCallingEnabled() bool {
	return os.Getenv("AZURE_OPENAI_FUNCTION_CALLING") != "0"
}

// ChatWithTools は Azure OpenAI Responses API でツール対応チャットを実行する。
func (p *Provider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	model = api.GetDefaultModelWithContext(ctx, model, "azure", "azure-gpt-5.4")
	return p.chatWithResponses(ctx, systemPrompt, history, model)
}

// ChatWithImage は Azure OpenAI Responses API で画像付きメッセージを処理する。
func (p *Provider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	if image == nil || image.Base64 == "" {
		history = append(history, api.Message{Role: "user", Content: userMessage})
		return p.ChatWithTools(ctx, systemPrompt, history, model)
	}

	model = api.GetDefaultModelWithContext(ctx, model, "azure", "azure-gpt-5.4")
	return p.chatWithImageResponses(ctx, systemPrompt, history, userMessage, image, model)
}

// HasCachedResponseID は Responses API のキャッシュ済み response ID があるか返す。
func (p *Provider) HasCachedResponseID() bool {
	return p.lastResponseID != ""
}

// SetResponseID は session 復元用に Responses API response ID を設定する。
func (p *Provider) SetResponseID(id string) {
	p.lastResponseID = strings.TrimSpace(id)
}

// GetResponseID は session 保存用に現在の Responses API response ID を返す。
func (p *Provider) GetResponseID() string {
	return p.lastResponseID
}

// ClearResponseID は Responses API response ID をクリアする。
func (p *Provider) ClearResponseID() {
	p.SetResponseID("")
}

// SetMCPTools は MCP ツール定義を設定する。
func (p *Provider) SetMCPTools(tools []api.ToolDefinition) {
	p.mcpTools = tools
}

// SetUsageCallback はトークン使用量コールバックを設定する。
func (p *Provider) SetUsageCallback(callback api.UsageCallback) {
	p.usageCallback = callback
}

// SetToolChoice は tool_choice を設定する。
func (p *Provider) SetToolChoice(name string) {
	p.toolChoice = &name
}

// ClearToolChoice は tool_choice をクリアする。
func (p *Provider) ClearToolChoice() {
	p.toolChoice = nil
}

// ClearCache は provider が保持する Responses API response ID をクリアする。
func (p *Provider) ClearCache() {
	p.ClearResponseID()
}

func (p *Provider) responsesURL() string {
	return joinEndpoint(p.APIURL, "responses")
}

func azureModelIdentity(ctx context.Context, model string) modelIdentity {
	cfg := config.FromContext(ctx)
	return newModelIdentity(model, cfg.ModelCatalogName("azure", model))
}
