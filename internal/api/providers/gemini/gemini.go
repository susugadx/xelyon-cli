package gemini

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
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
	runtime       *ui.Runtime          // 補助出力に使う UI runtime

	// Context Caching state（モデル別に管理）
	cacheMap map[string]*cacheEntry // key = model名
}

// defaultResponseHeaderTimeout はHTTPレスポンスヘッダー受信までの最大待機時間
// Google側でリクエスト処理が詰まった場合に無制限にぶら下がるのを防ぐ
const defaultResponseHeaderTimeout = 60 * time.Second

// New は新しいGeminiProviderを作成
func New(apiKey string) *Provider {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = defaultResponseHeaderTimeout
	return &Provider{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout:   config.DefaultHTTPTimeout,
			Transport: transport,
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

// IsFunctionCallingEnabled は Function Calling が有効かを返す。
func (p *Provider) IsFunctionCallingEnabled() bool {
	return true
}

// SetUsageCallback は使用量レポートのコールバックを設定する
func (p *Provider) SetUsageCallback(callback api.UsageCallback) {
	p.usageCallback = callback
}

// SetUIRuntime は provider が補助出力に使う UI runtime を設定する。
func (p *Provider) SetUIRuntime(runtime *ui.Runtime) {
	p.runtime = runtime
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
// Gemini 3 Pro:   "low", "high" のみ (3.1 Pro は "medium" も対応)
// Gemini 3 Flash: "minimal", "low", "medium", "high"
func levelToThinkingLevel(level string, model string) string {
	isFlash := strings.Contains(model, "flash")
	is31Pro := strings.Contains(model, "3.1")
	switch level {
	case "low":
		return "low"
	case "medium":
		if isFlash || is31Pro {
			return "medium"
		}
		return "low" // 3.0 Pro は medium 非対応
	case "high", "xhigh":
		return "high"
	default:
		return "low"
	}
}

// thinkingTimeoutRetryKey はthinking timeoutリトライ回数を追跡するcontext key
const thinkingTimeoutRetryKey ctxKey = "gemini_thinking_timeout_retry"

// maxThinkingTimeoutRetries はthinking timeout時のFCモードリトライ上限
const maxThinkingTimeoutRetries = 2

// idleTimeoutRetryKey はidle timeoutリトライ回数を追跡するcontext key
const idleTimeoutRetryKey ctxKey = "gemini_idle_timeout_retry"

// maxIdleTimeoutRetries はidle timeout時のFCモードリトライ上限
const maxIdleTimeoutRetries = 1

// fcErrorRetryKey はFC一般エラーのリトライ回数を追跡するcontext key
const fcErrorRetryKey ctxKey = "gemini_fc_error_retry"

// maxFCErrorRetries はFC一般エラー時のリトライ上限
const maxFCErrorRetries = 1

// responseStartTimeoutRetryKey はresponse-start timeoutリトライ回数を追跡するcontext key
const responseStartTimeoutRetryKey ctxKey = "gemini_response_start_timeout_retry"

// maxResponseStartTimeoutRetries はresponse-start timeout時のリトライ上限
const maxResponseStartTimeoutRetries = 1

// getThinkingSpinnerMessage はモデルとコンテキストに基づいて thinking スピナーメッセージを返す
// SSEストリーム開始後に "Waiting for Gemini..." から切り替える際に使用
func getThinkingSpinnerMessage(ctx context.Context, model string, isImage bool) string {
	if isImage {
		if isGemini3Model(model) {
			isFlash := strings.Contains(model, "flash")
			if isFlash && !api.IsThinkingEnabled(ctx) {
				return "Analyzing image"
			}
			return "Deep thinking (image)"
		}
		if api.IsThinkingEnabled(ctx) {
			return "Deep thinking (image)"
		}
		return "Analyzing image"
	}
	if isGemini3Model(model) {
		isFlash := strings.Contains(model, "flash")
		if isFlash && !api.IsThinkingEnabled(ctx) {
			return "Thinking"
		}
		return "Deep thinking"
	}
	if api.IsThinkingEnabled(ctx) {
		return "Deep thinking"
	}
	return "Thinking"
}

// isNetworkTimeout はネットワークタイムアウトエラーかどうかを判定する
// ResponseHeaderTimeout によるタイムアウトを検出するために使用
func isNetworkTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

// ChatWithTools は Provider interface の実装（context対応）
// MCPツールもFunction Calling経由で呼び出される
func (p *Provider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	// デバッグモード
	debug := os.Getenv("XELYON_DEBUG_GEMINI") == "1"
	errOut := api.ErrorWriterFromContext(ctx)

	// request mode で Function Calling を制御（デフォルト: 有効）
	useFunctionCalling := api.ShouldSendToolPayload(ctx, p.IsFunctionCallingEnabled())

	if debug {
		fmt.Fprintf(errOut, "[DEBUG Gemini] toolUseDisabled=%v, useFunctionCalling=%v, mcpTools=%d\n",
			api.IsToolUseDisabled(ctx), useFunctionCalling, len(p.mcpTools))
	}

	if useFunctionCalling {
		if debug {
			fmt.Fprintln(errOut, "[DEBUG Gemini] Mode: Function Calling")
		}
		result, err := p.chatWithFunctionCalling(ctx, systemPrompt, history, model)
		if err != nil {
			// Response-start timeout: レスポンスヘッダー受信前にタイムアウト（1回リトライ）
			var responseStartErr *ErrResponseStartTimeout
			if errors.As(err, &responseStartErr) {
				retryCount := 0
				if v := ctx.Value(responseStartTimeoutRetryKey); v != nil {
					retryCount = v.(int)
				}
				if retryCount >= maxResponseStartTimeoutRetries {
					return "", fmt.Errorf("response start timeout: exceeded max retries (%d): %w", maxResponseStartTimeoutRetries, err)
				}
				retryCount++
				api.StopSpinnerAndResetTerminal(ctx)
				fmt.Fprintf(api.ErrorWriterFromContext(ctx), "⚠️ Response start timeout, retrying (%d/%d)...\n", retryCount, maxResponseStartTimeoutRetries)
				ctx = context.WithValue(ctx, responseStartTimeoutRetryKey, retryCount)
				return p.ChatWithTools(ctx, systemPrompt, history, model)
			}

			// Transport idle timeout: FCモードで1回リトライ → それでもダメならエラー
			var idleErr *ErrIdleTimeout
			if errors.As(err, &idleErr) {
				retryCount := 0
				if v := ctx.Value(idleTimeoutRetryKey); v != nil {
					retryCount = v.(int)
				}
				if retryCount >= maxIdleTimeoutRetries {
					return "", fmt.Errorf("transport idle timeout: exceeded max retries (%d): %w", maxIdleTimeoutRetries, err)
				}
				retryCount++
				api.StopSpinnerAndResetTerminal(ctx)
				fmt.Fprintf(api.ErrorWriterFromContext(ctx), "⚠️ Transport idle timeout, retrying FC mode (attempt %d/%d)...\n", retryCount, maxIdleTimeoutRetries)
				ctx = context.WithValue(ctx, idleTimeoutRetryKey, retryCount)
				return p.ChatWithTools(ctx, systemPrompt, history, model)
			}

			// Thinking timeout: FCモードでリトライ（既存ロジック、変更なし）
			var thinkingErr *ErrThinkingTimeout
			if errors.As(err, &thinkingErr) {
				retryCount := 0
				if v := ctx.Value(thinkingTimeoutRetryKey); v != nil {
					retryCount = v.(int)
				}
				if retryCount >= maxThinkingTimeoutRetries {
					return "", fmt.Errorf("thinking timeout: exceeded max retries (%d): %w", maxThinkingTimeoutRetries, err)
				}
				retryCount++
				api.StopSpinnerAndResetTerminal(ctx)
				fmt.Fprintf(api.ErrorWriterFromContext(ctx), "⚠️ Thinking timeout, retrying FC mode (attempt %d/%d)...\n", retryCount, maxThinkingTimeoutRetries)
				ctx = context.WithValue(ctx, thinkingTimeoutRetryKey, retryCount)
				return p.ChatWithTools(ctx, systemPrompt, history, model)
			}

			// その他のFCエラー: FCモードで1回リトライ → それでもダメならエラー
			// テキストモードフォールバックは廃止（ツール定義が消えてキャッシュ汚染するため）
			retryCount := 0
			if v := ctx.Value(fcErrorRetryKey); v != nil {
				retryCount = v.(int)
			}
			if retryCount >= maxFCErrorRetries {
				api.StopSpinnerAndResetTerminal(ctx)
				return "", fmt.Errorf("FC mode failed after %d retries: %w", maxFCErrorRetries, err)
			}
			retryCount++
			api.StopSpinnerAndResetTerminal(ctx)
			fmt.Fprintf(api.ErrorWriterFromContext(ctx), "⚠️ FC error, retrying FC mode (attempt %d/%d)...\n", retryCount, maxFCErrorRetries)
			fmt.Fprintf(api.ErrorWriterFromContext(ctx), "  Reason: %v\n", err)
			if debug {
				fmt.Fprintf(api.ErrorWriterFromContext(ctx), "[DEBUG Gemini] FC error detail: %+v\n", err)
			}
			ctx = context.WithValue(ctx, fcErrorRetryKey, retryCount)
			return p.ChatWithTools(ctx, systemPrompt, history, model)
		}
		return result, nil
	}

	if debug {
		if api.IsToolUseDisabled(ctx) {
			fmt.Fprintln(errOut, "[DEBUG Gemini] Mode: TextMode (tool use disabled for request)")
		}
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
			// ResponseHeaderTimeout によるタイムアウトを検出
			// 親 context のキャンセルではない場合のみ ErrResponseStartTimeout に変換
			if ctx.Err() == nil && isNetworkTimeout(err) {
				return nil, &ErrResponseStartTimeout{
					Message: fmt.Sprintf("response start timeout: no response from Gemini within the timeout period (%v)", err),
				}
			}
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
		spinner := api.SpinnerFromContext(ctx)
		if spinner != nil && spinner.IsActive() {
			spinner.SetStatus(msg)
		} else {
			fmt.Fprintln(api.ErrorWriterFromContext(ctx), msg)
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
