package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	promptnormal "github.com/susugadx/xelyon-cli/internal/prompt/normal"
)

func TestBuildFullInputItems_PrePrunesToolResults(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)

	large := makeLargeContent(60)
	agent.History = []api.Message{
		{Role: "user", Content: "turn 1"},
		{Role: "assistant", Content: "a1"},
		{Role: "tool", Content: large, ToolCallID: "c1", ToolName: "search_code"},
		{Role: "user", Content: "turn 2"},
		{Role: "assistant", Content: "a2"},
		{Role: "user", Content: "turn 3"},
		{Role: "assistant", Content: "a3"},
		{Role: "user", Content: "turn 4"},
		{Role: "assistant", Content: "a4"},
		{Role: "user", Content: "turn 5"},
		{Role: "assistant", Content: "a5"},
	}

	items := agent.buildFullInputItems()

	if len(items) == 0 {
		t.Fatal("buildFullInputItems() returned empty")
	}

	foundTruncated := false
	for _, item := range items {
		if item.Type == "function_call_output" && strings.Contains(item.Output, "truncated") {
			foundTruncated = true
			break
		}
	}
	if !foundTruncated {
		t.Error("buildFullInputItems() should pre-prune old tool results (expected truncated marker)")
	}
	if strings.Contains(agent.History[2].Content, "truncated") {
		t.Error("original History should not be modified by buildFullInputItems()")
	}
}

func TestBuildFullInputItems_CompactedModeBypassesPrune(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)

	agent.isCompactedMode = true
	agent.compactedItems = []api.InputItem{
		{Type: "compacted", Data: "compressed-data"},
	}
	agent.History = []api.Message{{Role: "user", Content: "new turn"}}

	items := agent.buildFullInputItems()

	if len(items) != 2 {
		t.Fatalf("expected compacted item plus current history, got %d", len(items))
	}
	if items[0].Type != "compacted" {
		t.Errorf("expected type=compacted, got %q", items[0].Type)
	}
	if items[1].Role != "user" || items[1].Content != "new turn" {
		t.Fatalf("items[1] = %#v, want current history item", items[1])
	}
}

func TestBuildFullInputItems_UsesSessionHistoryWithoutNormalModePrompt(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	t.Cleanup(agent.Cleanup)

	raw := "normal request"
	agent.History = []api.Message{{Role: "user", Content: raw + promptnormal.NormalModePrompt}}
	agent.session.AddMessage("user", raw, agent.CurrentModel)

	items := agent.buildFullInputItems()

	if len(items) != 1 {
		t.Fatalf("len(buildFullInputItems()) = %d, want 1", len(items))
	}
	if items[0].Role != "user" || items[0].Content != raw {
		t.Fatalf("items[0] = %#v, want raw user content", items[0])
	}
}

func TestRequestContext_IncludesCompactedInputItems(t *testing.T) {
	provider := &compressionTestProvider{name: "openai", supportsCompact: true}
	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", config.DefaultConfig())
	agent.isCompactedMode = true
	agent.compactedItems = []api.InputItem{{Type: "compacted", Data: "compact-data"}}

	got := api.CompactedInputItemsFromContext(agent.requestContext(context.Background()))
	if len(got) != 1 {
		t.Fatalf("len(CompactedInputItemsFromContext()) = %d, want 1", len(got))
	}
	if got[0].Type != "compacted" || got[0].Data != "compact-data" {
		t.Fatalf("compacted input item = %#v, want compact-data", got[0])
	}
}

func TestCompressWithCompactAPI_DoesNotSendCurrentTaskStateActiveContext(t *testing.T) {
	provider := &compressionTestProvider{name: "openai", supportsCompact: true}
	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", config.DefaultConfig())
	agent.Runtime.Options.EnableCurrentTaskStateContext = true
	agent.Runtime.TaskLedger = newTaskLedgerWithPassedTest(t)
	agent.History = []api.Message{{Role: "user", Content: "message"}}

	if err := agent.CompressWithCompactAPI(context.Background()); err != nil {
		t.Fatalf("CompressWithCompactAPI() error = %v", err)
	}
	if provider.capturedCompactActiveContext != 0 {
		t.Fatalf("compact active context count = %d, want 0", provider.capturedCompactActiveContext)
	}
	if agent.Runtime.TaskLedger.Snapshot().IsEmpty() {
		t.Fatal("CompressWithCompactAPI() should not reset the runtime task ledger")
	}
}
