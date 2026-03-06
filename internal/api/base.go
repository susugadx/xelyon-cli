package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// BaseProvider は各プロバイダーの共通基盤
type BaseProvider struct {
	ProviderName string
	APIKey       string
	APIURL       string
	HTTPClient   *http.Client
}

// NewBaseProvider は共通のプロバイダー基盤を作成
func NewBaseProvider(name, apiKey, defaultURL, envURLKey string) BaseProvider {
	apiURL := defaultURL
	if envURLKey != "" {
		if envVal := getEnv(envURLKey); envVal != "" {
			apiURL = envVal
		}
	}

	return BaseProvider{
		ProviderName: name,
		APIKey:       apiKey,
		APIURL:       apiURL,
		HTTPClient: &http.Client{
			Timeout: config.DefaultHTTPTimeout,
		},
	}
}

// getEnv は環境変数を取得（os.Getenvのラッパー）
func getEnv(key string) string {
	return os.Getenv(key)
}

// CreateAPIRequest はAPIリクエストを作成
func (b *BaseProvider) CreateAPIRequest(ctx context.Context, body interface{}) (*http.Request, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", b.APIURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

// SetBearerAuth はBearerトークン認証を設定
func (b *BaseProvider) SetBearerAuth(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+b.APIKey)
}

// SetAPIKeyAuth はカスタムヘッダーでAPIキー認証を設定
func (b *BaseProvider) SetAPIKeyAuth(req *http.Request, headerName string) {
	req.Header.Set(headerName, b.APIKey)
}

// ExecuteRequest はHTTPリクエストを実行（429 自動リトライ付き）
func (b *BaseProvider) ExecuteRequest(req *http.Request) (*http.Response, error) {
	return DoWithRetry(req.Context(), b.HTTPClient, req)
}

// Name はプロバイダー名を返す
func (b *BaseProvider) Name() string {
	return b.ProviderName
}

// HandleHTTPError はHTTPエラーレスポンスを処理
func HandleHTTPError(resp *http.Response, spinner *ui.Spinner, providerName string) error {
	if spinner != nil {
		spinner.Stop()
	}

	// レート制限チェック
	if rateLimitErr := HandleRateLimit(resp); rateLimitErr != nil {
		return rateLimitErr
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%s API error (status %d): unable to read response body - %v", providerName, resp.StatusCode, err)
	}

	if len(body) == 0 {
		return fmt.Errorf("%s API error (status %d): empty response body", providerName, resp.StatusCode)
	}

	return SanitizeErrorMessage(body, resp.StatusCode)
}

// HandleNonStreamingResponse は非ストリーミングレスポンスを処理
func HandleNonStreamingResponse(resp *http.Response, spinner *ui.Spinner) (string, error) {
	var result ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		if spinner != nil {
			spinner.Stop()
		}
		return "", err
	}

	if spinner != nil {
		spinner.Stop()
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	content := result.Choices[0].Message.Content
	fmt.Println(content)
	return content, nil
}

// StartSpinner はスピナーを開始
func StartSpinner(message string) *ui.Spinner {
	spinner := ui.NewSpinner()
	spinner.Start(message)
	return spinner
}

// StopSpinner はスピナーを停止（nilセーフ）
func StopSpinner(spinner *ui.Spinner) {
	if spinner != nil {
		spinner.Stop()
	}
}

// GetDefaultModel returns the model to use, checking config first, then falling back.
// providerName must match the config key (e.g., "openai", "claude", "deepseek")
func GetDefaultModel(model, providerName, fallback string) string {
	if model != "" {
		return model
	}
	cfg := config.GetGlobalConfig()
	if pm, ok := cfg.ProviderModels[providerName]; ok && pm.DefaultModel != "" {
		return pm.DefaultModel
	}
	return fallback
}

// StartThinkingSpinner creates and starts a spinner with appropriate message.
// It checks config for thinking mode and adjusts the message accordingly.
// Parameters:
//   - isImage: true for image analysis, false for normal chat
//   - customSuffix: optional suffix like "Function Calling" or "Reasoner"
//   - forceDeep: optional override to show deep-thinking messaging even when config thinking is disabled
//
// Returns the started spinner (caller must call Stop()).
func StartThinkingSpinner(ctx context.Context, isImage bool, customSuffix string, forceDeep ...bool) *ui.Spinner {
	spinner := ui.NewSpinner()
	deep := IsThinkingEnabled(ctx)
	if len(forceDeep) > 0 && forceDeep[0] {
		deep = true
	}

	var msg string
	if isImage {
		if deep {
			msg = "Deep thinking (image)"
		} else {
			msg = "Analyzing image"
		}
	} else {
		if deep {
			msg = "Deep thinking"
		} else {
			msg = "Thinking"
		}
	}

	// Append custom suffix if provided
	if customSuffix != "" {
		msg = msg + " (" + customSuffix + ")"
	}

	spinner.Start(msg)
	ui.SetGlobalSpinner(spinner)
	return spinner
}
