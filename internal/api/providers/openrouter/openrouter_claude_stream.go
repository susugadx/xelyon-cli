package openrouter

import (
	"context"
	"net/http"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/providers/claude"
	claudestream "github.com/susugadx/xelyon-cli/internal/api/providers/claude_stream"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

// isClaudeModel はモデル名が Claude モデルかを判定する。
func isClaudeModel(model string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(model)), "anthropic/claude-")
}

// isCompactionSupported はモデルが Compaction 対応か判定する。
func isCompactionSupported(model string) bool {
	return claude.IsCompactionSupportedModel(model)
}

func buildOpenRouterClaudeContextManagement(model string, compression config.CompressionConfig, betaHeaders []string) (*claude.ContextManagement, []string) {
	if !isClaudeModel(model) {
		return nil, betaHeaders
	}

	contextManagement := claude.BuildContextManagement(compression, isCompactionSupported(model))
	if contextManagement == nil {
		return nil, betaHeaders
	}

	return contextManagement, claude.MergeAnthropicBetaHeaders(betaHeaders, contextManagement)
}

func shouldUseOpenRouterClaudeAPI(model string, compression config.CompressionConfig) bool {
	contextManagement, _ := buildOpenRouterClaudeContextManagement(model, compression, nil)
	return contextManagement != nil
}

// chatWithClaudeAPI は OpenRouter Claude 経路の request 構築と送信を担う。
func (p *Provider) chatWithClaudeAPI(ctx context.Context, systemPrompt string, history []api.Message, userMessage, model string, image *api.ImageData, route openRouterRoutePlan) (string, error) {
	payload, err := p.buildClaudeChatPayload(ctx, systemPrompt, history, userMessage, model, image)
	if err != nil {
		return "", err
	}
	return p.executeClaudeStreamingRequest(ctx, route.APIURL, payload, image != nil)
}

// handleClaudeStreamingResponse は Anthropic SSE の最小共通処理を claude_stream に委譲する。
func (p *Provider) handleClaudeStreamingResponse(ctx context.Context, resp *http.Response, spinner *uiruntime.Spinner) (string, error) {
	var toolCallsOutput strings.Builder
	toolUses := claudestream.NewToolUseCollector()
	compaction := claudestream.NewCompactionCollector()
	var lastUsage *api.Usage

	handler := func(event claudestream.StreamEvent, data string) (string, bool, error) {
		switch event.Type {
		case "message_start":
			if usage, err := claudestream.DecodeMessageStartUsage(data); err == nil {
				lastUsage = usage
			}
		case "message_delta":
			if event.Usage != nil && event.Usage.OutputTokens > 0 {
				lastUsage = claudestream.UpdateUsageFromMessageDelta(lastUsage, &claudestream.StreamUsage{
					OutputTokens: event.Usage.OutputTokens,
				}, false)
			}
			return "", false, nil
		case "message_stop":
			return "", true, nil
		case "content_block_start":
			claudestream.HandleContentBlockStart(event, toolUses, compaction)
			return "", false, nil
		case "content_block_delta":
			return claudestream.HandleContentBlockDelta(event, toolUses, compaction, nil), false, nil
		case "content_block_stop":
			if toolJSON := claudestream.HandleContentBlockStop(event, toolUses, compaction, claude.ConvertToolUseToToolJSON); toolJSON != "" {
				toolCallsOutput.WriteString(toolJSON)
			}
			return "", false, nil
		}
		return "", false, nil
	}

	content, streamErr := claudestream.RunStreamingResponse(ctx, resp, spinner, handler, claudestream.RunnerOptions{
		CancelMode:        claudestream.CancelModePartialAsError,
		WarnOnPartial:     false,
		IgnoreDecodeError: true,
		EnableIdleTimeout: false,
	})

	// 旧実装互換: cancel/stream error 時は usage を確定値として通知しない。
	if streamErr == nil && p.usageCallback != nil && lastUsage != nil {
		p.usageCallback(*lastUsage)
	}

	compactionOutput := compaction.Output()
	if compactionOutput != "" {
		content = "[COMPACTION]\n" + compactionOutput + "\n[/COMPACTION]\n" + content
	}

	if toolCallsOutput.Len() > 0 {
		if content != "" {
			return content + toolCallsOutput.String(), streamErr
		}
		return toolCallsOutput.String(), streamErr
	}
	return content, streamErr
}
