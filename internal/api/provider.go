package api

import (
	"context"
	"fmt"
	"github.com/susugadx/xelyon-cli/internal/config"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
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

// getAPIKeyForProvider はプロバイダー名から環境変数のAPIキーを取得
func getAPIKeyForProvider(providerName string) string {
	switch strings.ToLower(providerName) {
	case "deepseek":
		return os.Getenv("DEEPSEEK_API_KEY")
	case "openai":
		return os.Getenv("OPENAI_API_KEY")
	case "gemini":
		return os.Getenv("GEMINI_API_KEY")
	case "claude", "anthropic":
		return os.Getenv("ANTHROPIC_API_KEY")
	case "ollama":
		baseURL := os.Getenv("OLLAMA_BASE_URL")
		if baseURL == "" {
			return "http://localhost:11434"
		}
		return baseURL
	case "groq":
		return os.Getenv("GROQ_API_KEY")
	case "openrouter":
		return os.Getenv("OPENROUTER_API_KEY")
	case "bedrock":
		// Bedrock は AWS 認証チェーンを使用するため、常にダミー値を返す
		return "aws-credentials"
	case "serper":
		return os.Getenv("SERPER_API_KEY")
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

// ModelLister はモデル一覧取得に対応するプロバイダーのオプショナルインターフェース
// 現時点では OllamaProvider のみが実装
type ModelLister interface {
	// ListModels はインストール済み/利用可能なモデルの一覧を取得
	ListModels() ([]string, error)
}

// MCPToolProvider はMCPツール設定に対応するプロバイダーのオプショナルインターフェース
// 現時点では GeminiProvider のみが実装
type MCPToolProvider interface {
	// SetMCPEnabled はMCPが有効かどうかを設定する（レガシー、互換性のため）
	SetMCPEnabled(enabled bool)

	// SetMCPTools はMCPツールの定義を設定する
	SetMCPTools(tools []GeminiFunctionDeclaration)
}

// OpenAIMCPToolProvider はOpenAI用のMCPツール設定インターフェース
// OpenAI Chat Completions / Responses API の Function Calling で使用
type OpenAIMCPToolProvider interface {
	// SetMCPTools はMCPツールの定義を設定する
	SetMCPTools(tools []OpenAIToolFunction)
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

// knownModelMaxOutputTokens は既知モデルの最大出力トークン数マップ
var knownModelMaxOutputTokens = map[string]int{
	"deepseek-chat":          8192,
	"deepseek-reasoner":      64000,
	"claude-sonnet-4-5":      16384,
	"claude-opus-4-5":        32768,
	"gpt-5.2":                16384,
	"gemini-2.5-flash":       65536,
	"gemini-3-flash-preview": 65536,
}

// GetMaxOutputTokens は指定されたプロバイダーとモデルの最大出力トークン数を取得する（4段階フォールバック）
// 1. Extended Thinking 有効時は BudgetTokens を考慮
// 2. ユーザーの model_overrides (config)
// 3. 既知モデルマップ (knownModelMaxOutputTokens)
// 4. プロバイダーのデフォルト値 (ProviderModelConfig.MaxOutputTokens)
func GetMaxOutputTokens(providerName, model string) int {
	cfg := config.GetGlobalConfig()

	maxTokens := 0
	pName := strings.ToLower(providerName)
	pCfg, ok := cfg.ProviderModels[pName]
	if !ok {
		pCfg = config.ProviderModelConfig{}
	}

	// 1. ユーザーの model_overrides
	if pCfg.ModelOverrides != nil {
		if override, ok := pCfg.ModelOverrides[model]; ok && override.MaxOutputTokens > 0 {
			maxTokens = override.MaxOutputTokens
		}
	}

	// 2. 既知モデルマップ
	if maxTokens == 0 {
		if tokens, ok := knownModelMaxOutputTokens[model]; ok {
			maxTokens = tokens
		}
	}

	// 3. プロバイダーのデフォルト値
	if maxTokens == 0 {
		maxTokens = pCfg.MaxOutputTokens
	}

	// 4. Extended Thinking 有効時は BudgetTokens を考慮
	// Claude Opus 4.6 などでは max_tokens = budget_tokens + output_tokens とする必要がある
	if cfg.Thinking.Enabled && (pName == "claude" || pName == "anthropic" || pName == "bedrock") {
		budget := LevelToBudgetTokens(cfg.Thinking.Level)
		return budget + maxTokens
	}

	return maxTokens
}

// SupportsImages はプロバイダー名から画像対応を判定
func SupportsImages(providerName string) bool {
	switch strings.ToLower(providerName) {
	case "claude", "anthropic", "openai", "gemini", "bedrock":
		return true
	case "deepseek", "ollama", "groq":
		return false
	default:
		return false
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
	// まず登録済みプロバイダーをチェック
	if factory, ok := getRegisteredProvider(providerName); ok {
		apiKey := getAPIKeyForProvider(providerName)
		return factory(apiKey)
	}

	// フォールバック: 移行中の互換性のため既存switchも維持
	switch strings.ToLower(providerName) {
	case "deepseek":
		apiKey := os.Getenv("DEEPSEEK_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("DEEPSEEK_API_KEY not set")
		}
		return NewDeepSeekProvider(apiKey), nil

	case "openai":
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("OPENAI_API_KEY not set")
		}
		return NewOpenAIProvider(apiKey), nil

	case "gemini":
		apiKey := os.Getenv("GEMINI_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("GEMINI_API_KEY not set")
		}
		return NewGeminiProvider(apiKey), nil

	case "claude", "anthropic":
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY not set")
		}
		return NewClaudeProvider(apiKey), nil

	case "ollama":
		baseURL := os.Getenv("OLLAMA_BASE_URL")
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
		return NewOllamaProvider(baseURL), nil

	case "groq":
		apiKey := os.Getenv("GROQ_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("GROQ_API_KEY not set")
		}
		return NewGroqProvider(apiKey), nil

	default:
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}
}
