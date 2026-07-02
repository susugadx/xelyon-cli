package agent

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

type imageCapableChatProvider struct {
	name             string
	chatCalls        int
	imageCalls       int
	lastSystemPrompt string
	lastMessage      string
	lastImage        *api.ImageData
}

func (p *imageCapableChatProvider) Name() string {
	if p.name != "" {
		return p.name
	}
	return "openai"
}

func (p *imageCapableChatProvider) SupportsImages() bool { return true }

func (p *imageCapableChatProvider) IsFunctionCallingEnabled() bool { return true }

func (p *imageCapableChatProvider) ChatWithTools(_ context.Context, systemPrompt string, history []api.Message, _ string) (string, error) {
	if len(history) > 0 && history[len(history)-1].HasImage() {
		last := history[len(history)-1]
		p.imageCalls++
		p.lastSystemPrompt = systemPrompt
		p.lastMessage = last.Content
		p.lastImage = last.ImageData()
		return "image response", nil
	}
	p.chatCalls++
	return "text response", nil
}

func (p *imageCapableChatProvider) ChatWithImage(_ context.Context, systemPrompt string, _ []api.Message, userMessage string, image *api.ImageData, _ string) (string, error) {
	return "", fmt.Errorf("ChatWithImage should not be called: prompt=%q message=%q image=%v", systemPrompt, userMessage, image != nil)
}

func TestChatWrapper_UsesTextPath(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &scriptedChatProvider{name: "openai", functionCalling: true}
	agent := newChatRequestTestAgent(t, provider, &out)

	if err := agent.chat("hello"); err != nil {
		t.Fatalf("chat() error = %v", err)
	}

	if provider.callCount != 1 {
		t.Fatalf("provider.callCount = %d, want 1", provider.callCount)
	}
}

func TestChatInternal_WithImageUsesImageProviderPath(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &imageCapableChatProvider{name: "openai"}
	agent := newChatRequestTestAgent(t, provider, &out)
	image := &api.ImageData{Base64: "dGVzdA==", MediaType: "image/png", Path: "test.png", Size: 4}

	if err := agent.chatInternal("describe image", image); err != nil {
		t.Fatalf("chatInternal() error = %v", err)
	}

	if provider.imageCalls != 1 {
		t.Fatalf("provider.imageCalls = %d, want 1", provider.imageCalls)
	}
	if provider.lastMessage != "describe image" {
		t.Fatalf("provider.lastMessage = %q, want raw user input", provider.lastMessage)
	}
	if strings.Contains(provider.lastMessage, "[NORMAL MODE]") {
		t.Fatalf("provider.lastMessage = %q, should not contain normal-mode suffix", provider.lastMessage)
	}
	if !strings.Contains(provider.lastSystemPrompt, normalModeBaseRuntimeDirective) {
		t.Fatalf("provider.lastSystemPrompt = %q, want normal-mode runtime directive", provider.lastSystemPrompt)
	}
}

func TestChatWithImage_FallbackAndSupported(t *testing.T) {
	disableColors(t)

	t.Run("unsupported provider falls back to text chat", func(t *testing.T) {
		var out bytes.Buffer
		provider := &mockProvider{name: "ollama"}
		agent := newChatRequestTestAgent(t, provider, &out)
		image := &api.ImageData{Base64: "dGVzdA==", MediaType: "image/png", Path: "test.png", Size: 4}

		if err := agent.chatWithImage("fallback request", image); err != nil {
			t.Fatalf("chatWithImage() error = %v", err)
		}

		got := out.String()
		if !strings.Contains(got, "does not support images") {
			t.Fatalf("chatWithImage() output = %q, want fallback warning", got)
		}
	})

	t.Run("supported provider logs image send", func(t *testing.T) {
		var out bytes.Buffer
		provider := &imageCapableChatProvider{name: "openai"}
		agent := newChatRequestTestAgent(t, provider, &out)
		image := &api.ImageData{Base64: "dGVzdA==", MediaType: "image/png", Path: "test.png", Size: 4}

		if err := agent.chatWithImage("describe", image); err != nil {
			t.Fatalf("chatWithImage() error = %v", err)
		}

		if provider.imageCalls != 1 {
			t.Fatalf("provider.imageCalls = %d, want 1", provider.imageCalls)
		}
		if !strings.Contains(out.String(), "Sending image:") {
			t.Fatalf("chatWithImage() output = %q, want image send log", out.String())
		}
	})
}
