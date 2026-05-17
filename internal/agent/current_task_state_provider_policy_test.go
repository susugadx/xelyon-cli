package agent

import (
	"bytes"
	"testing"
)

func TestShouldSendActiveContextToProvider(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		fixture currentTaskStateProviderFixture
		want    bool
	}{
		{
			name:    "disabled",
			enabled: false,
			fixture: currentTaskStateOpenAIResponses,
			want:    false,
		},
		{
			name:    "openai responses",
			enabled: true,
			fixture: currentTaskStateOpenAIResponses,
			want:    true,
		},
		{
			name:    "azure responses",
			enabled: true,
			fixture: currentTaskStateAzureResponses,
			want:    true,
		},
		{
			name:    "openai chat completions",
			enabled: true,
			fixture: currentTaskStateOpenAIChatCompletions,
			want:    false,
		},
		{
			name:    "deepseek",
			enabled: true,
			fixture: currentTaskStateDeepSeek,
			want:    false,
		},
		{
			name:    "gemini",
			enabled: true,
			fixture: currentTaskStateGemini,
			want:    false,
		},
		{
			name:    "claude",
			enabled: true,
			fixture: currentTaskStateClaude,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &Agent{
				Runtime: &AgentRuntime{
					Options: RuntimeOptions{EnableCurrentTaskStateContext: tt.enabled},
				},
			}
			applyCurrentTaskStateProviderFixture(agent, tt.fixture)

			if got := agent.shouldSendActiveContextToProvider(); got != tt.want {
				t.Fatalf("shouldSendActiveContextToProvider() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestPrepareChatRequest_CurrentTaskStateContextClearsResponseContextBeforeSessionSave(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	agent, provider := newCurrentTaskStateResponseIDAgent(t, currentTaskStateOpenAIResponses, "resp_old", &out)

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

func TestPrepareChatRequest_CurrentTaskStateContextKeepsResponseContextForProvidersThatDoNotConsumeIt(t *testing.T) {
	disableColors(t)

	tests := []struct {
		name    string
		fixture currentTaskStateProviderFixture
	}{
		{
			name:    "deepseek",
			fixture: currentTaskStateDeepSeek,
		},
		{
			name:    "gemini",
			fixture: currentTaskStateGemini,
		},
		{
			name:    "claude",
			fixture: currentTaskStateClaude,
		},
		{
			name:    "openai chat completions",
			fixture: currentTaskStateOpenAIChatCompletions,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			agent, provider := newCurrentTaskStateResponseIDAgent(t, tt.fixture, "resp_old", &out)

			if len(agent.buildActiveContextBlocks()) == 0 {
				t.Fatal("test setup produced empty active context")
			}
			if agent.shouldSendActiveContextToProvider() {
				t.Fatalf("test setup provider/model %s/%s consumes active context", tt.fixture.providerName, tt.fixture.model)
			}

			agent.prepareChatRequest(&chatRequest{input: "hello"})

			if provider.GetResponseID() != "resp_old" {
				t.Fatalf("provider response ID = %q, want preserved for provider that does not consume active context", provider.GetResponseID())
			}
			if agent.session.ResponseID != "resp_old" {
				t.Fatalf("session.ResponseID = %q, want preserved for provider that does not consume active context", agent.session.ResponseID)
			}
			if len(agent.session.Messages) == 0 || agent.session.Messages[len(agent.session.Messages)-1].Content != "hello" {
				t.Fatalf("session messages = %#v, want appended user message", agent.session.Messages)
			}
		})
	}
}
