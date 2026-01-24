package gemini

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func init() {
	api.RegisterProvider("gemini", func(apiKey string) (api.Provider, error) {
		if apiKey == "" {
			return nil, fmt.Errorf("GEMINI_API_KEY not set")
		}
		return New(apiKey), nil
	})
}

const defaultGeminiURLTemplate = "https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent"

// getGeminiURL は環境変数またはデフォルトのURLテンプレートを使用してURLを生成
func getGeminiURL(model string) string {
	if baseURL := os.Getenv("GEMINI_API_URL"); baseURL != "" {
		// 環境変数が設定されている場合はそのまま使用（テスト用）
		return baseURL
	}
	return fmt.Sprintf(defaultGeminiURLTemplate, model)
}

// Provider はGemini APIのプロバイダー実装
type Provider struct {
	apiKey     string
	httpClient *http.Client
	mcpEnabled bool                            // MCP有効時はテキストモードにフォールバック（レガシー）
	mcpTools   []api.GeminiFunctionDeclaration // MCPツールの定義
}

// New は新しいGeminiProviderを作成
func New(apiKey string) *Provider {
	return &Provider{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: config.DefaultHTTPTimeout,
		},
	}
}

// Name はプロバイダー名を返す
func (p *Provider) Name() string {
	return "Gemini"
}

// SetMCPEnabled はMCPが有効かどうかを設定する
// レガシー: 現在はMCPツールもFunction Calling経由で呼び出すため、この設定は無視される
// 互換性のために残している
func (p *Provider) SetMCPEnabled(enabled bool) {
	p.mcpEnabled = enabled
}

// SetMCPTools はMCPツールの定義を設定する
// MCPツールはFunction Calling APIで組み込みツールと一緒に送信される
func (p *Provider) SetMCPTools(tools []api.GeminiFunctionDeclaration) {
	p.mcpTools = tools
}

// SupportsImages は画像入力対応を返す
func (p *Provider) SupportsImages() bool {
	return true
}

// ChatWithTools は Provider interface の実装（context対応）
// GEMINI_FUNCTION_CALLING=0の場合のみテキストモードを使用
// MCPツールもFunction Calling経由で呼び出される
func (p *Provider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	// デバッグモード
	debug := os.Getenv("XELYON_DEBUG_GEMINI") == "1"

	// 環境変数でFunction Callingを制御（デフォルト: 有効）
	useFunctionCalling := os.Getenv("GEMINI_FUNCTION_CALLING") != "0"

	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG Gemini] GEMINI_FUNCTION_CALLING=%q, useFunctionCalling=%v, mcpTools=%d\n",
			os.Getenv("GEMINI_FUNCTION_CALLING"), useFunctionCalling, len(p.mcpTools))
	}

	if useFunctionCalling {
		if debug {
			fmt.Fprintln(os.Stderr, "[DEBUG Gemini] Mode: Function Calling")
		}
		result, err := p.chatWithFunctionCalling(ctx, systemPrompt, history, model)
		if err != nil {
			// Function Calling失敗時はテキストモードにフォールバック
			fmt.Printf("Warning: Function Calling failed, falling back to text mode: %v\n", err)
			return p.chatWithTextMode(ctx, systemPrompt, history, model)
		}
		return result, nil
	}

	if debug {
		fmt.Fprintln(os.Stderr, "[DEBUG Gemini] Mode: TextMode (GEMINI_FUNCTION_CALLING=0)")
	}
	return p.chatWithTextMode(ctx, systemPrompt, history, model)
}
