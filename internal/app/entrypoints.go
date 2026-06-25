package app

import (
	"context"

	"github.com/susugadx/xelyon-cli/internal/agent"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// HeadlessResult は headless mode の実行結果を表す。
type HeadlessResult = agent.HeadlessResult

// HeadlessInput は headless prompt input metadata を表す。
type HeadlessInput = agent.HeadlessInput

// HeadlessSummary は headless JSON に出す CI 向け runtime summary を表す。
type HeadlessSummary = agent.HeadlessSummary

// HeadlessCommandSummary は tool 経由で実行されたコマンドの要約を表す。
type HeadlessCommandSummary = agent.HeadlessCommandSummary

// HeadlessFinalCheckSummary は final_checks.commands の実行結果要約を表す。
type HeadlessFinalCheckSummary = agent.HeadlessFinalCheckSummary

// HeadlessInputSource は headless prompt の入力元を表す。
type HeadlessInputSource = agent.HeadlessInputSource

// HeadlessExitPolicy は headless JSON の推奨 exit code policy を表す。
type HeadlessExitPolicy = agent.HeadlessExitPolicy

// HeadlessRunOptions は headless 実行時の追加ポリシーを表す。
type HeadlessRunOptions = agent.HeadlessRunOptions

const (
	// HeadlessSchemaVersion は headless JSON contract の schema version。
	HeadlessSchemaVersion = agent.HeadlessSchemaVersion
	// HeadlessStatusError は headless JSON の失敗 status。
	HeadlessStatusError = agent.HeadlessStatusError
	// HeadlessExitPolicyLegacy は既存互換の non-zero error code policy。
	HeadlessExitPolicyLegacy = agent.HeadlessExitPolicyLegacy
	// HeadlessExitPolicyCI は CI 向けの詳細 exit code policy。
	HeadlessExitPolicyCI = agent.HeadlessExitPolicyCI
	// HeadlessErrorTypeConfig は CLI/config/input validation 系の headless error type。
	HeadlessErrorTypeConfig = agent.HeadlessErrorTypeConfig
	// HeadlessErrorTypeFinalCheckFailed は headless final check 失敗の error type。
	HeadlessErrorTypeFinalCheckFailed = agent.HeadlessErrorTypeFinalCheckFailed
	// HeadlessInputSourceArgs は positional args 由来の prompt input source。
	HeadlessInputSourceArgs = agent.HeadlessInputSourceArgs
	// HeadlessInputSourcePromptFile は --prompt-file 由来の prompt input source。
	HeadlessInputSourcePromptFile = agent.HeadlessInputSourcePromptFile
	// HeadlessInputSourceStdin は stdin 由来の prompt input source。
	HeadlessInputSourceStdin = agent.HeadlessInputSourceStdin
)

// NewHeadlessInput は headless prompt input metadata を生成する。
func NewHeadlessInput(source HeadlessInputSource, promptFile string, byteCount int) HeadlessInput {
	return agent.NewHeadlessInput(source, promptFile, byteCount)
}

// ParseHeadlessExitPolicy は CLI flag 値を HeadlessExitPolicy に変換する。
func ParseHeadlessExitPolicy(value string) (HeadlessExitPolicy, error) {
	return agent.ParseHeadlessExitPolicy(value)
}

// ApplyHeadlessExitPolicy は HeadlessResult に exit policy と推奨 exit code を反映する。
func ApplyHeadlessExitPolicy(result *HeadlessResult, policy HeadlessExitPolicy) (*HeadlessResult, error) {
	return agent.ApplyHeadlessExitPolicy(result, policy)
}

// NewHeadlessConfigErrorResult は config/input validation 用の headless JSON error を作る。
func NewHeadlessConfigErrorResult(provider, model, message string) *HeadlessResult {
	return agent.NewErrorResult(provider, model, agent.HeadlessErrorTypeConfig, message, 0)
}

// NewHeadlessUsageErrorResult は CLI 入力 validation 用の headless JSON error を作る。
func NewHeadlessUsageErrorResult(provider, model, message string) *HeadlessResult {
	return agent.NewUsageErrorResult(provider, model, message, 0)
}

// NewHeadlessProviderSetupRequiredResult は provider setup 未完了の headless JSON error を作る。
func NewHeadlessProviderSetupRequiredResult(provider, model, message string) *HeadlessResult {
	return agent.NewErrorResult(provider, model, agent.HeadlessErrorTypeProviderSetupRequired, message, 0)
}

// RunLegacyInteractiveWithConfig は legacy classic REPL を実行する。
func RunLegacyInteractiveWithConfig(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
	agent.RunLegacyInteractiveWithConfig(model, provider, cfg, autoApprove)
}

// RunLegacyInteractiveWithResumeWithConfig は legacy classic REPL で前回 session を再開する。
func RunLegacyInteractiveWithResumeWithConfig(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
	agent.RunLegacyInteractiveWithResumeWithConfig(model, provider, cfg, autoApprove)
}

// RunLegacyInteractiveWithImageWithConfig は legacy classic REPL で画像付き初回 turn を実行する。
func RunLegacyInteractiveWithImageWithConfig(query string, model string, provider api.Provider, imagePath string, cfg *config.Config, autoApprove bool) error {
	return agent.RunLegacyInteractiveWithImageWithConfig(query, model, provider, imagePath, cfg, autoApprove)
}

// RunHeadlessWithConfig は指定設定で headless mode の query を実行する。
func RunHeadlessWithConfig(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config) *HeadlessResult {
	return agent.RunHeadlessWithConfig(ctx, query, model, provider, cfg)
}

// RunHeadlessWithConfigOptions は指定設定と追加ポリシーで headless mode の query を実行する。
func RunHeadlessWithConfigOptions(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config, options HeadlessRunOptions) *HeadlessResult {
	return agent.RunHeadlessWithConfigOptions(ctx, query, model, provider, cfg, options)
}

// RunOnceWithConfig は指定設定で単一 query を 1 turn だけ実行して終了する。
func RunOnceWithConfig(query string, model string, provider api.Provider, cfg *config.Config, autoApprove bool, quiet bool) error {
	return agent.RunOnceWithConfig(query, model, provider, cfg, autoApprove, quiet)
}

// RunOnceWithImageWithConfig は指定設定で画像付きの単一 query を実行する。
func RunOnceWithImageWithConfig(query string, model string, provider api.Provider, imagePath string, cfg *config.Config, autoApprove bool, quiet bool) error {
	return agent.RunOnceWithImageWithConfig(query, model, provider, imagePath, cfg, autoApprove, quiet)
}
