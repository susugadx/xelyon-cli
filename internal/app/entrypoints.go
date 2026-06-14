package app

import (
	"context"

	"github.com/susugadx/xelyon-cli/internal/agent"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// HeadlessResult は headless mode の実行結果を表す。
type HeadlessResult = agent.HeadlessResult

const (
	// HeadlessStatusError は headless JSON の失敗 status。
	HeadlessStatusError = agent.HeadlessStatusError
)

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

// RunOnceWithConfig は指定設定で単一 query を 1 turn だけ実行して終了する。
func RunOnceWithConfig(query string, model string, provider api.Provider, cfg *config.Config, autoApprove bool, quiet bool) error {
	return agent.RunOnceWithConfig(query, model, provider, cfg, autoApprove, quiet)
}

// RunOnceWithImageWithConfig は指定設定で画像付きの単一 query を実行する。
func RunOnceWithImageWithConfig(query string, model string, provider api.Provider, imagePath string, cfg *config.Config, autoApprove bool, quiet bool) error {
	return agent.RunOnceWithImageWithConfig(query, model, provider, imagePath, cfg, autoApprove, quiet)
}
