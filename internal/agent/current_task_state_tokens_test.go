package agent

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/token"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestEstimateTokens_IncludesModelFacingCurrentTaskStateContext(t *testing.T) {
	agent := &Agent{
		CurrentModel:      "gpt-5.4",
		ProviderName:      "openai",
		ProviderConfigKey: "openai",
		SystemPrompt:      "system",
		Runtime: &AgentRuntime{
			Options:    RuntimeOptions{EnableCurrentTaskStateContext: true},
			TaskLedger: newTaskLedgerWithPassedTest(t),
		},
		History: []api.Message{{Role: "user", Content: "hello"}},
	}

	blocks := api.ActiveContextBlocksFromContext(agent.requestContext(context.Background()))
	if len(blocks) != 1 {
		t.Fatalf("len(ActiveContextBlocksFromContext()) = %d, want 1", len(blocks))
	}

	wantActive := token.EstimateTokenCountForModel(agent.CurrentModel, blocks[0].Content)
	if got := agent.EstimateActiveContextTokens(); got != wantActive {
		t.Fatalf("EstimateActiveContextTokens() = %d, want %d", got, wantActive)
	}
	wantTotal := token.EstimateTokenCountForModel(agent.CurrentModel, agent.SystemPrompt) +
		token.EstimateTokenCountForModel(agent.CurrentModel, "hello") +
		wantActive
	if got := agent.EstimateTokens(); got != wantTotal {
		t.Fatalf("EstimateTokens() = %d, want system + history + active context = %d", got, wantTotal)
	}
}

func TestEstimateTokens_DoesNotCountCurrentTaskStateForProvidersThatDoNotConsumeIt(t *testing.T) {
	agent := &Agent{
		CurrentModel:      activeContextUnsupported.model,
		ProviderName:      activeContextUnsupported.providerName,
		ProviderConfigKey: activeContextUnsupported.providerConfigKey,
		SystemPrompt:      "system",
		Runtime: &AgentRuntime{
			Options:    RuntimeOptions{EnableCurrentTaskStateContext: true},
			TaskLedger: newTaskLedgerWithPassedTest(t),
		},
		History: []api.Message{{Role: "user", Content: "hello"}},
	}

	if got := agent.EstimateActiveContextTokens(); got != 0 {
		t.Fatalf("EstimateActiveContextTokens() = %d, want 0 for unsupported active-context transport", got)
	}
	wantTotal := token.EstimateTokenCountForModel(agent.CurrentModel, agent.SystemPrompt) +
		token.EstimateTokenCountForModel(agent.CurrentModel, "hello")
	if got := agent.EstimateTokens(); got != wantTotal {
		t.Fatalf("EstimateTokens() = %d, want system + history = %d", got, wantTotal)
	}
}

func TestHandleTokensCommand_ShowsCurrentTaskStateContextTokens(t *testing.T) {
	var out bytes.Buffer
	agent := &Agent{
		CurrentModel:      "gpt-5.4",
		ProviderName:      "openai",
		ProviderConfigKey: "openai",
		Runtime: &AgentRuntime{
			Options:    RuntimeOptions{EnableCurrentTaskStateContext: true},
			TaskLedger: newTaskLedgerWithPassedTest(t),
			UI:         ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
	}

	if !handleTokensCommand(agent) {
		t.Fatal("handleTokensCommand() = false, want true")
	}
	if !strings.Contains(out.String(), "Active Context:") {
		t.Fatalf("tokens output missing active context row:\n%s", out.String())
	}
}
