package agent

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

type runtimeDirectiveLifecycleProvider struct {
	errByCall     map[int]error
	responses     []string
	systemPrompts []string
	imageCalls    int
	callCount     int
}

func (p *runtimeDirectiveLifecycleProvider) Name() string { return "openai" }

func (p *runtimeDirectiveLifecycleProvider) SupportsImages() bool { return true }

func (p *runtimeDirectiveLifecycleProvider) IsFunctionCallingEnabled() bool { return true }

func (p *runtimeDirectiveLifecycleProvider) ChatWithTools(_ context.Context, systemPrompt string, _ []api.Message, _ string) (string, error) {
	return p.nextResponse(systemPrompt)
}

func (p *runtimeDirectiveLifecycleProvider) ChatWithImage(_ context.Context, systemPrompt string, _ []api.Message, _ string, image *api.ImageData, _ string) (string, error) {
	if image == nil {
		return "", errors.New("image is required")
	}
	p.imageCalls++
	return p.nextResponse(systemPrompt)
}

func (p *runtimeDirectiveLifecycleProvider) nextResponse(systemPrompt string) (string, error) {
	call := p.callCount
	p.callCount++
	p.systemPrompts = append(p.systemPrompts, systemPrompt)
	if err := p.errByCall[call]; err != nil {
		return "", err
	}
	if call < len(p.responses) {
		return p.responses[call], nil
	}
	return "done", nil
}

func TestRuntimeDirectivesStayPendingUntilNormalProviderCallSucceeds(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &runtimeDirectiveLifecycleProvider{
		errByCall: map[int]error{0: errors.New("temporary provider error")},
	}
	agent := newChatRequestTestAgent(t, provider, &out)
	agent.queueRuntimeDirective(finalCheckFailureRuntimeDirective)
	runner := newTurnRunner(agent, context.Background())

	if _, err := runner.requestNormalModeResponse("fix final check", nil, 0); err == nil {
		t.Fatal("first request should return provider error")
	}
	if len(agent.runtimeDirectives) != 1 || agent.runtimeDirectives[0] != finalCheckFailureRuntimeDirective {
		t.Fatalf("runtime directives after failed provider call = %#v, want pending final-check directive", agent.runtimeDirectives)
	}

	if _, err := runner.requestNormalModeResponse("fix final check", nil, 1); err != nil {
		t.Fatalf("second request returned error: %v", err)
	}
	if len(agent.runtimeDirectives) != 0 {
		t.Fatalf("runtime directives after successful provider call = %#v, want cleared", agent.runtimeDirectives)
	}
	if len(provider.systemPrompts) != 2 {
		t.Fatalf("provider prompts = %d, want 2", len(provider.systemPrompts))
	}
	for i, prompt := range provider.systemPrompts {
		if !strings.Contains(prompt, finalCheckFailureRuntimeDirective) {
			t.Fatalf("provider prompt %d = %q, want final-check directive", i, prompt)
		}
	}
}

func TestRuntimeDirectivesStayPendingUntilImageProviderCallSucceeds(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &runtimeDirectiveLifecycleProvider{
		errByCall: map[int]error{0: errors.New("temporary image provider error")},
	}
	agent := newChatRequestTestAgent(t, provider, &out)
	agent.queueRuntimeDirective(strReplaceLoopRuntimeDirective)
	runner := newTurnRunner(agent, context.Background())
	image := &api.ImageData{Base64: "dGVzdA==", MediaType: "image/png", Path: "test.png", Size: 4}

	if _, err := runner.requestNormalModeResponse("retry image edit", image, 0); err == nil {
		t.Fatal("first image request should return provider error")
	}
	if len(agent.runtimeDirectives) != 1 || agent.runtimeDirectives[0] != strReplaceLoopRuntimeDirective {
		t.Fatalf("runtime directives after failed image call = %#v, want pending str_replace directive", agent.runtimeDirectives)
	}

	if _, err := runner.requestNormalModeResponse("retry image edit", image, 0); err != nil {
		t.Fatalf("second image request returned error: %v", err)
	}
	if len(agent.runtimeDirectives) != 0 {
		t.Fatalf("runtime directives after successful image call = %#v, want cleared", agent.runtimeDirectives)
	}
	if provider.imageCalls != 2 {
		t.Fatalf("image calls = %d, want 2", provider.imageCalls)
	}
	for i, prompt := range provider.systemPrompts {
		if !strings.Contains(prompt, strReplaceLoopRuntimeDirective) {
			t.Fatalf("image provider prompt %d = %q, want str_replace directive", i, prompt)
		}
	}
}

func TestRuntimeDirectivesStayPendingUntilHeadlessProviderCallSucceeds(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	provider := &runtimeDirectiveLifecycleProvider{
		errByCall: map[int]error{0: errors.New("temporary headless provider error")},
	}
	runner := newHeadlessRunner("fix headless", "test-model", provider, newProjectMapDisabledConfig())
	t.Cleanup(runner.agent.Cleanup)
	runner.agent.queueRuntimeDirective(finalCheckFailureRuntimeDirective)

	if _, err := runner.requestAssistantResponse(context.Background(), 0); err == nil {
		t.Fatal("first headless request should return provider error")
	}
	if len(runner.agent.runtimeDirectives) != 1 || runner.agent.runtimeDirectives[0] != finalCheckFailureRuntimeDirective {
		t.Fatalf("headless runtime directives after failed provider call = %#v, want pending final-check directive", runner.agent.runtimeDirectives)
	}

	if _, err := runner.requestAssistantResponse(context.Background(), 1); err != nil {
		t.Fatalf("second headless request returned error: %v", err)
	}
	if len(runner.agent.runtimeDirectives) != 0 {
		t.Fatalf("headless runtime directives after successful provider call = %#v, want cleared", runner.agent.runtimeDirectives)
	}
	if len(provider.systemPrompts) != 2 {
		t.Fatalf("headless provider prompts = %d, want 2", len(provider.systemPrompts))
	}
	for i, prompt := range provider.systemPrompts {
		if !strings.Contains(prompt, finalCheckFailureRuntimeDirective) {
			t.Fatalf("headless provider prompt %d = %q, want final-check directive", i, prompt)
		}
	}
}
