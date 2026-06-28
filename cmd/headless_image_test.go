package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/agent"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestRootCommand_HeadlessWithImagePassesMetadataAndOptions(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OPENAI_API_KEY", "test-key")

	imageBytes := []byte("fake png data for headless image")
	imagePath := writeHeadlessImageTestFile(t, "screen.png", imageBytes)
	var gotImage *api.ImageData
	runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config, options agent.HeadlessRunOptions) *agent.HeadlessResult {
		if query != "describe" {
			t.Fatalf("query = %q, want describe", query)
		}
		if options.Image == nil {
			t.Fatal("options.Image = nil, want loaded image")
		}
		gotImage = options.Image
		if gotImage.Path != imagePath || gotImage.MediaType != "image/png" || gotImage.Size != int64(len(imageBytes)) || gotImage.Base64 == "" {
			t.Fatalf("options.Image = %+v, want loaded png metadata and base64 body", gotImage)
		}
		return agent.NewSuccessResult(provider.Name(), model, "ok", nil, 0)
	}

	parsed, output, stderr, err := executeRootCommandForHeadlessJSONTest(t, []string{"--headless", "--provider", "openai", "--no-update-check", "--image", imagePath, "describe"}, "")
	if err != nil {
		t.Fatalf("Execute() error = %v\nstderr=%s", err, stderr)
	}
	requireHeadlessInputImage(t, parsed.Input, imagePath, "image/png", int64(len(imageBytes)), true)
	if gotImage == nil {
		t.Fatal("headless runner was not called")
	}
	if strings.Contains(output, gotImage.Base64) {
		t.Fatalf("stdout JSON leaked raw image base64: %q", output)
	}
}

func TestRootCommand_HeadlessWithUnsupportedImageProviderReturnsJSON(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())

	runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config, options agent.HeadlessRunOptions) *agent.HeadlessResult {
		t.Fatal("headless runner must not run when provider does not support image input")
		return nil
	}

	imagePath := filepath.Join(t.TempDir(), "missing-is-not-read.png")
	parsed, _, stderr, err := executeRootCommandForHeadlessJSONTest(t, []string{"--headless", "--exit-code-policy", "ci", "--provider", "ollama", "--no-update-check", "--image", imagePath, "describe"}, "")
	if err == nil {
		t.Fatal("expected unsupported image provider to fail")
	}
	requireCommandExitCode(t, err, 9)
	if strings.Contains(stderr, "Usage:") {
		t.Fatalf("stderr contains Cobra usage after headless JSON error:\n%s", stderr)
	}
	if parsed.Error == nil || parsed.Error.Type != agent.HeadlessErrorTypeUnsupportedCapability {
		t.Fatalf("error = %+v, want %s", parsed.Error, agent.HeadlessErrorTypeUnsupportedCapability)
	}
	if parsed.FailureReason != agent.HeadlessFailureReasonUnsupportedCapability {
		t.Fatalf("failure_reason = %q, want %q", parsed.FailureReason, agent.HeadlessFailureReasonUnsupportedCapability)
	}
	if parsed.RecommendedExitCode != 9 {
		t.Fatalf("recommended_exit_code = %d, want 9", parsed.RecommendedExitCode)
	}
	requireHeadlessInputImage(t, parsed.Input, imagePath, "", 0, false)
}

func TestRootCommand_HeadlessImageUnsupportedProviderSetupRequiredPrefersUnsupportedCapability(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GROQ_API_KEY", "")

	runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config, options agent.HeadlessRunOptions) *agent.HeadlessResult {
		t.Fatal("headless runner must not run when provider does not support image input")
		return nil
	}

	imagePath := filepath.Join(t.TempDir(), "missing-is-not-read.png")
	parsed, _, stderr, err := executeRootCommandForHeadlessJSONTest(t, []string{"--headless", "--exit-code-policy", "ci", "--provider", "groq", "--no-update-check", "--image", imagePath, "describe"}, "")
	if err == nil {
		t.Fatal("expected unsupported image provider to fail")
	}
	requireCommandExitCode(t, err, 9)
	if strings.Contains(stderr, "Usage:") {
		t.Fatalf("stderr contains Cobra usage after headless JSON error:\n%s", stderr)
	}
	if parsed.Error == nil || parsed.Error.Type != agent.HeadlessErrorTypeUnsupportedCapability {
		t.Fatalf("error = %+v, want %s", parsed.Error, agent.HeadlessErrorTypeUnsupportedCapability)
	}
	if parsed.FailureReason != agent.HeadlessFailureReasonUnsupportedCapability {
		t.Fatalf("failure_reason = %q, want %q", parsed.FailureReason, agent.HeadlessFailureReasonUnsupportedCapability)
	}
	if parsed.RecommendedExitCode != 9 {
		t.Fatalf("recommended_exit_code = %d, want 9", parsed.RecommendedExitCode)
	}
	requireHeadlessInputImage(t, parsed.Input, imagePath, "", 0, false)
}

func TestRootCommand_HeadlessMissingImageReturnsJSONUsageError(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OPENAI_API_KEY", "test-key")

	runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config, options agent.HeadlessRunOptions) *agent.HeadlessResult {
		t.Fatal("headless runner must not run when image loading fails")
		return nil
	}

	imagePath := filepath.Join(t.TempDir(), "missing.png")
	parsed, _, stderr, err := executeRootCommandForHeadlessJSONTest(t, []string{"--headless", "--exit-code-policy", "ci", "--provider", "openai", "--no-update-check", "--image", imagePath, "describe"}, "")
	if err == nil {
		t.Fatal("expected missing image to fail")
	}
	requireCommandExitCode(t, err, 2)
	if strings.Contains(stderr, "Usage:") {
		t.Fatalf("stderr contains Cobra usage after headless JSON error:\n%s", stderr)
	}
	if parsed.Error == nil || parsed.Error.Type != agent.HeadlessErrorTypeConfig {
		t.Fatalf("error = %+v, want %s", parsed.Error, agent.HeadlessErrorTypeConfig)
	}
	if parsed.FailureReason != agent.HeadlessFailureReasonUsageError {
		t.Fatalf("failure_reason = %q, want %q", parsed.FailureReason, agent.HeadlessFailureReasonUsageError)
	}
	requireHeadlessInputImage(t, parsed.Input, imagePath, "", 0, true)
}

func TestRootCommand_HeadlessImageProviderSetupRequiredSkipsImageLoad(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OPENAI_API_KEY", "")

	runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config, options agent.HeadlessRunOptions) *agent.HeadlessResult {
		t.Fatal("headless runner must not run when provider setup is required")
		return nil
	}

	imagePath := filepath.Join(t.TempDir(), "missing-is-not-read.png")
	parsed, _, stderr, err := executeRootCommandForHeadlessJSONTest(t, []string{"--headless", "--exit-code-policy", "ci", "--provider", "openai", "--no-update-check", "--image", imagePath, "describe"}, "")
	if err == nil {
		t.Fatal("expected provider setup required error")
	}
	requireCommandExitCode(t, err, 3)
	if strings.Contains(stderr, "Usage:") {
		t.Fatalf("stderr contains Cobra usage after headless JSON error:\n%s", stderr)
	}
	if parsed.Error == nil || parsed.Error.Type != agent.HeadlessErrorTypeProviderSetupRequired {
		t.Fatalf("error = %+v, want %s", parsed.Error, agent.HeadlessErrorTypeProviderSetupRequired)
	}
	if parsed.FailureReason != agent.HeadlessFailureReasonProviderSetupRequired {
		t.Fatalf("failure_reason = %q, want %q", parsed.FailureReason, agent.HeadlessFailureReasonProviderSetupRequired)
	}
	requireHeadlessInputImage(t, parsed.Input, imagePath, "", 0, true)
}

func requireHeadlessInputImage(t *testing.T, input *agent.HeadlessInput, path string, mimeType string, bytes int64, providerSupported bool) {
	t.Helper()
	if input == nil {
		t.Fatal("input = nil")
	}
	if input.Image == nil {
		t.Fatal("input.image = nil")
	}
	if input.Image.Path != path {
		t.Fatalf("input.image.path = %q, want %q", input.Image.Path, path)
	}
	if input.Image.MIMEType != mimeType {
		t.Fatalf("input.image.mime_type = %q, want %q", input.Image.MIMEType, mimeType)
	}
	if input.Image.Bytes != bytes {
		t.Fatalf("input.image.bytes = %d, want %d", input.Image.Bytes, bytes)
	}
	if input.Image.ProviderSupported != providerSupported {
		t.Fatalf("input.image.provider_supported = %t, want %t", input.Image.ProviderSupported, providerSupported)
	}
}

func writeHeadlessImageTestFile(t *testing.T, name string, body []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}
