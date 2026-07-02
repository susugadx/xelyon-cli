package agent

import (
	"context"
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// mockErrorProvider は常にエラーを返すプロバイダー
type mockErrorProvider struct{}

func (m *mockErrorProvider) Name() string                   { return "test-error" }
func (m *mockErrorProvider) SupportsImages() bool           { return false }
func (m *mockErrorProvider) IsFunctionCallingEnabled() bool { return false }
func (m *mockErrorProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	return "", fmt.Errorf("mock error")
}
func (m *mockErrorProvider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	return "", fmt.Errorf("mock error")
}

type headlessToolSetProbeProvider struct {
	name         string
	systemPrompt string
	toolNames    []string
}

func (p *headlessToolSetProbeProvider) Name() string {
	if p.name != "" {
		return p.name
	}
	return "openai"
}

func (p *headlessToolSetProbeProvider) SupportsImages() bool { return false }

func (p *headlessToolSetProbeProvider) IsFunctionCallingEnabled() bool { return true }

func (p *headlessToolSetProbeProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	p.systemPrompt = systemPrompt
	defs := tools.RegistryFromContext(ctx).GetToolDefinitions()
	p.toolNames = make([]string, len(defs))
	for i, def := range defs {
		p.toolNames[i] = def.Name
	}
	return "done", nil
}

func (p *headlessToolSetProbeProvider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	return p.ChatWithTools(ctx, systemPrompt, history, model)
}

type headlessActiveContextProbeProvider struct {
	activeContextBlocks int
}

func (p *headlessActiveContextProbeProvider) Name() string { return "openai" }

func (p *headlessActiveContextProbeProvider) SupportsImages() bool { return false }

func (p *headlessActiveContextProbeProvider) IsFunctionCallingEnabled() bool { return true }

func (p *headlessActiveContextProbeProvider) ChatWithTools(ctx context.Context, _ string, _ []api.Message, _ string) (string, error) {
	p.activeContextBlocks = len(api.ActiveContextBlocksFromContext(ctx))
	return "done", nil
}

func (p *headlessActiveContextProbeProvider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	return p.ChatWithTools(ctx, systemPrompt, history, model)
}

type headlessUsageProvider struct {
	usageCallback api.UsageCallback
}

func (p *headlessUsageProvider) Name() string { return "openai" }

func (p *headlessUsageProvider) SupportsImages() bool { return false }

func (p *headlessUsageProvider) IsFunctionCallingEnabled() bool { return true }

func (p *headlessUsageProvider) SetUsageCallback(callback api.UsageCallback) {
	p.usageCallback = callback
}

func (p *headlessUsageProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	if p.usageCallback != nil {
		p.usageCallback(api.Usage{
			InputTokens:       1000,
			CachedInputTokens: 200,
			OutputTokens:      300,
			ThinkingTokens:    50,
		})
	}
	return "done", nil
}

func (p *headlessUsageProvider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	return p.ChatWithTools(ctx, systemPrompt, history, model)
}

type headlessWebSearchUsageProvider struct {
	usageCallback api.UsageCallback
}

func (p *headlessWebSearchUsageProvider) Name() string { return "kimi" }

func (p *headlessWebSearchUsageProvider) SupportsImages() bool { return false }

func (p *headlessWebSearchUsageProvider) IsFunctionCallingEnabled() bool { return true }

func (p *headlessWebSearchUsageProvider) SetUsageCallback(callback api.UsageCallback) {
	p.usageCallback = callback
}

func (p *headlessWebSearchUsageProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	if p.usageCallback != nil {
		p.usageCallback(api.Usage{
			StorageCost:           0.005,
			WebSearchCalls:        1,
			WebSearchResultTokens: 222,
		})
	}
	return "done", nil
}

func (p *headlessWebSearchUsageProvider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	return p.ChatWithTools(ctx, systemPrompt, history, model)
}

type headlessGeminiWebSearchUsageProvider struct {
	usageCallback api.UsageCallback
}

func (p *headlessGeminiWebSearchUsageProvider) Name() string { return "gemini" }

func (p *headlessGeminiWebSearchUsageProvider) SupportsImages() bool { return false }

func (p *headlessGeminiWebSearchUsageProvider) IsFunctionCallingEnabled() bool { return true }

func (p *headlessGeminiWebSearchUsageProvider) SetUsageCallback(callback api.UsageCallback) {
	p.usageCallback = callback
}

func (p *headlessGeminiWebSearchUsageProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	if p.usageCallback != nil {
		p.usageCallback(api.Usage{
			InputTokens:       17,
			OutputTokens:      5,
			ThinkingTokens:    3,
			CachedInputTokens: 4,
		})
	}
	return "done", nil
}

func (p *headlessGeminiWebSearchUsageProvider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	return p.ChatWithTools(ctx, systemPrompt, history, model)
}

type headlessHistoryProbeProvider struct {
	responses []string
	histories [][]api.Message
	callCount int
}

func (p *headlessHistoryProbeProvider) Name() string { return "gemini" }

func (p *headlessHistoryProbeProvider) SupportsImages() bool { return false }

func (p *headlessHistoryProbeProvider) IsFunctionCallingEnabled() bool { return true }

func (p *headlessHistoryProbeProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	p.histories = append(p.histories, cloneHeadlessHistory(history))
	if p.callCount >= len(p.responses) {
		return p.responses[len(p.responses)-1], nil
	}
	resp := p.responses[p.callCount]
	p.callCount++
	return resp, nil
}

func (p *headlessHistoryProbeProvider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	return p.ChatWithTools(ctx, systemPrompt, history, model)
}

func cloneHeadlessHistory(history []api.Message) []api.Message {
	return api.CloneMessages(history)
}
