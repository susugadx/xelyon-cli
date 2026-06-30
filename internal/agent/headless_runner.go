package agent

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/prompt"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

type headlessRunner struct {
	agent             *Agent
	provider          api.Provider
	model             string
	query             string
	options           HeadlessRunOptions
	startedAt         time.Time
	toolCalls         []ToolCallResult
	commands          []HeadlessCommandSummary
	finalChecks       []HeadlessFinalCheckSummary
	finalReply        string
	initErr           error
	cancelledErr      error
	finalCheckFailed  bool
	readOnlyViolation bool
}

// RunHeadlessWithConfig は指定設定で Headless モードのクエリを実行する。
// ctx が Done になるとサブエージェント含め処理を中断する。
func RunHeadlessWithConfig(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config) *HeadlessResult {
	return RunHeadlessWithConfigOptions(ctx, query, model, provider, cfg, HeadlessRunOptions{})
}

// RunHeadlessWithConfigOptions は指定設定と追加ポリシーで Headless モードのクエリを実行する。
// ctx が Done になるとサブエージェント含め処理を中断する。
func RunHeadlessWithConfigOptions(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config, options HeadlessRunOptions) *HeadlessResult {
	query = defaultHeadlessImagePrompt(query, options)
	startedAt := time.Now()
	runner := newHeadlessRunnerWithOptions(query, model, provider, cfg, options)
	runner.startedAt = startedAt
	defer runner.agent.Cleanup()
	result := runner.run(ctx)
	input := NewHeadlessInput(HeadlessInputSourceArgs, "", len([]byte(query)))
	if options.Image != nil {
		input = input.WithImage(NewHeadlessInputImageFromData(options.Image, provider.SupportsImages()))
	}
	return result.WithInput(input)
}

func defaultHeadlessImagePrompt(query string, options HeadlessRunOptions) string {
	if strings.TrimSpace(query) == "" && options.Image != nil {
		return DefaultImagePrompt
	}
	return query
}

func newHeadlessRunner(query, model string, provider api.Provider, cfg *config.Config) *headlessRunner {
	return newHeadlessRunnerWithOptions(query, model, provider, cfg, HeadlessRunOptions{})
}

func newHeadlessRunnerWithOptions(query, model string, provider api.Provider, cfg *config.Config, options HeadlessRunOptions) *headlessRunner {
	runtime := newHeadlessAgentRuntime(cfg, options)
	runtime.AutoApprove = true
	runtime.UI = uiruntime.NewRuntime(strings.NewReader(""), io.Discard, io.Discard)
	if !options.ReadOnly {
		configureRuntimeAuditLoggerFromEnv(runtime, io.Discard, false)
	}

	agent := NewAgentWithRuntime(model, provider, true, runtime)
	agent.setAutoApprove(true) // Headlessモードは自動承認（SafetyLow以外）

	if cfg != nil && cfg.SubAgentPrompt != "" {
		agent.SystemPrompt = prompt.BuildProviderSystemPromptWithConfig(cfg.SubAgentPrompt, agent.ProviderName, model, agent.cfg())
	}

	// プロジェクト instruction 読み込み（xelyon.yaml + guidance）
	initErr := initializeProjectInstructions(agent, projectInstructionApplyOptions{
		showStatus:       false,
		injectProjectMap: true,
	})

	// Headless Mode は Normal Mode 相当: 親と同じツール除外
	allowSubAgents := cfg == nil || cfg.SubAgentPrompt == ""
	toolVisibility := resolveToolVisibilityPolicyWithConfig(agent.ProviderName, model, agent.cfg(), toolSurfacePhaseNormal, toolVisibilityOptions{
		allowSubAgents: allowSubAgents,
	})
	excludedTools := agent.excludedToolsForVisibilityPolicy(toolVisibility)
	if options.ReadOnly {
		excludedTools = headlessReadOnlyExcludedTools(agent.registry(), excludedTools)
	}
	agent.registry().SetExcludedTools(excludedTools)

	// 初期ユーザーメッセージをHistoryに追加
	agent.History = append(agent.History, api.Message{
		Role:    "user",
		Content: query,
	})

	return &headlessRunner{
		agent:    agent,
		provider: provider,
		model:    model,
		query:    query,
		options:  options,
		initErr:  initErr,
	}
}

func (r *headlessRunner) run(ctx context.Context) *HeadlessResult {
	if r.initErr != nil {
		return r.errorResult(HeadlessErrorTypeConfig, r.initErr.Error())
	}
	if r.options.Image != nil && !r.provider.SupportsImages() {
		return r.errorResult(HeadlessErrorTypeUnsupportedCapability, fmt.Sprintf("provider %q does not support image input", r.provider.Name()))
	}

	maxIterations := normalizeToolLoopLimit(r.agent.cfg().General.ToolLoopLimit)

	for iteration := 0; maxIterations == 0 || iteration < maxIterations; iteration++ {
		if ctx.Err() != nil {
			return r.errorResult(HeadlessErrorTypeCancelled, ctx.Err().Error())
		}

		response, err := r.requestAssistantResponse(ctx, iteration)
		if err != nil {
			return r.errorResult(HeadlessErrorTypeAPI, err.Error())
		}

		if done := r.handleAssistantResponse(ctx, response); done {
			return r.successResult()
		}
	}

	if maxIterations > 0 {
		return r.loopLimitResult(maxIterations)
	}
	return r.successResult()
}
