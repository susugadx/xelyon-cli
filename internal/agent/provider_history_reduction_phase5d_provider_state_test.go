package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/providers/gemini"
)

func TestPhase5DGeminiFunctionResponsesUseProjectedToolNames(t *testing.T) {
	var captured map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode Gemini request: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(`data: {"candidates":[{"content":{"parts":[{"text":"done"}]}}]}` + "\n\n"))
	}))
	defer server.Close()
	t.Setenv("GEMINI_API_URL", server.URL)
	t.Setenv("GEMINI_CONTEXT_CACHING", "0")
	t.Setenv("GEMINI_FUNCTION_CALLING", "1")

	readOutput := phase5DOutput("old read result")
	searchOutput := phase5DOutput("old search result")
	gatherOutput := phase5DOutput("old gather result")
	agent := &Agent{
		Runtime: &AgentRuntime{TaskLedger: providerHistoryTaskLedgerWithEvidence(t,
			providerHistoryEvidenceItem{ToolName: "read_file", ToolCallID: "call_read", Path: "README.md", StartLine: 1},
			providerHistoryEvidenceItem{ToolName: "search_code", ToolCallID: "call_search", Path: "internal/search.go", StartLine: 7},
			providerHistoryEvidenceItem{ToolName: "gather_context", ToolCallID: "call_gather", Path: "internal/gather.go", StartLine: 9},
		)},
		History: []api.Message{
			providerHistoryAssistantToolCall("call_read", "read_file"),
			providerHistoryToolResult("call_read", "", readOutput),
			{Role: "assistant", Content: "read processed"},
			providerHistoryAssistantToolCall("call_search", "search_code"),
			providerHistoryToolResult("call_search", "", searchOutput),
			{Role: "assistant", Content: "search processed"},
			providerHistoryAssistantToolCall("call_gather", "gather_context"),
			providerHistoryToolResult("call_gather", "", gatherOutput),
			{Role: "assistant", Content: "gather processed"},
			providerHistoryAssistantToolCall("call_latest_write", "apply_patch"),
			providerHistoryToolResult("call_latest_write", "apply_patch", "latest raw patch output"),
			{Role: "assistant", Content: "done"},
		},
	}
	rawBefore := api.CloneMessages(agent.History)
	result := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionApply})

	if _, err := gemini.New("test-key").ChatWithTools(context.Background(), "system", result.History, "gemini-2.0-flash-exp"); err != nil {
		t.Fatalf("Gemini ChatWithTools() error = %v", err)
	}

	functionResponses := phase5DGeminiFunctionResponses(t, captured)
	for callID, toolName := range map[string]string{
		"call_read":   "read_file",
		"call_search": "search_code",
		"call_gather": "gather_context",
	} {
		response := functionResponses[toolName]
		if response == nil {
			t.Fatalf("Gemini request missing functionResponse.name %q: %#v", toolName, functionResponses)
		}
		resultText, _ := response["result"].(string)
		if !strings.HasPrefix(resultText, "[omitted old "+toolName+" result;") {
			t.Fatalf("Gemini %s response result = %q, want reduction placeholder", toolName, resultText)
		}
		if got := phase5DProjectedToolName(result.History, callID); got != toolName {
			t.Fatalf("projection ToolName for %s = %q, want %q", callID, got, toolName)
		}
	}
	for _, idx := range []int{1, 4, 7} {
		if agent.History[idx].ToolName != "" {
			t.Fatalf("raw Agent.History[%d].ToolName = %q, want unchanged empty", idx, agent.History[idx].ToolName)
		}
	}
	if !reflect.DeepEqual(agent.History, rawBefore) {
		t.Fatalf("Agent.History changed after Gemini request:\n got %#v\nwant %#v", agent.History, rawBefore)
	}
}

func TestPhase5DAnthropicProviderStateProjectionCloneIsolation(t *testing.T) {
	oldRead := phase5DOutput("old Claude read result")
	readResult := providerHistoryToolResult("call_claude_old", "read_file", oldRead)
	readResult.ReasoningContent = "provider reasoning"
	readResult.SetAnthropicContentBlocks([]api.AnthropicContentBlock{
		{Type: "thinking", Thinking: "private thought", Signature: "sig-1"},
		{Type: "tool_use", ID: "toolu_1", Name: "read_file", Input: map[string]any{"path": "README.md"}},
	})
	agent := &Agent{
		Runtime: &AgentRuntime{TaskLedger: providerHistoryTaskLedgerWithEvidence(t,
			providerHistoryEvidenceItem{ToolName: "read_file", ToolCallID: "call_claude_old", Path: "README.md", StartLine: 1, EndLine: 4},
		)},
		History: []api.Message{
			providerHistoryAssistantToolCall("call_claude_old", "read_file"),
			readResult,
			{Role: "assistant", Content: "after read"},
			providerHistoryAssistantToolCall("call_latest", "read_file"),
			providerHistoryToolResult("call_latest", "read_file", "latest raw output"),
			{Role: "assistant", Content: "done"},
		},
	}
	rawBefore := api.CloneMessages(agent.History)

	result := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionApply})

	if result.History[1].Role != "tool" || result.History[1].ToolCallID != "call_claude_old" || result.History[1].ToolName != "read_file" {
		t.Fatalf("projected tool message shape = %#v, want role/tool id/name preserved", result.History[1])
	}
	if !strings.HasPrefix(result.History[1].Content, "[omitted old read_file result;") {
		t.Fatalf("projected content = %q, want reduction placeholder", result.History[1].Content)
	}
	if result.History[1].ReasoningContent != "provider reasoning" {
		t.Fatalf("projected ReasoningContent = %q, want preserved provider metadata", result.History[1].ReasoningContent)
	}
	blocks := result.History[1].AnthropicContentBlocks()
	if len(blocks) != 2 || blocks[0].Thinking != "private thought" || blocks[1].Input["path"] != "README.md" {
		t.Fatalf("projected AnthropicContentBlocks = %#v, want preserved provider state", blocks)
	}
	blocks[0].Thinking = "mutated projected thought"
	blocks[1].Input["path"] = "mutated.go"
	result.History[1].SetAnthropicContentBlocks(blocks)
	result.History[1].Content = "provider mutated projection content"

	rawBlocks := agent.History[1].AnthropicContentBlocks()
	if agent.History[1].Content != oldRead ||
		rawBlocks[0].Thinking != "private thought" ||
		rawBlocks[1].Input["path"] != "README.md" {
		t.Fatalf("raw Agent.History leaked projected provider-state mutation: content=%q blocks=%#v", agent.History[1].Content, rawBlocks)
	}
	if !reflect.DeepEqual(agent.History, rawBefore) {
		t.Fatalf("Agent.History changed after Anthropic projection mutation:\n got %#v\nwant %#v", agent.History, rawBefore)
	}
}

func phase5DGeminiFunctionResponses(t *testing.T, captured map[string]any) map[string]map[string]any {
	t.Helper()
	contents, ok := captured["contents"].([]any)
	if !ok {
		t.Fatalf("Gemini contents = %#v, want []any", captured["contents"])
	}
	responses := make(map[string]map[string]any)
	for _, content := range contents {
		contentMap, ok := content.(map[string]any)
		if !ok {
			continue
		}
		parts, ok := contentMap["parts"].([]any)
		if !ok {
			continue
		}
		for _, part := range parts {
			partMap, ok := part.(map[string]any)
			if !ok {
				continue
			}
			responseMap, ok := partMap["functionResponse"].(map[string]any)
			if !ok {
				continue
			}
			name, _ := responseMap["name"].(string)
			body, _ := responseMap["response"].(map[string]any)
			if name != "" {
				responses[name] = body
			}
		}
	}
	return responses
}

func phase5DProjectedToolName(history []api.Message, callID string) string {
	for _, msg := range history {
		if msg.Role == "tool" && msg.ToolCallID == callID {
			return msg.ToolName
		}
	}
	return ""
}
