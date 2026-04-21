package agent

import (
	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/prompt"
)

func loadProjectConfig() *config.ProjectConfig {
	return config.LoadProjectConfig()
}

// injectProjectConfig は ProjectConfig を SystemPrompt に注入する。
// 入力内容に一致した rules/context のみを注入する。
func injectProjectConfig(systemPrompt string, pc *config.ProjectConfig, input string) string {
	systemPrompt = prompt.StripProjectConfigSections(systemPrompt)
	if pc == nil {
		return systemPrompt
	}

	selection := config.SelectProjectPromptSelection(pc, input)
	projectBlock := prompt.BuildProjectConfigBlock(selection.Rules, selection.Contexts)
	return prompt.InjectProjectConfigBlock(systemPrompt, projectBlock)
}

// applyProjectConfig はプロジェクト設定をエージェントに適用する統一ヘルパー。
// SystemPrompt 注入 + final checks 解決 + UI 表示を行う。
func applyProjectConfig(agent *Agent, pc *config.ProjectConfig) {
	if pc == nil {
		return
	}

	// 1. System prompt 注入
	agent.SystemPrompt = injectProjectConfig(agent.SystemPrompt, pc, "")

	// 2. final checks 解決（xelyon.yaml 優先、config.yaml フォールバック）
	if resolved := config.ResolveFinalChecks(agent.cfg(), pc); resolved != nil {
		cfg := agent.cfg()
		cfg.FinalChecks = *resolved
	}

	// 3. UI 表示
	green.Fprintln(agent.output(), "📋 xelyon.yaml loaded")
}

// rebuildSystemPromptForCurrentProvider は現在の provider/model に合わせて
// SystemPrompt をベースから再構築する。
func (a *Agent) rebuildSystemPromptForCurrentProvider() {
	a.promptManager().RebuildSystemPromptForCurrentProvider()
}

func estimateProjectConfigTokens(pc *config.ProjectConfig) int {
	if pc == nil {
		return 0
	}

	selection := config.SelectProjectPromptSelection(pc, "")
	return token.EstimateTokenCount(prompt.BuildProjectConfigBlock(selection.Rules, selection.Contexts))
}

func (a *Agent) refreshProjectPrompt(input string) {
	a.promptManager().RefreshProjectPrompt(input)
}

func (a *Agent) refreshProjectPromptIfDirty(input string) {
	a.promptManager().RefreshProjectPromptIfDirty(input)
}

func (a *Agent) shouldRefreshProjectPrompt(input string) bool {
	return a.promptManager().ShouldRefreshProjectPrompt(input)
}
