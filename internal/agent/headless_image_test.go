package agent

import (
	"context"
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
		t.Fatalf("ChatWithImage() called %d times, want 1", provider.imageCalls)
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
