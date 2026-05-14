package agent

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type compressionTestProvider struct {
	name                         string
	summary                      string
	chatErr                      error
	chatCalls                    int
	capturedChatModel            string
	capturedChatHistory          []api.Message
	capturedChatActiveContext    int
	capturedChatResponseID       string
	cachedResponseID             bool
	responseID                   string
	serverCompactionLocalSkip    bool
	supportsCompact              bool
	compactErr                   error
	compactCalls                 int
	capturedCompactModel         string
	capturedCompactInput         []api.InputItem
	capturedCompactActiveContext int
	capturedCompactResponseID    string
	compactOutput                []api.InputItem
	supportsClaudeCompaction     bool
}

func (m *compressionTestProvider) Name() string {
	if m.name == "" {
		return "test"
	}
	return m.name
}

func (m *compressionTestProvider) SupportsImages() bool {
	return false
}

func (m *compressionTestProvider) IsFunctionCallingEnabled() bool {
	return false
}

func (m *compressionTestProvider) ChatWithTools(ctx context.Context, _ string, history []api.Message, model string) (string, error) {
	m.chatCalls++
	m.capturedChatModel = model
	m.capturedChatHistory = append([]api.Message(nil), history...)
	m.capturedChatActiveContext = len(api.ActiveContextBlocksFromContext(ctx))
	m.capturedChatResponseID = m.responseID
	if m.chatErr != nil {
		return "", m.chatErr
	}
	if m.summary != "" {
		return m.summary, nil
	}
	return "summary", nil
}

func (m *compressionTestProvider) ChatWithImage(_ context.Context, _ string, _ []api.Message, _ string, _ *api.ImageData, _ string) (string, error) {
	return "", nil
}

func (m *compressionTestProvider) HasCachedResponseID() bool {
	return m.cachedResponseID
}

func (m *compressionTestProvider) SetResponseID(id string) {
	m.responseID = id
	m.cachedResponseID = id != ""
}

func (m *compressionTestProvider) GetResponseID() string {
	return m.responseID
}

func (m *compressionTestProvider) ShouldSkipLocalAutoCompressionForServerCompaction() bool {
	return m.serverCompactionLocalSkip
}

func (m *compressionTestProvider) SupportsCompact() bool {
	return m.supportsCompact
}

func (m *compressionTestProvider) CompactHistory(ctx context.Context, input []api.InputItem, model, _ string) (*api.CompactResponse, error) {
	m.compactCalls++
	m.capturedCompactModel = model
	m.capturedCompactInput = api.CloneInputItems(input)
	m.capturedCompactActiveContext = len(api.ActiveContextBlocksFromContext(ctx))
	m.capturedCompactResponseID = m.responseID
	if m.compactErr != nil {
		return nil, m.compactErr
	}
	output := m.compactOutput
	if len(output) == 0 {
		output = []api.InputItem{{Type: "compacted", Data: "compressed"}}
	}
	return &api.CompactResponse{Output: output, Model: model}, nil
}

func (m *compressionTestProvider) SupportsClaudeCompaction() bool {
	return m.supportsClaudeCompaction
}

func (m *compressionTestProvider) SupportsClaudeCompactionWithContext(ctx context.Context, _ string) bool {
	cfg := config.FromContext(ctx)
	return m.supportsClaudeCompaction && cfg.Compression.ClaudeCompaction
}

func newCompressionTestAgent(t *testing.T, provider *compressionTestProvider, model string, cfg *config.Config) (*Agent, *bytes.Buffer) {
	t.Helper()

	var out bytes.Buffer
	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.UI = ui.NewRuntime(strings.NewReader(""), &out, &out)

	agent := NewAgentWithRuntime(model, provider, false, runtime)
	t.Cleanup(agent.Cleanup)
	return agent, &out
}

func oversizedCompressionHistory() []api.Message {
	return []api.Message{
		{Role: "user", Content: strings.Repeat("x", 260000)},
		{Role: "assistant", Content: "latest message"},
	}
}

func hugeCompressionHistory() []api.Message {
	return []api.Message{
		{Role: "user", Content: strings.Repeat("x", 400000)},
		{Role: "assistant", Content: "latest message"},
	}
}
