package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/providerhistory"
	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestNormalModeRequestApplyCompactsMCPResultAndInjectsRawOutputContext(t *testing.T) {
	agent, provider, store := newProviderHistoryRawOutputRequestAgent(t)
	mcpOutput := providerHistoryLargeSafeMCPResult()
	agent.Runtime.RawOutputArtifactStore = store
	configureProviderHistoryRawOutputRequestApply(agent, 4096, 8192)
	agent.History = []api.Message{
		{Role: "user", Content: "inspect mcp history"},
		providerHistoryAssistantToolCall("call_mcp_docs", "mcp_context7_get_library_docs"),
		providerHistoryToolResult("call_mcp_docs", "mcp_context7_get_library_docs", mcpOutput),
		{Role: "assistant", Content: "mcp data reviewed"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest read"),
		{Role: "assistant", Content: "ready"},
	}
	syncProviderHistoryRawOutputRequestSession(agent)

	if err := agent.chatInternal("show safe documentation result", nil); err != nil {
		t.Fatalf("chatInternal() error = %v", err)
	}

	projected := provider.capturedHistory[2].Content
	if projected == mcpOutput ||
		!strings.Contains(projected, "[compacted old MCP tool result;") ||
		!strings.Contains(projected, "raw_output_ref=") {
		t.Fatalf("provider MCP output = %q, want artifact-backed placeholder", projected)
	}
	if len(provider.capturedActiveContextBlocks) != 1 {
		t.Fatalf("active context blocks = %#v, want one raw output block", provider.capturedActiveContextBlocks)
	}
	block := provider.capturedActiveContextBlocks[0]
	for _, want := range []string{
		"surface: mcp_tool_result",
		"tool_name: mcp_context7_get_library_docs",
		"family: mcp",
		"classifier: mcp_json_result",
		"safe documentation result",
	} {
		if !strings.Contains(block.Content, want) {
			t.Fatalf("raw output active context missing %q:\n%s", want, block.Content)
		}
	}
	report := agent.Runtime.LastProviderHistoryProjectionReport
	if report.ReplacedCount != 1 ||
		report.RawOutputRefCount != 1 ||
		report.DataBearingCandidateCount != 1 ||
		!report.ResponsesChainDisabled {
		t.Fatalf("LastProviderHistoryProjectionReport = %#v, want applied MCP artifact report", report)
	}
}

func TestNormalModeRequestInjectsRuntimeCompactedMCPRawOutputContextWithoutReprojection(t *testing.T) {
	agent, provider, store := newProviderHistoryRawOutputRequestAgent(t)
	rawMCPOutput := providerHistoryNumberedLines("runtime-mcp-target", 6000)
	agent.Runtime.RawOutputArtifactStore = store
	configureProviderHistoryRawOutputRequestApply(agent, 4096, 8192)
	created, err := store.Create(context.Background(), rawoutputs.CreateRequest{
		Surface:   rawoutputs.SurfaceMCPToolResult,
		SessionID: agent.session.ID,
		Source: rawoutputs.SourceMetadata{
			Provider:   agent.ProviderName,
			Model:      agent.CurrentModel,
			ToolName:   "mcp_context7_get_library_docs",
			ToolCallID: "call_mcp_runtime",
			EventID:    "tool_call:call_mcp_runtime",
		},
		Classification: rawoutputs.ClassificationMetadata{
			SemanticRole: "data_bearing",
			Family:       "mcp",
			Classifier:   "mcp_runtime_large_result",
		},
		Body:          strings.NewReader(rawMCPOutput),
		SizeHintBytes: int64(len(rawMCPOutput)),
	})
	if err != nil {
		t.Fatalf("Create(runtime MCP raw output) error = %v", err)
	}
	placeholder := buildMCPRuntimeResultPlaceholder(created.Ref, "", rawMCPOutput)
	agent.History = []api.Message{
		{Role: "user", Content: "inspect runtime mcp history"},
		providerHistoryAssistantToolCall("call_mcp_runtime", "mcp_context7_get_library_docs"),
		providerHistoryToolResult("call_mcp_runtime", "mcp_context7_get_library_docs", placeholder),
		{Role: "assistant", Content: "runtime mcp data reviewed"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest read"),
		{Role: "assistant", Content: "ready"},
	}
	syncProviderHistoryRawOutputRequestSession(agent)

	if err := agent.chatInternal("show runtime-mcp-target-3000", nil); err != nil {
		t.Fatalf("chatInternal() error = %v", err)
	}

	if got := provider.capturedHistory[2].Content; got != placeholder {
		t.Fatalf("provider MCP placeholder changed:\n got %q\nwant %q", got, placeholder)
	}
	if provider.capturedResponseIDChainDisabled {
		t.Fatal("response ID chain disabled despite context-only runtime MCP placeholder")
	}
	if len(provider.capturedActiveContextBlocks) != 1 {
		t.Fatalf("active context blocks = %#v, want one raw output block", provider.capturedActiveContextBlocks)
	}
	block := provider.capturedActiveContextBlocks[0]
	for _, want := range []string{
		"surface: mcp_tool_result",
		"tool_name: mcp_context7_get_library_docs",
		"family: mcp",
		"classifier: mcp_runtime_large_result",
		"matched raw output excerpt",
		"runtime-mcp-target-3000",
	} {
		if !strings.Contains(block.Content, want) {
			t.Fatalf("runtime MCP raw output active context missing %q:\n%s", want, block.Content)
		}
	}
	report := agent.Runtime.LastProviderHistoryProjectionReport
	if report.ReplacedCount != 0 ||
		report.RawOutputRefCount != 0 ||
		report.RawOutputContextRefCount != 1 ||
		report.RawOutputContextRefs[0].RefID != created.Ref.RefID ||
		report.ResponsesChainDisabled {
		t.Fatalf("LastProviderHistoryProjectionReport = %#v, want context-only runtime MCP ref", report)
	}
}

func TestNormalModeRequestInjectsLatestRuntimeCompactedMCPRawOutputContext(t *testing.T) {
	agent, provider, store := newProviderHistoryRawOutputRequestAgent(t)
	rawMCPOutput := providerHistoryNumberedLines("runtime-mcp-latest-target", 6000)
	agent.Runtime.RawOutputArtifactStore = store
	configureProviderHistoryRawOutputRequestApply(agent, 4096, 8192)

	execResult := agent.guardMCPToolExecutionResult(context.Background(), &tools.ToolCall{
		ID:   "call_mcp_runtime_latest",
		Tool: "mcp_context7_get_library_docs",
	}, tools.ExecutionResult{Result: rawMCPOutput})
	placeholder := execResult.Result
	if placeholder == rawMCPOutput ||
		!strings.Contains(placeholder, "[compacted MCP tool result;") ||
		!strings.Contains(placeholder, "raw_output_ref=") {
		t.Fatalf("runtime MCP placeholder = %q, want compacted result with raw ref", placeholder)
	}
	agent.History = []api.Message{
		{Role: "user", Content: "inspect latest runtime mcp history"},
		providerHistoryAssistantToolCall("call_mcp_runtime_latest", "mcp_context7_get_library_docs"),
		providerHistoryToolResult("call_mcp_runtime_latest", "mcp_context7_get_library_docs", placeholder),
	}
	syncProviderHistoryRawOutputRequestSession(agent)

	if err := agent.chatInternal("show runtime-mcp-latest-target-3000", nil); err != nil {
		t.Fatalf("chatInternal() error = %v", err)
	}

	if got := provider.capturedHistory[2].Content; got != placeholder {
		t.Fatalf("provider MCP placeholder changed:\n got %q\nwant %q", got, placeholder)
	}
	if provider.capturedResponseIDChainDisabled {
		t.Fatal("response ID chain disabled despite context-only latest runtime MCP placeholder")
	}
	if len(provider.capturedActiveContextBlocks) != 1 {
		t.Fatalf("active context blocks = %#v, want one raw output block", provider.capturedActiveContextBlocks)
	}
	block := provider.capturedActiveContextBlocks[0]
	for _, want := range []string{
		"surface: mcp_tool_result",
		"tool_name: mcp_context7_get_library_docs",
		"family: mcp",
		"classifier: mcp_runtime_large_result",
		"matched raw output excerpt",
		"runtime-mcp-latest-target-3000",
	} {
		if !strings.Contains(block.Content, want) {
			t.Fatalf("latest runtime MCP raw output active context missing %q:\n%s", want, block.Content)
		}
	}
	report := agent.Runtime.LastProviderHistoryProjectionReport
	if report.ReplacedCount != 0 ||
		report.RawOutputRefCount != 0 ||
		report.RawOutputContextRefCount != 1 ||
		len(report.RawOutputContextRefs) != 1 ||
		report.RawOutputContextRefs[0].RefID == "" ||
		!strings.Contains(placeholder, "raw_output_ref="+report.RawOutputContextRefs[0].RefID) ||
		report.ResponsesChainDisabled {
		t.Fatalf("LastProviderHistoryProjectionReport = %#v, want context-only latest runtime MCP ref", report)
	}
}

func TestTokenBudgetIncludesLatestRuntimeCompactedMCPRawOutputContext(t *testing.T) {
	agent, _, store := newProviderHistoryRawOutputRequestAgent(t)
	countingStore := &countingRawOutputArtifactStore{inner: store}
	rawMCPOutput := providerHistoryNumberedLines("runtime-mcp-budget-target", 6000)
	agent.Runtime.RawOutputArtifactStore = countingStore
	configureProviderHistoryRawOutputRequestApply(agent, 4096, 8192)

	execResult := agent.guardMCPToolExecutionResult(context.Background(), &tools.ToolCall{
		ID:   "call_mcp_runtime_budget",
		Tool: "mcp_context7_get_library_docs",
	}, tools.ExecutionResult{Result: rawMCPOutput})
	placeholder := execResult.Result
	if placeholder == rawMCPOutput || !strings.Contains(placeholder, "raw_output_ref=") {
		t.Fatalf("runtime MCP placeholder = %q, want compacted result with raw ref", placeholder)
	}
	countingStore.createCalls = 0
	countingStore.verifyCalls = 0
	countingStore.scanCalls = 0
	countingStore.lookupCalls = 0
	agent.History = []api.Message{
		{Role: "user", Content: "show runtime-mcp-budget-target-3000"},
		providerHistoryAssistantToolCall("call_mcp_runtime_budget", "mcp_context7_get_library_docs"),
		providerHistoryToolResult("call_mcp_runtime_budget", "mcp_context7_get_library_docs", placeholder),
	}
	syncProviderHistoryRawOutputRequestSession(agent)

	blocks := agent.providerFacingActiveContextBlocksForTokenBudget(context.Background())
	if len(blocks) != 1 {
		t.Fatalf("token budget active context blocks = %#v, want one raw output block", blocks)
	}
	if !strings.Contains(blocks[0].Content, "runtime-mcp-budget-target-3000") ||
		!strings.Contains(blocks[0].Content, "classifier: mcp_runtime_large_result") {
		t.Fatalf("token budget raw output active context missing runtime MCP excerpt:\n%s", blocks[0].Content)
	}
	if countingStore.lookupCalls == 0 || countingStore.scanCalls == 0 {
		t.Fatalf("token budget raw output lookup/scan calls = lookup:%d scan:%d, want read-only rehydrate context", countingStore.lookupCalls, countingStore.scanCalls)
	}
	if countingStore.createCalls != 0 || countingStore.verifyCalls != 0 {
		t.Fatalf("token budget artifact create/verify calls = create:%d verify:%d, want read-only lookup/scan only", countingStore.createCalls, countingStore.verifyCalls)
	}
}

func TestNormalModeRequestOmitsRuntimeMCPResultWithoutRawOutputRef(t *testing.T) {
	tests := []struct {
		name       string
		tool       string
		output     string
		wantReason string
		forbidden  []string
	}{
		{
			name:       "private-looking output",
			tool:       "mcp_customer_export",
			output:     providerHistoryLargePrivateLookingMCPRuntimeResult(),
			wantReason: providerhistory.MCPSensitiveOrPrivateResultKeepReason,
		},
		{
			name: "sensitive output",
			tool: "mcp_secret_export",
			output: "prefix\n" +
				strings.Repeat("safe line\n", 7000) +
				"api_key=secret-value\nsuffix\n",
			wantReason: string(rawoutputs.ReasonSensitiveArtifactForbidden),
			forbidden:  []string{"api_key", "secret-value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, provider, _ := newProviderHistoryRawOutputRequestAgent(t)
			configureProviderHistoryRawOutputRequestApply(agent, 4096, 8192)
			execResult := agent.guardMCPToolExecutionResult(context.Background(), &tools.ToolCall{
				ID:   "call_mcp_no_ref_runtime",
				Tool: tt.tool,
			}, tools.ExecutionResult{Result: tt.output})
			placeholder := execResult.Result
			if placeholder == tt.output ||
				!strings.Contains(placeholder, "[compacted MCP tool result;") ||
				!strings.Contains(placeholder, "full_output_omitted_reason="+tt.wantReason+";") ||
				strings.Contains(placeholder, "raw_output_ref=") {
				t.Fatalf("runtime MCP placeholder = %q, want omitted placeholder without raw ref", placeholder)
			}
			for _, forbidden := range tt.forbidden {
				if strings.Contains(placeholder, forbidden) {
					t.Fatalf("runtime MCP placeholder leaked %q:\n%s", forbidden, placeholder)
				}
			}
			agent.History = []api.Message{
				{Role: "user", Content: "inspect omitted runtime mcp history"},
				providerHistoryAssistantToolCall("call_mcp_no_ref_runtime", tt.tool),
				providerHistoryToolResult("call_mcp_no_ref_runtime", tt.tool, placeholder),
				{Role: "assistant", Content: "runtime mcp data reviewed"},
				providerHistoryAssistantToolCall("call_latest", "read_file"),
				providerHistoryToolResult("call_latest", "read_file", "latest read"),
				{Role: "assistant", Content: "ready"},
			}
			syncProviderHistoryRawOutputRequestSession(agent)

			if err := agent.chatInternal("summarize omitted runtime mcp", nil); err != nil {
				t.Fatalf("chatInternal() error = %v", err)
			}

			projected := provider.capturedHistory[2].Content
			if projected != placeholder {
				t.Fatalf("provider MCP result changed:\n got %q\nwant %q", projected, placeholder)
			}
			for _, forbidden := range tt.forbidden {
				if strings.Contains(projected, forbidden) {
					t.Fatalf("provider MCP result leaked %q:\n%s", forbidden, projected)
				}
			}
			if len(provider.capturedActiveContextBlocks) != 0 {
				t.Fatalf("active context blocks = %#v, want none for non-persisted MCP result", provider.capturedActiveContextBlocks)
			}
			report := agent.Runtime.LastProviderHistoryProjectionReport
			if report.ReplacedCount != 0 ||
				report.RawOutputRefCount != 0 ||
				report.RawOutputContextRefCount != 0 ||
				report.ResponsesChainDisabled {
				t.Fatalf("LastProviderHistoryProjectionReport = %#v, want context-free omitted runtime MCP keep", report)
			}
		})
	}
}
