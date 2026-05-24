package agent

import (
	"bytes"
	"testing"
)

func TestShouldSendActiveContextToProvider(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		fixture activeContextProviderFixture
		want    bool
	}{
		{
			name:    "disabled",
			enabled: false,
			fixture: activeContextOpenAIResponses,
			want:    false,
		},
		{
			name:    "openai responses",
			enabled: true,
			fixture: activeContextOpenAIResponses,
			want:    true,
		},
		{
			name:    "azure responses",
			enabled: true,
			fixture: activeContextAzureResponses,
			want:    true,
		},
		{
			name:    "openai chat completions",
			enabled: true,
			fixture: activeContextOpenAIChatCompletions,
			want:    true,
		},
		{
			name:    "deepseek",
			enabled: true,
			fixture: activeContextDeepSeek,
			want:    true,
		},
		{
			name:    "gemini",
			enabled: true,
			fixture: activeContextGemini,
			want:    true,
		},
		{
			name:    "claude",
			enabled: true,
			fixture: activeContextClaude,
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &Agent{
				Runtime: &AgentRuntime{
					Options: RuntimeOptions{EnableCurrentTaskStateContext: tt.enabled},
				},
			}
			applyActiveContextProviderFixture(agent, tt.fixture)

			if got := agent.shouldSendActiveContextToProvider(); got != tt.want {
				t.Fatalf("shouldSendActiveContextToProvider() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestPrepareChatRequest_CurrentTaskStateContextClearsResponseContextBeforeSessionSave(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	agent, provider := newCurrentTaskStateResponseIDAgent(t, activeContextOpenAIResponses, "resp_old", &out)

	agent.prepareChatRequest(&chatRequest{input: "hello"})

	if provider.GetResponseID() != "" {
		t.Fatalf("provider response ID = %q, want cleared before active-context request", provider.GetResponseID())
	}
	if agent.session.ResponseID != "" {
		t.Fatalf("session.ResponseID = %q, want cleared before session save", agent.session.ResponseID)
	}
	if len(agent.session.Messages) == 0 || agent.session.Messages[len(agent.session.Messages)-1].Content != "hello" {
		t.Fatalf("session messages = %#v, want appended user message", agent.session.Messages)
	}
}

func TestPrepareChatRequest_CurrentTaskStateContextKeepsResponseContextForUnsupportedProvider(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	agent, provider := newCurrentTaskStateResponseIDAgent(t, activeContextUnsupported, "resp_old", &out)

	if len(agent.buildActiveContextBlocks()) == 0 {
		t.Fatal("test setup produced empty active context")
	}
	if agent.shouldSendActiveContextToProvider() {
		t.Fatalf("test setup provider/model %s/%s consumes active context", activeContextUnsupported.providerName, activeContextUnsupported.model)
	}

	agent.prepareChatRequest(&chatRequest{input: "hello"})

	if provider.GetResponseID() != "resp_old" {
		t.Fatalf("provider response ID = %q, want preserved for unsupported active-context transport", provider.GetResponseID())
	}
	if agent.session.ResponseID != "resp_old" {
		t.Fatalf("session.ResponseID = %q, want preserved for unsupported active-context transport", agent.session.ResponseID)
	}
	if len(agent.session.Messages) == 0 || agent.session.Messages[len(agent.session.Messages)-1].Content != "hello" {
		t.Fatalf("session messages = %#v, want appended user message", agent.session.Messages)
	}
}
