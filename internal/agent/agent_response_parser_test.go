package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	claudeprovider "github.com/susugadx/xelyon-cli/internal/api/providers/claude"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestHandleNormalResponse_UsesRuntimeOutputForCompactionNotice(t *testing.T) {
	var outA bytes.Buffer
	var outB bytes.Buffer

	agentA := &Agent{
		CurrentModel:    "test-model",
		CurrentProvider: &mockProvider{name: "test"},
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader(""), &outA, &outA),
		},
	}

	agentA.handleNormalResponse("before [COMPACTION]hidden[/COMPACTION] after")

	if !strings.Contains(outA.String(), "Context compacted by Claude") {
		t.Fatalf("expected runtime output to contain compaction notice, got %q", outA.String())
	}
	if outB.Len() != 0 {
		t.Fatalf("expected other runtime output to stay empty, got %q", outB.String())
	}
	if got := agentA.lastOutputs[len(agentA.lastOutputs)-1]; got != "before  after" {
		t.Fatalf("last output = %q, want %q", got, "before  after")
	}
}

func TestHandleNormalResponse_PrintsFinalResponseWhenAssistantUpdatesSuppressed(t *testing.T) {
	var out bytes.Buffer
	cfg := config.DefaultConfig()
	cfg.Output.AssistantUpdates = "phase"

	agent := &Agent{
		CurrentModel:    "test-model",
		CurrentProvider: &mockProvider{name: "test"},
		Runtime: &AgentRuntime{
			Config: cfg,
			UI:     ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
	}

	agent.handleNormalResponse("final response")

	if !strings.Contains(out.String(), "💬 final response") {
		t.Fatalf("expected suppressed mode to print final response once, got %q", out.String())
	}
}

func TestClaudeNonStreamingAssistantUpdates_PrintsFinalResponseExactlyOnce(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(claudeprovider.Response{
			Content: []claudeprovider.Content{{Type: "text", Text: "Claude final response"}},
		})
	}))
	defer server.Close()

	t.Setenv("ANTHROPIC_API_URL", server.URL)

	tests := []struct {
		name string
		mode string
	}{
		{name: "phase", mode: "phase"},
		{name: "off", mode: "off"},
		{name: "verbose", mode: "verbose"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			cfg := config.DefaultConfig()
			cfg.Output.AssistantUpdates = tt.mode

			provider := claudeprovider.New("test-key")
			agent := &Agent{
				CurrentModel:    "claude-sonnet-4-6",
				CurrentProvider: provider,
				Runtime: &AgentRuntime{
					Config: cfg,
					UI:     ui.NewRuntime(strings.NewReader(""), &out, &out),
				},
			}

			response, err := provider.ChatWithTools(
				agent.requestContext(context.Background()),
				"System prompt",
				nil,
				agent.CurrentModel,
			)
			if err != nil {
				t.Fatalf("ChatWithTools() error = %v", err)
			}

			agent.handleNormalResponse(response)

			if got := strings.Count(out.String(), "Claude final response"); got != 1 {
				t.Fatalf("expected final response to be printed exactly once in %s mode, got %d in output %q", tt.mode, got, out.String())
			}
		})
	}
}
