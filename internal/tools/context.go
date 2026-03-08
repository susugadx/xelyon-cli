package tools

import (
	"context"
	"io"
	"os"

	"github.com/susugadx/xelyon-cli/internal/audit"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// ExecutionContext はツール実行時の周辺コンテキストを保持する。
// web_search などが現在のプロバイダー/モデルや対話 I/O を参照するために使用する。
// 各実行経路が明示的に組み立てて注入し、process-global 状態には依存しない。
type ExecutionContext struct {
	ProviderName string
	Model        string
	Stdin        io.Reader
	Stdout       io.Writer
	Stderr       io.Writer
	PromptReader *ui.MultilineReader
	Registry     *Registry
	ToolCache    ToolCacheInterface
	Config       *config.Config
	AutoApprove  bool
	AuditLogger  audit.ToolLogger
}

// DefaultExecutionContext は標準入出力を使う実行コンテキストを返す。
func DefaultExecutionContext() ExecutionContext {
	return normalizeExecutionContext(ExecutionContext{
		Registry:    DefaultRegistry,
		ToolCache:   GlobalToolCache,
		Config:      config.GetGlobalConfig(),
		AutoApprove: common.GlobalAutoApprove,
		AuditLogger: audit.GetLogger(),
	})
}

// Output は common.Output へ変換する。
func (ctx ExecutionContext) Output() common.Output {
	normalized := normalizeExecutionContext(ctx)
	return common.NewOutput(normalized.Stdout, normalized.Stderr)
}

// PromptIO は対話 UI 用の入出力コンテキストへ変換する。
func (ctx ExecutionContext) PromptIO() ui.PromptIO {
	normalized := normalizeExecutionContext(ctx)
	return ui.NewPromptIO(normalized.Stdin, normalized.Stdout, normalized.Stderr, normalized.PromptReader)
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

// EffectiveConfig は実行時に使う設定を返す。
func (ctx ExecutionContext) EffectiveConfig() *config.Config {
	normalized := normalizeExecutionContext(ctx)
	return normalized.Config
}

// EffectiveAuditLogger は実行時に使う監査ロガーを返す。
func (ctx ExecutionContext) EffectiveAuditLogger() audit.ToolLogger {
	normalized := normalizeExecutionContext(ctx)
	return normalized.AuditLogger
}

func normalizeExecutionContext(ctx ExecutionContext) ExecutionContext {
	if ctx.Stdin == nil {
		ctx.Stdin = os.Stdin
	}
	if ctx.Stdout == nil {
		ctx.Stdout = os.Stdout
	}
	if ctx.Stderr == nil {
		ctx.Stderr = os.Stderr
	}
	if ctx.Registry == nil {
		ctx.Registry = DefaultRegistry
	}
	if ctx.ToolCache == nil {
		ctx.ToolCache = GlobalToolCache
	}
	if ctx.Config == nil {
		ctx.Config = config.GetGlobalConfig()
	}
	if ctx.AuditLogger == nil {
		ctx.AuditLogger = audit.GetLogger()
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
