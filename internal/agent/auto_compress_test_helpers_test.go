package agent

import (
	"bytes"
	"context"
	"encoding/json"
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
	capturedChatSystemPrompt     string
	capturedChatActiveContext    int
	capturedChatResponseID       string
	capturedChatUpdateMode       string
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
	capturedCompactUpdateMode    string
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

func (m *compressionTestProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	m.chatCalls++
	m.capturedChatModel = model
	m.capturedChatSystemPrompt = systemPrompt
	m.capturedChatHistory = append([]api.Message(nil), history...)
	m.capturedChatActiveContext = len(api.ActiveContextBlocksFromContext(ctx))
	m.capturedChatResponseID = m.responseID
	m.capturedChatUpdateMode = api.AssistantUpdateModeFromContext(ctx)
	if m.chatErr != nil {
		return "", m.chatErr
	}
	if m.summary != "" {
		return compressionSummaryResponseForHistory(history, m.summary), nil
	}
	return compressionSummaryResponseForHistory(history, "summary"), nil
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
	m.capturedCompactUpdateMode = api.AssistantUpdateModeFromContext(ctx)
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

func compressionSummaryContinuationJSON(summary string) string {
	trimmed := strings.TrimSpace(summary)
	if strings.HasPrefix(trimmed, "{") {
		return summary
	}
	payload := map[string]any{
		"schema_version":            "xelyon.continuation.v1",
		"goal":                      summary,
		"acceptance_criteria":       []string{},
		"explicit_constraints":      []string{},
		"material_assumptions":      []string{},
		"decisions":                 []map[string]any{},
		"files_changed":             []map[string]any{},
		"verification":              []map[string]any{},
		"open_work":                 []string{},
		"blockers":                  []string{},
		"do_not_repeat":             []string{},
		"relevant_instruction_refs": []string{},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func compressionSummaryResponseForHistory(history []api.Message, response string) string {
	if isCompressionSummaryRequest(history) {
		return compressionSummaryContinuationJSON(response)
	}
	return response
}

func isCompressionSummaryRequest(history []api.Message) bool {
	return len(history) == 1 &&
		history[0].Role == "user" &&
		strings.Contains(history[0].Content, "xelyon.continuation.v1")
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
