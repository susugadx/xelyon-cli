package api

import (
	"context"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestRenderActiveContextBlocks_JoinsNonBlankContent(t *testing.T) {
	got := RenderActiveContextBlocks([]ActiveContextBlock{
		{Name: "blank", Content: " \n\t "},
		{Name: "a", Content: "\n<current_task_state>\nstate\n</current_task_state>\n"},
		{Name: "b", Content: "<rehydrated_evidence>\nevidence\n</rehydrated_evidence>"},
	})
	want := "<current_task_state>\nstate\n</current_task_state>\n\n<rehydrated_evidence>\nevidence\n</rehydrated_evidence>"
	if got != want {
		t.Fatalf("RenderActiveContextBlocks() = %q, want %q", got, want)
	}
}

func TestSystemPromptWithActiveContext_AppendsDynamicSuffix(t *testing.T) {
	got := SystemPromptWithActiveContext("static"+SystemPromptCacheBoundary+"dynamic", []ActiveContextBlock{{
		Name:    "provider_history_rehydrated_evidence",
		Content: "<rehydrated_evidence>\nstate\n</rehydrated_evidence>",
	}})
	want := "static" + SystemPromptCacheBoundary + "dynamic\n\n<rehydrated_evidence>\nstate\n</rehydrated_evidence>"
	if got != want {
		t.Fatalf("SystemPromptWithActiveContext() = %q, want %q", got, want)
	}
}

func TestProviderActiveContextTransportForProviderName(t *testing.T) {
	cfg := config.DefaultConfig()
	ctx := config.WithContext(context.Background(), cfg)

	tests := []struct {
		name            string
		runtimeProvider string
		catalogProvider string
		model           string
		want            ActiveContextTransport
	}{
		{name: "openai responses", runtimeProvider: "openai", catalogProvider: "openai", model: "gpt-5.4", want: ActiveContextTransportNativeResponses},
		{name: "openai chat completions", runtimeProvider: "openai", catalogProvider: "openai", model: "gpt-4-turbo", want: ActiveContextTransportEphemeralSystem},
		{name: "azure responses", runtimeProvider: "azure", catalogProvider: "azure", model: "corp-deployment", want: ActiveContextTransportNativeResponses},
		{name: "claude", runtimeProvider: "claude", catalogProvider: "claude", model: "claude-sonnet-4-6", want: ActiveContextTransportSystemPromptSuffix},
		{name: "gemini", runtimeProvider: "gemini", catalogProvider: "gemini", model: "gemini-2.5-pro", want: ActiveContextTransportEphemeralUserContent},
		{name: "deepseek", runtimeProvider: "deepseek", catalogProvider: "deepseek", model: "deepseek-chat", want: ActiveContextTransportEphemeralSystem},
		{name: "groq", runtimeProvider: "groq", catalogProvider: "groq", model: "llama-3.3-70b-versatile", want: ActiveContextTransportEphemeralSystem},
		{name: "kimi", runtimeProvider: "kimi", catalogProvider: "kimi", model: "kimi-k2.6", want: ActiveContextTransportEphemeralSystem},
		{name: "ollama", runtimeProvider: "ollama", catalogProvider: "ollama", model: "qwen2.5-coder:14b", want: ActiveContextTransportEphemeralSystem},
		{name: "bedrock claude", runtimeProvider: "bedrock", catalogProvider: "bedrock", model: "global.anthropic.claude-sonnet-4-6", want: ActiveContextTransportSystemPromptSuffix},
		{name: "bedrock converse", runtimeProvider: "bedrock", catalogProvider: "bedrock", model: "amazon.nova-pro-v1:0", want: ActiveContextTransportBedrockSystemBlock},
		{name: "unsupported", runtimeProvider: "unknown", catalogProvider: "unknown", model: "model", want: ActiveContextTransportNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ProviderActiveContextTransportForProviderName(tt.runtimeProvider, tt.catalogProvider, ctx, tt.model)
			if got != tt.want {
				t.Fatalf("ProviderActiveContextTransportForProviderName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProviderActiveContextTransportForRequest_RespectsProviderCapabilityNone(t *testing.T) {
	provider := fakeActiveContextTransportProvider{transport: ActiveContextTransportNone}

	got := ProviderActiveContextTransportForRequest(provider, "", "", context.Background(), "gpt-5.4")

	if got != ActiveContextTransportNone {
		t.Fatalf("ProviderActiveContextTransportForRequest() = %q, want provider capability None without generic fallback", got)
	}
}

type fakeActiveContextTransportProvider struct {
	transport ActiveContextTransport
}

func (p fakeActiveContextTransportProvider) Name() string { return "openai" }

func (p fakeActiveContextTransportProvider) ChatWithTools(context.Context, string, []Message, string) (string, error) {
	return "", nil
}

func (p fakeActiveContextTransportProvider) SupportsImages() bool { return false }

func (p fakeActiveContextTransportProvider) ChatWithImage(context.Context, string, []Message, string, *ImageData, string) (string, error) {
	return "", nil
}

func (p fakeActiveContextTransportProvider) IsFunctionCallingEnabled() bool { return false }

func (p fakeActiveContextTransportProvider) ActiveContextTransport(context.Context, string) ActiveContextTransport {
	return p.transport
}
