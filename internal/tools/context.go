package tools

import (
	"context"
	"io"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/audit"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/locator"
	lsplib "github.com/susugadx/xelyon-cli/internal/lsp"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// ToolResultInfo はツール実行結果の構造化データ。
// TUIモードで ToolResultCallback 経由で通知される。
type ToolResultInfo struct {
	ToolName string
	Args     map[string]string
	Result   string        // ツール出力全文
	Error    bool          // エラーかどうか
	Duration time.Duration // 実行時間
}

// ExecutionContext はツール実行時の周辺コンテキストを保持する。
// web_search などが現在のプロバイダー/モデルや対話 I/O を参照するために使用する。
// 各実行経路が明示的に組み立てて注入し、process-global 状態には依存しない。
type ExecutionContext struct {
	Context         context.Context
	Provider        api.Provider
	ProviderName    string
	Model           string
	Stdin           io.Reader
	Stdout          io.Writer
	Stderr          io.Writer
	PromptReader    *ui.MultilineReader
	Runtime         *ui.Runtime // スピナー停止に使用する UI Runtime
	Registry        *Registry
	ToolCache       ToolCacheInterface
	LSPClient       *lsplib.Client
	Config          *config.Config
	AutoApprove     bool
	AuditLogger     audit.ToolLogger
	LocatorRegistry *locator.Registry

	// ToolResultCallback はツール実行結果を構造化データとして通知するコールバック。
	// TUIモードで設定される。nilの場合はstdoutに従来形式で出力する。
	ToolResultCallback func(ToolResultInfo)
}

// Output は common.Output へ変換する。
func (ctx ExecutionContext) Output() common.Output {
	normalized := normalizeExecutionContext(ctx)
	return common.NewOutput(normalized.Stdout, normalized.Stderr)
}

// EffectiveContext は実行時に使う request context を返す。
func (ctx ExecutionContext) EffectiveContext() context.Context {
	normalized := normalizeExecutionContext(ctx)
	return normalized.Context
}

// PromptIO は対話 UI 用の入出力コンテキストへ変換する。
func (ctx ExecutionContext) PromptIO() ui.PromptIO {
	normalized := normalizeExecutionContext(ctx)
	return ui.NewPromptIOWithRuntime(normalized.Stdin, normalized.Stdout, normalized.Stderr, normalized.PromptReader, ctx.Runtime)
}

// ConfirmOptions は確認 UI 用の設定を返す。
func (ctx ExecutionContext) ConfirmOptions() common.ConfirmOptions {
	normalized := normalizeExecutionContext(ctx)
	return common.ConfirmOptions{
		AutoApprove: normalized.AutoApprove,
		Config:      normalized.Config,
	}
}

// EffectiveRegistry は実行時に使う Tool Registry を返す。
func (ctx ExecutionContext) EffectiveRegistry() *Registry {
	normalized := normalizeExecutionContext(ctx)
	return normalized.Registry
}

// EffectiveToolCache は実行時に使う ToolCache を返す。
func (ctx ExecutionContext) EffectiveToolCache() ToolCacheInterface {
	normalized := normalizeExecutionContext(ctx)
	return normalized.ToolCache
}

// EffectiveLSPClient は実行時に使う LSP Client を返す。
func (ctx ExecutionContext) EffectiveLSPClient() *lsplib.Client {
	normalized := normalizeExecutionContext(ctx)
	return normalized.LSPClient
}

// EffectiveConfig は実行時に使う設定を返す。
func (ctx ExecutionContext) EffectiveConfig() *config.Config {
	normalized := normalizeExecutionContext(ctx)
	return normalized.Config
}

// EffectiveProvider は実行時に使う Provider を返す。
func (ctx ExecutionContext) EffectiveProvider() api.Provider {
	normalized := normalizeExecutionContext(ctx)
	return normalized.Provider
}

// EffectiveAuditLogger は実行時に使う監査ロガーを返す。
func (ctx ExecutionContext) EffectiveAuditLogger() audit.ToolLogger {
	normalized := normalizeExecutionContext(ctx)
	return normalized.AuditLogger
}

// EffectiveLocatorRegistry は実行時に使う Locator Registry を返す。
func (ctx ExecutionContext) EffectiveLocatorRegistry() *locator.Registry {
	normalized := normalizeExecutionContext(ctx)
	return normalized.LocatorRegistry
}

func normalizeExecutionContext(ctx ExecutionContext) ExecutionContext {
	runtime := ui.DefaultRuntime()
	if ctx.Context == nil {
		ctx.Context = context.Background()
	}
	if ctx.Stdin == nil {
		ctx.Stdin = runtime.Input()
	}
	if ctx.Stdout == nil {
		ctx.Stdout = runtime.Output()
	}
	if ctx.Stderr == nil {
		ctx.Stderr = runtime.ErrorOutput()
	}
	if ctx.Registry == nil {
		ctx.Registry = DefaultRegistry
	}
	if ctx.Config == nil {
		ctx.Config = config.DefaultConfig()
	}
	if ctx.AuditLogger == nil {
		ctx.AuditLogger = audit.NewDisabledLogger()
	}
	if ctx.LocatorRegistry == nil {
		ctx.LocatorRegistry = locator.NewRegistry()
	}
	return ctx
}

type (
	registryContextKey struct{}
)

// WithRegistry は request context に Tool Registry を埋め込む。
func WithRegistry(ctx context.Context, registry *Registry) context.Context {
	if registry == nil {
		return ctx
	}
	return context.WithValue(ctx, registryContextKey{}, registry)
}

// RegistryFromContext は request context から Tool Registry を取得する。
func RegistryFromContext(ctx context.Context) *Registry {
	if ctx == nil {
		return DefaultRegistry
	}
	if registry, ok := ctx.Value(registryContextKey{}).(*Registry); ok && registry != nil {
		return registry
	}
	return DefaultRegistry
}

// WithConfig は request context に設定を埋め込む。
func WithConfig(ctx context.Context, cfg *config.Config) context.Context {
	return config.WithContext(ctx, cfg)
}

// ConfigFromContext は request context から設定を取得する。
func ConfigFromContext(ctx context.Context) *config.Config {
	return config.FromContext(ctx)
}
