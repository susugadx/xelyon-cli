package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// ProviderFactory はプロバイダーを生成するファクトリ関数
type ProviderFactory func(apiKey string) (Provider, error)

var (
	providerRegistry   = make(map[string]ProviderFactory)
	providerRegistryMu sync.RWMutex
)

// RegisterProvider はプロバイダーを登録する（init()から呼ばれる）
func RegisterProvider(name string, factory ProviderFactory) {
	providerRegistryMu.Lock()
	defer providerRegistryMu.Unlock()
	providerRegistry[strings.ToLower(name)] = factory
}

// getRegisteredProvider は登録済みプロバイダーを取得
func getRegisteredProvider(name string) (ProviderFactory, bool) {
	providerRegistryMu.RLock()
	defer providerRegistryMu.RUnlock()
	factory, ok := providerRegistry[strings.ToLower(name)]
	return factory, ok
}

// IsRegisteredProvider は指定名がレジストリに登録済みか返す
func IsRegisteredProvider(name string) bool {
	_, ok := getRegisteredProvider(name)
	return ok
}

// ListProviders は登録済みプロバイダー名をソート済みで返す（エイリアスを除く）
func ListProviders() []string {
	providerRegistryMu.RLock()
	defer providerRegistryMu.RUnlock()

	// エイリアス（anthropic → claude）を除外
	aliases := map[string]bool{"anthropic": true}

	var names []string
	for name := range providerRegistry {
		if !aliases[name] {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// getAPIKeyForProvider はプロバイダー名から環境変数のAPIキーを取得
func getAPIKeyForProvider(providerName string) string {
	entry, ok := llmcatalog.ProviderDescriptorFor(providerName)
	if !ok {
		return ""
	}
	switch entry.CredentialKind {
	case "base_url":
		if entry.BaseURLEnv != "" {
			if value := os.Getenv(entry.BaseURLEnv); value != "" {
				return value
			}
		}
		return entry.DefaultBaseURL
	case "static":
		return entry.StaticCredential
	case "api_key", "":
		if entry.APIKeyEnv == "" {
			return ""
		}
		return os.Getenv(entry.APIKeyEnv)
	default:
		return ""
	}
}

// Provider はLLMプロバイダーの共通インターフェース
type Provider interface {
	// Name はプロバイダー名を返す
	Name() string

	// ChatWithTools はツール対応の会話を行う（ストリーミング）
	ChatWithTools(ctx context.Context, systemPrompt string, history []Message, model string) (string, error)

	// SupportsImages はプロバイダーが画像入力に対応しているかを返す
	SupportsImages() bool

	// ChatWithImage は画像付きメッセージで会話を行う
	// imageがnilまたはプロバイダーが画像非対応の場合、テキストのみで会話する
	ChatWithImage(ctx context.Context, systemPrompt string, history []Message, userMessage string, image *ImageData, model string) (string, error)

	// IsFunctionCallingEnabled は Function Calling が有効かを返す
	// true の場合、System Prompt からツール定義を削除して重複を避ける
	IsFunctionCallingEnabled() bool
}

// CompactCapable は圧縮対応プロバイダーのオプショナルインターフェース
// 現時点では OpenAIProvider のみが実装
type CompactCapable interface {
	// CompactHistory は会話履歴を圧縮する
	CompactHistory(ctx context.Context, input []InputItem, model, instructions string) (*CompactResponse, error)

	// SupportsCompact は Compact API 対応を返す
	SupportsCompact() bool
}

// ClaudeCompactionCapable は Claude Compaction API 対応プロバイダーのオプショナルインターフェース
type ClaudeCompactionCapable interface {
	// SupportsClaudeCompaction は Claude Compaction 対応を返す
	SupportsClaudeCompaction() bool
}

// ClaudeCompactionRuntimeCapable は runtime/config 注入付きで Claude Compaction 判定を行えるプロバイダー。
type ClaudeCompactionRuntimeCapable interface {
	// SupportsClaudeCompactionWithContext は request context とモデル名を使って Claude Compaction 対応可否を返す。
	SupportsClaudeCompactionWithContext(ctx context.Context, model string) bool
}

// ModelLister はモデル一覧取得に対応するプロバイダーのオプショナルインターフェース
// 現時点では OllamaProvider のみが実装
type ModelLister interface {
	// ListModels はインストール済み/利用可能なモデルの一覧を取得
	ListModels() ([]string, error)
}

// MCPProvider はMCPツール設定に対応するプロバイダーのオプショナルインターフェース
type MCPProvider interface {
	// SetMCPEnabled はMCPが有効かどうかを設定する（レガシー、互換性のため）
	SetMCPEnabled(enabled bool)

	// SetMCPTools はMCPツールの定義を設定する
	SetMCPTools(tools []ToolDefinition)
}

// RuntimeConfigurable は provider に runtime 単位の設定を注入できるオプショナルインターフェース。
type RuntimeConfigurable interface {
	// SetRuntimeConfig は provider が参照する runtime 設定を差し替える。
	SetRuntimeConfig(cfg *config.Config)
}

// UIRuntimeConfigurable は provider に UI runtime を注入できるオプショナルインターフェース。
type UIRuntimeConfigurable interface {
	// SetUIRuntime は provider が補助出力に使う UI runtime を差し替える。
	SetUIRuntime(runtime *ui.Runtime)
}

// UsageReporter はトークン使用量レポートに対応するプロバイダーのオプショナルインターフェース
type UsageReporter interface {
	// SetUsageCallback は使用量レポートのコールバックを設定する
	SetUsageCallback(callback UsageCallback)
}

// ReasoningContentProvider は reasoning_content（思考内容）に対応するプロバイダーのオプショナルインターフェース
// DeepSeek Reasoner などの思考モデルで使用
type ReasoningContentProvider interface {
	// LastReasoningContent は最後の API 呼び出しで返された reasoning_content を返す
	// ChatWithTools 呼び出し後に取得し、assistant メッセージに含めて次のリクエストに送る
	LastReasoningContent() string
}

// CacheClearable はモデル/プロバイダー切り替え時にキャッシュをクリア可能なプロバイダーのオプショナルインターフェース
type CacheClearable interface {
	// ClearCache はプロバイダーが保持するキャッシュ（リモート/ローカル）をクリアする
	ClearCache()
}

// GetMaxOutputTokens は指定されたプロバイダーとモデルの最大出力トークン数を取得する。
// 優先順位: model_overrides > catalog の既知モデル値 > provider default > Thinking 加算。
func GetMaxOutputTokens(ctx context.Context, providerName, model string) int {
	cfg := config.FromContext(ctx)

	maxTokens := 0
	pName := config.NormalizeProviderName(providerName)
	lookupProvider := cfg.RuntimeProviderConfigKey(providerName, model)
	pCfg, _ := cfg.GetProviderModelConfig(lookupProvider)

	// 1. ユーザーの model_overrides
	if pCfg.ModelOverrides != nil {
		if override, ok := pCfg.ModelOverrides[model]; ok && override.MaxOutputTokens > 0 {
			maxTokens = override.MaxOutputTokens
		}
	}

	// 2. catalog の既知モデル値
	if maxTokens == 0 {
		if tokens, ok := llmcatalog.KnownMaxOutputTokens(model); ok {
			maxTokens = tokens
		}
	}

	// 3. プロバイダーのデフォルト値
	if maxTokens == 0 {
		maxTokens = pCfg.MaxOutputTokens
	}

	// 4. Extended Thinking 有効時は BudgetTokens を考慮
	// adaptive thinking モデル（Claude 4.6）は API が自動管理するため加算不要
	// それ以外は max_tokens = budget_tokens + output_tokens
	if IsThinkingEnabled(ctx) && (pName == "claude" || pName == "anthropic" || pName == "bedrock") {
		if !isAdaptiveThinkingModel(model) {
			budget := LevelToBudgetTokens(cfg.Thinking.Level)
			return budget + maxTokens
		}
	}

	return maxTokens
}

// SupportsImages はプロバイダー名から画像対応を判定
func SupportsImages(providerName string) bool {
	return config.ProviderSupportsImages(providerName)
}

// ApplyRuntimeConfig は provider が RuntimeConfigurable の場合に runtime 設定を注入する。
func ApplyRuntimeConfig(provider Provider, cfg *config.Config) {
	if provider == nil || cfg == nil {
		return
	}
	if configurable, ok := provider.(RuntimeConfigurable); ok {
		configurable.SetRuntimeConfig(cfg)
	}
}

// ApplyUIRuntime は provider が UIRuntimeConfigurable の場合に UI runtime を注入する。
func ApplyUIRuntime(provider Provider, runtime *ui.Runtime) {
	if provider == nil || runtime == nil {
		return
	}
	if configurable, ok := provider.(UIRuntimeConfigurable); ok {
		configurable.SetUIRuntime(runtime)
	}
}

// SanitizeErrorMessage はエラーメッセージから機密情報を削除
func SanitizeErrorMessage(body []byte, statusCode int) error {
	const maxLen = 200 // エラーメッセージの最大長

	if len(body) == 0 {
		return fmt.Errorf("API error (%d): empty response", statusCode)
	}

	message := string(body)

	// APIキーのパターンを削除（sk-, Bearer, api_key= など）
	apiKeyPatterns := []*regexp.Regexp{
		regexp.MustCompile(`sk-[a-zA-Z0-9]{20,}`),                 // OpenAI形式
		regexp.MustCompile(`Bearer [a-zA-Z0-9_\-\.]{20,}`),        // Bearer token
		regexp.MustCompile(`api_key[=:]\s*[a-zA-Z0-9_\-\.]{20,}`), // api_key=
		regexp.MustCompile(`"key":\s*"[a-zA-Z0-9_\-\.]{20,}"`),    // JSON key
		regexp.MustCompile(`AIza[a-zA-Z0-9_\-]{30,}`),             // Google API key
		regexp.MustCompile(`AKIA[A-Z0-9]{16}`),                    // AWS key
	}

	for _, pattern := range apiKeyPatterns {
		message = pattern.ReplaceAllString(message, "[REDACTED]")
	}

	// 長すぎる場合は切り詰め
	if len(message) > maxLen {
		message = message[:maxLen] + "... (truncated)"
	}

	return fmt.Errorf("API error (%d): %s", statusCode, message)
}

// HandleRateLimit はレート制限エラーを処理
func HandleRateLimit(resp *http.Response) error {
	if resp.StatusCode != 429 {
		return nil // レート制限エラーではない
	}

	// Retry-Afterヘッダーをチェック
	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter == "" {
		// ヘッダーがない場合はデフォルト待機時間
		return fmt.Errorf("rate limit exceeded (429). Please retry after 60 seconds")
	}

	// Retry-Afterは秒数またはHTTP-date形式
	// まず秒数として解釈を試みる
	if seconds, err := strconv.Atoi(retryAfter); err == nil {
		return fmt.Errorf("rate limit exceeded (429). Please retry after %d seconds", seconds)
	}

	// HTTP-date形式の場合
	if retryTime, err := http.ParseTime(retryAfter); err == nil {
		waitDuration := time.Until(retryTime)
		if waitDuration > 0 {
			return fmt.Errorf("rate limit exceeded (429). Please retry after %v", waitDuration.Round(time.Second))
		}
	}

	return fmt.Errorf("rate limit exceeded (429). Please retry later")
}

// isAdaptiveThinkingModel は adaptive thinking を使用するモデルか判定する。
// claude パッケージと循環依存を避けるため api パッケージにも定義。
func isAdaptiveThinkingModel(model string) bool {
	return llmcatalog.IsAdaptiveClaudeThinkingModel(model)
}

// LevelToBudgetTokens は Thinking Level を budget_tokens に変換（Claude/Gemini共通）
func LevelToBudgetTokens(level string) int {
	switch level {
	case "low":
		return 5000
	case "medium":
		return 10000
	case "high":
		return 20000
	case "xhigh":
		return 40000
	default:
		return 10000
	}
}

// NewProvider はプロバイダー名から Provider インスタンスを生成
func NewProvider(providerName string) (Provider, error) {
	// 登録済みプロバイダーをチェック
	if factory, ok := getRegisteredProvider(providerName); ok {
		apiKey := getAPIKeyForProvider(providerName)
		return factory(apiKey)
	}

	return nil, fmt.Errorf("unknown provider: %s", providerName)
}
