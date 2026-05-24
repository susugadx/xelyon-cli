package agent

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/ledger"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestProviderHistoryRehydrateContextDefaultOffDoesNotChangeActiveContext(t *testing.T) {
	agent, _, _ := newProviderHistoryRehydrateContextFixture(t, currentTaskStateOpenAIResponses)
	agent.Runtime.Options.EnableProviderHistoryRehydrateContext = false

	requestCtx, projected := agent.providerFacingHistoryForRequest(agent.requestContext(context.Background()))

	if blocks := api.ActiveContextBlocksFromContext(requestCtx); blocks != nil {
		t.Fatalf("active context blocks = %#v, want nil when rehydrate gate is off", blocks)
	}
	if !providerHistoryMessagesContainReductionPlaceholder(projected) {
		t.Fatalf("projected history missing reduction placeholder: %#v", projected)
	}
}

func TestProviderHistoryRehydrateContextTokenBudgetGateSkipsDefaultOff(t *testing.T) {
	agent, _, _ := newProviderHistoryRehydrateContextFixture(t, currentTaskStateOpenAIResponses)
	agent.Runtime.Options.EnableProviderHistoryRehydrateContext = false
	agent.Runtime.Options.EnableCurrentTaskStateContext = true

	if agent.shouldBuildProviderHistoryRehydratedEvidenceActiveContext() {
		t.Fatal("shouldBuildProviderHistoryRehydratedEvidenceActiveContext() = true, want false when gate is off")
	}
	blocks := agent.providerFacingActiveContextBlocksForTokenBudget(context.Background())
	if len(blocks) != 1 || blocks[0].Name != currentTaskStateActiveContextName {
		t.Fatalf("token budget active context blocks = %#v, want current task state only", blocks)
	}
}

func TestProviderHistoryRehydrateContextTokenBudgetGateSkipsUnsupportedProvider(t *testing.T) {
	agent, _, _ := newProviderHistoryRehydrateContextFixture(t, currentTaskStateGemini)
	agent.Runtime.Options.EnableCurrentTaskStateContext = true

	if agent.shouldBuildProviderHistoryRehydratedEvidenceActiveContext() {
		t.Fatal("shouldBuildProviderHistoryRehydratedEvidenceActiveContext() = true, want false for unsupported provider")
	}
	if blocks := agent.providerFacingActiveContextBlocksForTokenBudget(context.Background()); blocks != nil {
		t.Fatalf("token budget active context blocks = %#v, want nil for unsupported provider", blocks)
	}
}

func TestProviderHistoryRehydrateContextAppendsRehydratedEvidenceBlock(t *testing.T) {
	agent, _, _ := newProviderHistoryRehydrateContextFixture(t, currentTaskStateOpenAIResponses)

	requestCtx, _ := agent.providerFacingHistoryForRequest(agent.requestContext(context.Background()))

	blocks := api.ActiveContextBlocksFromContext(requestCtx)
	if len(blocks) != 1 {
		t.Fatalf("active context blocks = %#v, want one rehydrated evidence block", blocks)
	}
	if blocks[0].Name != providerHistoryRehydratedEvidenceActiveContextName {
		t.Fatalf("active context name = %q, want %q", blocks[0].Name, providerHistoryRehydratedEvidenceActiveContextName)
	}
	for _, want := range []string{
		ledger.RehydratedEvidenceStartMarker,
		"RehydratedEvidence:",
		"- path: README.md",
		"  range: L1-L3",
		"  source: read_file",
		"  reason: edit_target_missing_recent_evidence",
		"  tool_call_id: call_rehydrate_ctx",
		"    L1: current one",
		"    L3: current three",
		ledger.RehydratedEvidenceEndMarker,
	} {
		if !strings.Contains(blocks[0].Content, want) {
			t.Fatalf("rehydrated active context missing %q:\n%s", want, blocks[0].Content)
		}
	}
}

func TestProviderHistoryRehydrateContextAppendsAfterCurrentTaskState(t *testing.T) {
	agent, _, _ := newProviderHistoryRehydrateContextFixture(t, currentTaskStateOpenAIResponses)
	agent.Runtime.Options.EnableCurrentTaskStateContext = true

	requestCtx, _ := agent.providerFacingHistoryForRequest(agent.requestContext(context.Background()))

	blocks := api.ActiveContextBlocksFromContext(requestCtx)
	if len(blocks) != 2 {
		t.Fatalf("active context blocks = %#v, want current task state plus rehydrated evidence", blocks)
	}
	if blocks[0].Name != currentTaskStateActiveContextName || blocks[1].Name != providerHistoryRehydratedEvidenceActiveContextName {
		t.Fatalf("active context names = %#v, want current task state then rehydrated evidence", blocks)
	}
}

func TestProviderHistoryRehydrateContextDoesNotMutateRawHistoryOrSession(t *testing.T) {
	agent, session, oldRead := newProviderHistoryRehydrateContextFixture(t, currentTaskStateOpenAIResponses)
	for _, msg := range agent.History {
		session.AddMessageFromAPI(msg, agent.CurrentModel)
	}
	beforeHistory := api.CloneMessages(agent.History)
	beforeSessionMessages := append([]history.MessageEntry(nil), session.Messages...)

	_, _ = agent.providerFacingHistoryForRequest(agent.requestContext(context.Background()))

	if !reflect.DeepEqual(agent.History, beforeHistory) {
		t.Fatalf("Agent.History changed after rehydrate injection:\n got %#v\nwant %#v", agent.History, beforeHistory)
	}
	if !reflect.DeepEqual(session.Messages, beforeSessionMessages) {
		t.Fatalf("session.Messages changed after rehydrate injection:\n got %#v\nwant %#v", session.Messages, beforeSessionMessages)
	}
	if agent.History[1].Content != oldRead || session.Messages[1].Content != oldRead {
		t.Fatalf("raw old read changed: history=%q session=%q want %q", agent.History[1].Content, session.Messages[1].Content, oldRead)
	}
}

func TestProviderHistoryRehydrateContextTokenEstimateMatchesProviderRequest(t *testing.T) {
	agent, _, _ := newProviderHistoryRehydrateContextFixture(t, currentTaskStateOpenAIResponses)
	staleReport := ProviderHistoryProjectionReport{Mode: ProviderHistoryReductionApply, OriginalMessageCount: 99}
	agent.Runtime.LastProviderHistoryProjectionReport = staleReport

	estimatedActive := agent.EstimateActiveContextTokens()
	estimatedTotal := agent.EstimateTokens()
	assertLastProviderHistoryProjectionReportPreserved(t, agent.Runtime, staleReport)

	requestCtx, projected := agent.providerFacingHistoryForRequest(agent.requestContext(context.Background()))
	blocks := api.ActiveContextBlocksFromContext(requestCtx)
	if len(blocks) != 1 || blocks[0].Name != providerHistoryRehydratedEvidenceActiveContextName {
		t.Fatalf("active context blocks = %#v, want one rehydrated evidence block", blocks)
	}

	wantActive := token.EstimateTokenCountForModel(agent.CurrentModel, blocks[0].Content)
	if estimatedActive != wantActive {
		t.Fatalf("EstimateActiveContextTokens() = %d, want request active context tokens %d", estimatedActive, wantActive)
	}
	wantTotal := token.EstimateTokenCountForModel(agent.CurrentModel, agent.SystemPrompt) +
		estimateTokens(agent.CurrentModel, projected) +
		wantActive
	if estimatedTotal != wantTotal {
		t.Fatalf("EstimateTokens() = %d, want provider-facing history plus active context %d", estimatedTotal, wantTotal)
	}
}

func TestHandleTokensCommand_ShowsProviderHistoryRehydratedEvidenceTokens(t *testing.T) {
	var out bytes.Buffer
	agent, _, _ := newProviderHistoryRehydrateContextFixture(t, currentTaskStateOpenAIResponses)
	agent.Runtime.UI = ui.NewRuntime(strings.NewReader(""), &out, &out)

	activeTokens := agent.EstimateActiveContextTokens()
	if activeTokens <= 0 {
		t.Fatalf("EstimateActiveContextTokens() = %d, want rehydrated evidence tokens", activeTokens)
	}
	if !handleTokensCommand(agent) {
		t.Fatal("handleTokensCommand() = false, want true")
	}
	output := out.String()
	for _, want := range []string{"Active Context:", formatNumber(activeTokens)} {
		if !strings.Contains(output, want) {
			t.Fatalf("/tokens output missing %q:\n%s", want, output)
		}
	}
}

func TestProviderHistoryRehydrateContextOnlyForResponsesProviders(t *testing.T) {
	tests := []struct {
		name    string
		fixture currentTaskStateProviderFixture
		want    bool
	}{
		{name: "openai responses", fixture: currentTaskStateOpenAIResponses, want: true},
		{name: "azure responses", fixture: currentTaskStateAzureResponses, want: true},
		{name: "openai chat completions", fixture: currentTaskStateOpenAIChatCompletions},
		{name: "gemini", fixture: currentTaskStateGemini},
		{name: "claude", fixture: currentTaskStateClaude},
		{name: "deepseek", fixture: currentTaskStateDeepSeek},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, _, _ := newProviderHistoryRehydrateContextFixture(t, tt.fixture)

			requestCtx, _ := agent.providerFacingHistoryForRequest(agent.requestContext(context.Background()))

			blocks := api.ActiveContextBlocksFromContext(requestCtx)
			if tt.want {
				if len(blocks) != 1 || blocks[0].Name != providerHistoryRehydratedEvidenceActiveContextName {
					t.Fatalf("active context blocks = %#v, want rehydrated evidence for %s", blocks, tt.name)
				}
				return
			}
			if blocks != nil {
				t.Fatalf("active context blocks = %#v, want nil for %s", blocks, tt.name)
			}
		})
	}
}

func newProviderHistoryRehydrateContextFixture(t *testing.T, fixture currentTaskStateProviderFixture) (*Agent, *history.Session, string) {
	t.Helper()
	root := t.TempDir()
	path := "README.md"
	writeProviderHistoryRehydrateContextFile(t, root, path, "current one\ncurrent two\ncurrent three\n")
	store := ledger.NewStoreWithRoot(root)
	store.Recorder().RecordToolObservation(ledger.ToolObservation{
		ToolName:   "read_file",
		ToolCallID: "call_rehydrate_ctx",
		Structured: &tools.RuntimeObservation{
			Evidence: []tools.ObservationEvidence{{
				Path:         path,
				ResolvedPath: filepath.Join(root, filepath.FromSlash(path)),
				StartLine:    1,
				EndLine:      3,
				Excerpt:      "old evidence excerpt",
			}},
		},
	})
	store.RecordEditReadinessObservation(ledger.EditReadinessObservation{
		Path:           path,
		NormalizedPath: path,
		Status:         ledger.EditReadinessStatusWarning,
		Reasons:        []ledger.EditReadinessReason{ledger.EditReadinessReasonNoRecentRead},
	})

	oldRead := strings.Repeat("old read_file output that should be replaced\n", 12)
	session := history.NewSession(fixture.model)
	agent := &Agent{
		Runtime: &AgentRuntime{
			Options: RuntimeOptions{
				EnableProviderHistoryReduction:        true,
				EnableProviderHistoryRehydrateContext: true,
			},
			TaskLedger: store,
		},
		History: providerHistoryReductionRequestHistory("call_rehydrate_ctx", oldRead),
		agentConversationState: agentConversationState{
			session: session,
		},
	}
	applyCurrentTaskStateProviderFixture(agent, fixture)
	return agent, session, oldRead
}

func writeProviderHistoryRehydrateContextFile(t *testing.T, root, relativePath, content string) {
	t.Helper()
	fullPath := filepath.Join(root, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(fullPath), err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", fullPath, err)
	}
}
