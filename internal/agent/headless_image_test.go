package agent

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestRunHeadlessWithConfigOptions_ImageUsesMultimodalFirstRequest(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	image := &api.ImageData{
		Path:      "screen.png",
		MediaType: "image/png",
		Base64:    "raw-image-base64",
		Size:      12,
	}
	provider := &imageOnceProvider{}
	result := RunHeadlessWithConfigOptions(context.Background(), "describe image", "test-model", provider, newProjectMapDisabledConfig(), HeadlessRunOptions{
		Image: image,
	})

	if result.Status != HeadlessStatusSuccess {
		t.Fatalf("status = %q, want success: error=%+v", result.Status, result.Error)
	}
	if provider.imageCalls != 1 {
		t.Fatalf("ChatWithTools image calls = %d, want 1", provider.imageCalls)
	}
	if result.Input == nil || result.Input.Image == nil {
		t.Fatalf("input.image = nil, result=%+v", result)
	}
	if result.Input.Image.Path != image.Path || result.Input.Image.MIMEType != image.MediaType || result.Input.Image.Bytes != image.Size || !result.Input.Image.ProviderSupported {
		t.Fatalf("input.image = %+v, want bounded image metadata", result.Input.Image)
	}
	jsonOutput, err := result.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}
	if strings.Contains(jsonOutput, image.Base64) {
		t.Fatalf("headless JSON leaked raw image base64: %s", jsonOutput)
	}
}

type headlessImageToolLoopProvider struct {
	histories [][]api.Message
	responses []string
	index     int
}

func (p *headlessImageToolLoopProvider) Name() string { return "openai" }

func (p *headlessImageToolLoopProvider) SupportsImages() bool { return true }

func (p *headlessImageToolLoopProvider) IsFunctionCallingEnabled() bool { return true }

func (p *headlessImageToolLoopProvider) ChatWithTools(_ context.Context, _ string, history []api.Message, _ string) (string, error) {
	p.histories = append(p.histories, api.CloneMessages(history))
	if p.index >= len(p.responses) {
		return "done", nil
	}
	response := p.responses[p.index]
	p.index++
	return response, nil
}

func (p *headlessImageToolLoopProvider) ChatWithImage(context.Context, string, []api.Message, string, *api.ImageData, string) (string, error) {
	return "", fmt.Errorf("ChatWithImage should not be called for headless image")
}

func TestRunHeadlessWithConfigOptions_ImageToolContinuationKeepsImageAndToolResult(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	image := &api.ImageData{
		Path:      "screen.png",
		MediaType: "image/png",
		Base64:    "raw-image-base64",
		Size:      12,
	}
	provider := &headlessImageToolLoopProvider{
		responses: []string{
			`{"tool":"bash","args":{"command":"printf ok"}}`,
			"done",
		},
	}

	result := RunHeadlessWithConfigOptions(context.Background(), "describe image", "test-model", provider, newProjectMapDisabledConfig(), HeadlessRunOptions{
		Image: image,
	})

	if result.Status != HeadlessStatusSuccess {
		t.Fatalf("status = %q, want success: error=%+v", result.Status, result.Error)
	}
	if len(provider.histories) != 2 {
		t.Fatalf("provider histories = %d, want initial request and continuation", len(provider.histories))
	}
	first := provider.histories[0]
	if len(first) != 1 || !first[0].HasImage() || first[0].Content != "describe image" {
		t.Fatalf("first history = %#v, want image-bearing user message", first)
	}
	second := provider.histories[1]
	if len(second) < 3 {
		t.Fatalf("second history = %#v, want image message + assistant tool call + tool result", second)
	}
	if !second[0].HasImage() {
		t.Fatalf("second history[0] lost image state: %#v", second[0])
	}
	if second[1].Role != "assistant" || len(second[1].ToolCalls) == 0 {
		t.Fatalf("second history[1] = %#v, want assistant tool call", second[1])
	}
	if second[2].Role != "tool" || !strings.Contains(second[2].Content, "ok") {
		t.Fatalf("second history[2] = %#v, want bash tool result", second[2])
	}
}

func TestRunHeadlessWithConfigOptions_ImageOnlyDefaultsPrompt(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	image := &api.ImageData{
		Path:      "screen.png",
		MediaType: "image/png",
		Base64:    "raw-image-base64",
		Size:      12,
	}
	provider := &imageOnceProvider{}
	result := RunHeadlessWithConfigOptions(context.Background(), "", "test-model", provider, newProjectMapDisabledConfig(), HeadlessRunOptions{
		Image: image,
	})

	if result.Status != HeadlessStatusSuccess {
		t.Fatalf("status = %q, want success: error=%+v", result.Status, result.Error)
	}
	if provider.lastMessage != DefaultImagePrompt {
		t.Fatalf("image prompt = %q, want default image prompt", provider.lastMessage)
	}
	if result.Input == nil || result.Input.Bytes != len([]byte(DefaultImagePrompt)) {
		t.Fatalf("input = %+v, want default prompt byte metadata", result.Input)
	}
}

func TestHeadlessImageDocumentationIncludesJSONContract(t *testing.T) {
	body, err := os.ReadFile("../../docs/commands.md")
	if err != nil {
		t.Fatalf("ReadFile(docs/commands.md) error = %v", err)
	}
	text := string(body)
	required := []string{
		"--image screenshot.png",
		"`--image` は headless / JSON mode でも使えます",
		"input.image.path",
		"input.image.mime_type",
		"input.image.bytes",
		"input.image.provider_supported",
		"raw image bytes",
		"base64",
		`failure_reason:"unsupported_capability"`,
		`failure_reason:"usage_error"`,
	}
	for _, want := range required {
		if !strings.Contains(text, want) {
			t.Fatalf("docs/commands.md should include %q for the headless image JSON contract", want)
		}
	}
	forbidden := []string{
		"`--image` は `--headless` / `--output-format json` と併用できません",
	}
	for _, wantMissing := range forbidden {
		if strings.Contains(text, wantMissing) {
			t.Fatalf("docs/commands.md should not say %q after headless image support", wantMissing)
		}
	}
}
