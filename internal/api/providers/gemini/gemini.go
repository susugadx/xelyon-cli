package gemini

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func init() {
	api.RegisterProvider("gemini", func(apiKey string) (api.Provider, error) {
		if apiKey == "" {
			return nil, fmt.Errorf("GEMINI_API_KEY not set")
		}
		return New(apiKey), nil
	})
}

const defaultGeminiURLTemplate = "https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse"

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
	apiKey        string
	httpClient    *http.Client
	mcpEnabled    bool                 // MCP有効時はテキストモードにフォールバック（レガシー）
	mcpTools      []api.ToolDefinition // MCPツールの定義
	usageCallback api.UsageCallback    // トークン使用量コールバック

	// Context Caching state
	activeCacheName    string    // 現在有効なキャッシュのリソース名
	cachedTokenCount   int       // キャッシュされたトークン数
	cachedMessageCount int       // キャッシュに含まれるメッセージ数
	cacheExpireTime    time.Time // キャッシュの有効期限
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
func (p *Provider) SetMCPTools(tools []api.ToolDefinition) {
	p.mcpTools = tools
}

// SupportsImages は画像入力対応を返す
func (p *Provider) SupportsImages() bool {
	return true
}

// IsFunctionCallingEnabled は Function Calling が有効かを返す
// GEMINI_FUNCTION_CALLING=0 で無効化可能
func (p *Provider) IsFunctionCallingEnabled() bool {
	return os.Getenv("GEMINI_FUNCTION_CALLING") != "0"
}

// SetUsageCallback は使用量レポートのコールバックを設定する
func (p *Provider) SetUsageCallback(callback api.UsageCallback) {
	p.usageCallback = callback
}

// isGemini3Model は Gemini 3 モデルかどうかを判定
func isGemini3Model(model string) bool {
	return strings.Contains(model, "gemini-3")
}

// getThinkingConfigForModel はモデルに応じた ThinkingConfig を返す
// Gemini 3: thinkingLevel（常時ON、デフォルトは Flash="minimal", Pro="low" でlatency最小化）
// Gemini 2.5: thinkingBudget（thinking.enabled=true のときのみ）
func getThinkingConfigForModel(ctx context.Context, model string, cfg *config.Config) *GeminiGenerationConfig {
	maxTokens := api.GetMaxOutputTokens(ctx, "gemini", model)

	if isGemini3Model(model) {
		// Gemini 3: thinking は無効化不可
		// Flash は "minimal" が使える（最も latency が低い）
		// Pro は "low" が最小
		isFlash := strings.Contains(model, "flash")
		var thinkingLevel string
		if api.IsThinkingEnabled(ctx) {
			thinkingLevel = levelToThinkingLevel(cfg.Thinking.Level, model)
		} else {
			if isFlash {
				thinkingLevel = "minimal"
			} else {
				thinkingLevel = "low"
			}
		}
		// Gemini 3 は temperature=1.0 推奨（Google公式）
		// 1.0 以外だとループや性能劣化が発生する
		temp := float32(1.0)
		return &GeminiGenerationConfig{
			ThinkingConfig: &GeminiThinkingConfig{
				ThinkingLevel: thinkingLevel,
			},
			MaxOutputTokens: maxTokens,
			Temperature:     &temp,
		}
	}

	// Gemini 2.5 以前: thinking.enabled=true のときのみ thinkingBudget を送信
	if api.IsThinkingEnabled(ctx) {
		return &GeminiGenerationConfig{
			ThinkingConfig: &GeminiThinkingConfig{
				ThinkingBudget: api.LevelToBudgetTokens(cfg.Thinking.Level),
			},
			MaxOutputTokens: maxTokens,
		}
	}

	return &GeminiGenerationConfig{
		MaxOutputTokens: maxTokens,
	}
}

// levelToThinkingLevel は thinking level を Gemini 3 の thinkingLevel 文字列に変換
// Gemini 3 Pro:   "low", "high" のみ
// Gemini 3 Flash: "minimal", "low", "medium", "high"
func levelToThinkingLevel(level string, model string) string {
	isFlash := strings.Contains(model, "flash")
	switch level {
	case "low":
		return "low"
	case "medium":
		if isFlash {
			return "medium"
		}
		return "low" // Pro は medium 非対応
	case "high", "xhigh":
		return "high"
	default:
		return "low"
	}
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
			// スピナーgoroutineの完全停止とターミナル状態のリセットを保証
			ui.StopGlobalSpinner()
			ui.ResetTerminalState()
			fmt.Fprintf(os.Stderr, "Warning: Function Calling failed, falling back to text mode\n")
			fmt.Fprintf(os.Stderr, "  Reason: %v\n", err)
			if debug {
				fmt.Fprintf(os.Stderr, "[DEBUG Gemini] FC error detail: %+v\n", err)
			}
			return p.chatWithTextMode(ctx, systemPrompt, history, model)
		}
		return result, nil
	}

	if debug {
		fmt.Fprintln(os.Stderr, "[DEBUG Gemini] Mode: TextMode (GEMINI_FUNCTION_CALLING=0)")
	}
	return p.chatWithTextMode(ctx, systemPrompt, history, model)
}

// doRequestWithRetry は HTTP リクエストを実行し、503/429 の場合にリトライ
// 503: 指数バックオフ（1s, 2s, 4s）
// 429: Retry-After ヘッダー優先、なければ指数バックオフ（20s, 40s, 60s）
func (p *Provider) doRequestWithRetry(ctx context.Context, req *http.Request, bodyBytes []byte) (*http.Response, error) {
	const maxRetries = 3

	for attempt := 0; attempt <= maxRetries; attempt++ {
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		resp, err := p.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		if (resp.StatusCode != 503 && resp.StatusCode != 429) || attempt == maxRetries {
			return resp, nil
		}

		var backoff time.Duration
		var reason string
		if resp.StatusCode == 429 {
			reason = "Rate limited (429)"
			backoff = api.RetryAfterDuration(resp, attempt)
		} else {
			reason = "503 Service Unavailable"
			backoff = time.Duration(1<<attempt) * time.Second
		}
		resp.Body.Close()

		msg := fmt.Sprintf("⚠️  %s, retrying (%d/%d) after %v...", reason, attempt+1, maxRetries, backoff)
		spinner := ui.GetGlobalSpinner()
		if spinner != nil && spinner.IsActive() {
			spinner.SetStatus(msg)
		} else {
			fmt.Fprintln(os.Stderr, msg)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
	}

	// ここには到達しないが、コンパイラ対策
	return nil, fmt.Errorf("unexpected: exceeded max retries")
}
