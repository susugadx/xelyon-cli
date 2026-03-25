package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/audit"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/prompt"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/subagent"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// RunHeadlessWithConfig は指定設定で Headless モードのクエリを実行する。
// ctx が Done になるとサブエージェント含め処理を中断する。
func RunHeadlessWithConfig(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config) *HeadlessResult {
	startTime := time.Now()

	// Agent初期化
	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.AutoApprove = true
	runtime.UI = ui.NewRuntime(strings.NewReader(""), io.Discard, io.Discard)
	if logger, err := audit.NewDefaultLogger(os.Getenv("XELYON_AUDIT_LOG") == "1"); err == nil {
		runtime.AuditLogger = logger
	}
	agent := NewAgentWithRuntime(model, provider, true, runtime)
	defer agent.Cleanup()
	agent.setAutoApprove(true) // Headlessモードは自動承認（SafetyLow以外）
	if cfg != nil && cfg.SubAgentPrompt != "" {
		agent.SystemPrompt = prompt.BuildProviderSystemPromptWithConfig(cfg.SubAgentPrompt, provider.Name(), model, agent.cfg())
	}

	// プロジェクト設定読み込み（xelyon.yaml）
	if pc := loadProjectConfig(); pc != nil {
		agent.SystemPrompt = injectProjectConfig(agent.SystemPrompt, pc, "")
		// headless では hooks 解決のみ（UI 表示不要）
		if resolved := config.ResolveHooks(agent.cfg(), pc); resolved != nil {
			cfg := agent.cfg()
			cfg.Hooks = *resolved
		}
	}
	injectProjectMap(agent, "")

	// Headless Mode は Normal Mode 相当: 親と同じツール除外
	excludedTools := normalModeExcludedTools()
	if cfg != nil && cfg.SubAgentPrompt != "" {
		excludedTools = appendUniqueStrings(excludedTools, "spawn_agent", "wait_agent")
	}
	agent.registry().SetExcludedTools(excludedTools)

	// ツール呼び出し結果を記録
	var allToolCalls []ToolCallResult

	// 初期ユーザーメッセージをHistoryに追加
	agent.History = append(agent.History, api.Message{
		Role:    "user",
		Content: query,
	})

	maxIterations := normalizeToolLoopLimit(agent.cfg().General.ToolLoopLimit)
	var finalResponse string
	execCtx := agent.toolExecutionContext(ctx, nil, nil, nil)

	for iteration := 0; maxIterations == 0 || iteration < maxIterations; iteration++ {
		// 親 context がキャンセルされた場合は即終了
		if ctx.Err() != nil {
			duration := time.Since(startTime).Milliseconds()
			return attachHeadlessStats(agent, NewErrorResult(provider.Name(), model, "cancelled", ctx.Err().Error(), duration))
		}
		// API呼び出し
		timeout := time.Duration(agent.cfg().APIRetry.Timeout) * time.Second
		if timeout <= 0 {
			timeout = 3600 * time.Second
		}
		reqCtx, cancel := context.WithTimeout(ctx, timeout)
		// ツールループ初回のみ Project Map を更新。ループ中の再生成はキャッシュを破壊する。
		if iteration == 0 {
			agent.refreshProjectPromptIfDirty(query)
		}

		response, err := provider.ChatWithTools(agent.requestContext(reqCtx), agent.SystemPrompt, agent.History, model)
		cancel()

		if err != nil {
			duration := time.Since(startTime).Milliseconds()
			return attachHeadlessStats(agent, NewErrorResult(provider.Name(), model, "api_error", err.Error(), duration))
		}

		// ツール呼び出し解析
		parsedCalls := agent.parseToolCalls(response)
		assignHeadlessRescueToolCallIDs(parsedCalls)

		// ツール呼び出しがなければ最終レスポンスとして終了
		if len(parsedCalls) == 0 {
			finalResponse = response
			break
		}

		appendHeadlessToolCallsToHistory(agent, response, parsedCalls)

		// ツール実行と結果収集
		for _, tc := range parsedCalls {
			toolCount := len(allToolCalls) + 1
			subagent.EmitEvent(ctx, subagent.SubAgentEvent{
				Tool:      tc.Tool,
				Phase:     "start",
				FilePath:  extractToolFilePath(tc),
				ToolIndex: toolCount,
			})

			output, change := tools.ExecuteQuietWithContext(execCtx, tc)
			agent.noteProjectMapMutation(tc, change)

			// 成功判定（"Error:"を含むかどうかで簡易判定）
			success := !strings.Contains(output, "Error:")
			if agent.Stats != nil {
				agent.Stats.AddToolExecution(tc.Tool)
			}

			allToolCalls = append(allToolCalls, ToolCallResult{
				Tool:    tc.Tool,
				Args:    tc.Args,
				Output:  output,
				Success: success,
			})
			event := subagent.SubAgentEvent{
				Tool:      tc.Tool,
				Phase:     "end",
				FilePath:  extractToolFilePath(tc),
				Success:   success,
				Output:    truncateEventOutput(output, 200),
				ToolIndex: toolCount,
			}
			if tc.Tool == "str_replace" {
				event.OldStr = tc.Args["old_str"]
				event.NewStr = tc.Args["new_str"]
			}
			subagent.EmitEvent(ctx, event)
			appendHeadlessToolResultToHistory(agent, tc, output)

			// ファイル変更履歴を記録
			if change != nil {
				agent.changeStack = append(agent.changeStack, *change)
			}
		}

		finalResponse = response // 最大イテレーション到達時のフォールバック
	}

	duration := time.Since(startTime).Milliseconds()
	return attachHeadlessStats(agent, NewSuccessResult(provider.Name(), model, finalResponse, allToolCalls, duration))
}

func attachHeadlessStats(agent *Agent, result *HeadlessResult) *HeadlessResult {
	if agent == nil || result == nil || agent.Stats == nil {
		return result
	}

	agent.statsMu.Lock()
	defer agent.statsMu.Unlock()

	result.Tokens = &TokenUsage{
		Input:    agent.Stats.InputTokens,
		Cached:   agent.Stats.CachedInputTokens,
		Output:   agent.Stats.OutputTokens,
		Thinking: agent.Stats.ThinkingTokens,
		Total:    agent.Stats.TotalTokens(),
	}
	result.Cost = agent.Stats.EstimatedCost()
	return result
}

func assignHeadlessRescueToolCallIDs(toolCalls []*tools.ToolCall) {
	for i, tc := range toolCalls {
		if tc == nil || tc.ID != "" {
			continue
		}
		toolCalls[i].ID = fmt.Sprintf("call_rescue_%03d", i+1)
	}
}

func appendHeadlessToolCallsToHistory(agent *Agent, response string, toolCalls []*tools.ToolCall) {
	if agent == nil || len(toolCalls) == 0 {
		return
	}

	explanation, _ := extractExplanationAndTool(response)
	reasoningContent := agent.getLastReasoningContent()
	historyContent, historyReasoning := agent.assistantToolHistoryContent(explanation, reasoningContent)

	openAIToolCalls := make([]api.OpenAIToolCall, len(toolCalls))
	for i, tc := range toolCalls {
		openAIToolCalls[i] = api.OpenAIToolCall{
			ID:   tc.ID,
			Type: "function",
			Function: api.OpenAIToolCallFunction{
				Name:      tc.Tool,
				Arguments: argsToJSON(tc.RawArgs),
			},
		}
		if i == 0 {
			openAIToolCalls[i].ThoughtSignature = tc.ThoughtSignature
			openAIToolCalls[i].ThoughtParts = tc.ThoughtParts
		}
	}

	agent.History = append(agent.History, api.Message{
		Role:             "assistant",
		Content:          historyContent,
		ReasoningContent: historyReasoning,
		ToolCalls:        openAIToolCalls,
	})
	if agent.Stats != nil {
		agent.Stats.AssistantMessages++
	}
}

func appendHeadlessToolResultToHistory(agent *Agent, toolCall *tools.ToolCall, result string) {
	if agent == nil || toolCall == nil {
		return
	}
	if toolCall.ID != "" {
		agent.History = append(agent.History, api.Message{
			Role:       "tool",
			Content:    result,
			ToolCallID: toolCall.ID,
			ToolName:   toolCall.Tool,
		})
		return
	}

	agent.History = append(agent.History, api.Message{
		Role:    "user",
		Content: fmt.Sprintf("[Tool Result for %s]\n%s", toolCall.Tool, result),
	})
}

// extractToolFilePath はツール呼び出しから表示用ターゲットを抽出する。
func extractToolFilePath(tc *tools.ToolCall) string {
	if tc == nil {
		return ""
	}

	switch tc.Tool {
	case "read_file", "write_file", "str_replace", "delete_file", "list_dir", "lint", "format":
		if path := tc.Args["path"]; path != "" {
			return path
		}
	case "search_code":
		if pattern := tc.Args["pattern"]; pattern != "" {
			return fmt.Sprintf("%q", pattern)
		}
		if path := tc.Args["path"]; path != "" {
			return path
		}
	case "bash":
		if cmd := tc.Args["command"]; cmd != "" {
			return truncateEventOutput(cmd, 40)
		}
	}

	if rawPaths := tc.Args["paths"]; rawPaths != "" {
		var paths []string
		if err := json.Unmarshal([]byte(rawPaths), &paths); err == nil && len(paths) > 0 {
			if len(paths) == 1 {
				return paths[0]
			}
			return fmt.Sprintf("%s (+%d files)", paths[0], len(paths)-1)
		}
	}
	if path := tc.Args["path"]; path != "" {
		return path
	}
	if pattern := tc.Args["pattern"]; pattern != "" {
		return pattern
	}
	if symbol := tc.Args["symbol"]; symbol != "" {
		return symbol
	}
	return ""
}

// truncateEventOutput はイベント出力を制限する。
func truncateEventOutput(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// RunOnceWithConfig は指定設定で単一クエリを1ターンだけ実行して終了する。
func RunOnceWithConfig(query string, model string, provider api.Provider, cfg *config.Config, autoApprove bool, quiet bool) error {
	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.AutoApprove = autoApprove
	out := runtime.effectiveUI().Output()

	// 監査ログ初期化（環境変数で制御: XELYON_AUDIT_LOG=1 で有効化）
	auditEnabled := os.Getenv("XELYON_AUDIT_LOG") == "1"
	logger, err := audit.NewDefaultLogger(auditEnabled)
	if err != nil {
		yellow.Fprintf(out, "Warning: Failed to initialize audit log: %v\n", err)
	}
	if auditEnabled && !quiet {
		green.Fprintln(out, "📝 Audit logging enabled")
	}
	if logger != nil {
		runtime.AuditLogger = logger
	}
	agent := NewAgentWithRuntime(model, provider, false, runtime)
	agent.setAutoApprove(autoApprove)
	defer agent.Cleanup()

	// ヘッダー表示（quiet 時はスキップ）
	if !quiet {
		printHeaderToWriter(runtime.effectiveUI().Output(), model, provider)
		printModeInfoToWriter(runtime.effectiveUI().Output(), autoApprove, false)
	}

	// プロジェクト設定読み込み（xelyon.yaml）
	if pc := loadProjectConfig(); pc != nil {
		applyProjectConfig(agent, pc)
	}
	injectProjectMap(agent, "")

	// 明示的に1ターンのみ実行（ChatOnce は stdin を読まず、REPL に入らない）
	return agent.ChatOnce(query)
}

// RunOnceWithImageWithConfig は指定設定で画像付きの単一クエリを実行する。
func RunOnceWithImageWithConfig(query string, model string, provider api.Provider, imagePath string, cfg *config.Config, autoApprove bool) {
	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.AutoApprove = autoApprove
	out := runtime.effectiveUI().Output()

	// 監査ログ初期化（環境変数で制御: XELYON_AUDIT_LOG=1 で有効化）
	auditEnabled := os.Getenv("XELYON_AUDIT_LOG") == "1"
	logger, err := audit.NewDefaultLogger(auditEnabled)
	if err != nil {
		yellow.Fprintf(out, "Warning: Failed to initialize audit log: %v\n", err)
	}
	if logger != nil {
		runtime.AuditLogger = logger
	}
	agent := NewAgentWithRuntime(model, provider, false, runtime)
	agent.setAutoApprove(autoApprove)
	defer agent.Cleanup()

	// ヘッダー表示
	printHeaderToWriter(runtime.effectiveUI().Output(), model, provider)
	printModeInfoToWriter(runtime.effectiveUI().Output(), autoApprove, false)

	// プロバイダーが画像対応かチェック
	if !api.SupportsImages(provider.Name()) {
		red.Fprintf(out, "❌ Provider '%s' does not support image input\n", provider.Name())
		_, _ = fmt.Fprintln(agent.output(), "Supported providers for image input: gemini, claude, openai")
		return
	}

	// 画像読み込み
	image, err := api.LoadImage(imagePath)
	if err != nil {
		red.Fprintf(out, "❌ Failed to load image: %v\n", err)
		return
	}
	green.Fprintf(out, "🖼️  Image loaded: %s (%s)\n", image.Path, api.FormatImageSize(image.Size))

	// プロジェクト設定読み込み（xelyon.yaml）
	if pc := loadProjectConfig(); pc != nil {
		applyProjectConfig(agent, pc)
	}
	injectProjectMap(agent, "")

	_, _ = fmt.Fprintln(agent.output())

	// デフォルトメッセージ
	if query == "" {
		query = "Please analyze this image."
	}

	// 画像付きで会話
	agent.chatWithImage(query, image)

	// 対話ループに入る
	runtimeUI := runtime.effectiveUI()
	mlReader := ui.NewMultilineReaderWithRuntime(runtimeUI)
	runtimeUI.SetPromptReader(mlReader)
	mlReader.EnableBracketedPaste()
	defer mlReader.DisableBracketedPaste()
	agent.setPromptReader(mlReader)

	for {
		mlReader.FlushInput()
		agent.PrintStatusFooter()

		input, err := mlReader.ReadInput("\n> ")
		if err != nil {
			break
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// 特殊コマンド
		if handleSpecialCommand(input, agent) {
			continue
		}

		// 通常の会話
		agent.chat(input)
	}
}
