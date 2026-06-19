package agent

import (
	"context"
	"fmt"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// mockProvider は api.Provider の軽量モック実装です。
type mockProvider struct {
	name string
}

func newAgentChatTestAgent(t *testing.T, provider api.Provider) *Agent {
	t.Helper()

	runtime := NewAgentRuntimeWithConfig(newProjectMapDisabledConfig())
	agent := NewAgentWithRuntime("test-model", provider, false, runtime)
	t.Cleanup(agent.Cleanup)
	return agent
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) SupportsImages() bool {
	return false
}

func (m *mockProvider) IsFunctionCallingEnabled() bool {
	return true
}

func (m *mockProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	return compressionSummaryResponseForHistory(history, "mock response"), nil
}

func (m *mockProvider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	return "mock image response", nil
}

// sequenceMockProvider は呼び出しごとに異なるレスポンスを返すモックです。
type sequenceMockProvider struct {
	name          string
	responses     []string
	contexts      []context.Context
	systemPrompts []string
	histories     [][]api.Message
	callCount     int
}

func (m *sequenceMockProvider) Name() string { return m.name }

func (m *sequenceMockProvider) SupportsImages() bool { return false }

func (m *sequenceMockProvider) IsFunctionCallingEnabled() bool { return true }

func (m *sequenceMockProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	m.contexts = append(m.contexts, ctx)
	m.systemPrompts = append(m.systemPrompts, systemPrompt)
	m.histories = append(m.histories, append([]api.Message(nil), history...))
	if m.callCount >= len(m.responses) {
		return compressionSummaryResponseForHistory(history, m.responses[len(m.responses)-1]), nil
	}
	resp := m.responses[m.callCount]
	m.callCount++
	return compressionSummaryResponseForHistory(history, resp), nil
}

func (m *sequenceMockProvider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	return m.ChatWithTools(ctx, systemPrompt, history, model)
}

// blockingCancelProvider は context cancel までブロックするモックです。
type blockingCancelProvider struct {
	started chan struct{}
}

func (m *blockingCancelProvider) Name() string { return "blocking-cancel" }

func (m *blockingCancelProvider) SupportsImages() bool { return false }

func (m *blockingCancelProvider) IsFunctionCallingEnabled() bool { return false }

func (m *blockingCancelProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	if m.started != nil {
		select {
		case <-m.started:
		default:
			close(m.started)
		}
	}
	<-ctx.Done()
	return "", fmt.Errorf("API call failed: %w", ctx.Err())
}

func (m *blockingCancelProvider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	return m.ChatWithTools(ctx, systemPrompt, history, model)
}
