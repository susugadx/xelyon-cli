package agent

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/api"
)

type imageOnceProvider struct {
	imageCalls int
}

func (p *imageOnceProvider) Name() string { return "openai" }

func (p *imageOnceProvider) SupportsImages() bool { return true }

func (p *imageOnceProvider) IsFunctionCallingEnabled() bool { return true }

func (p *imageOnceProvider) ChatWithTools(context.Context, string, []api.Message, string) (string, error) {
	return "", fmt.Errorf("ChatWithTools should not be called for image one-shot")
}

func (p *imageOnceProvider) ChatWithImage(_ context.Context, _ string, _ []api.Message, _ string, image *api.ImageData, _ string) (string, error) {
	if image == nil {
		return "", fmt.Errorf("image is required")
	}
	p.imageCalls++
	return "mock image response", nil
}

func TestRunOnceWithImageWithConfig_UnsupportedProviderReturnsError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	err := RunOnceWithImageWithConfig("describe", "test-model", &mockProvider{name: "ollama"}, "/tmp/test.png", newProjectMapDisabledConfig(), false, false)
	if err == nil {
		t.Fatal("expected error for unsupported image provider")
	}
	if !strings.Contains(err.Error(), "does not support image input") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunOnceWithImageWithConfig_MissingImageReturnsError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	err := RunOnceWithImageWithConfig("describe", "test-model", &imageOnceProvider{}, filepath.Join(t.TempDir(), "missing.png"), newProjectMapDisabledConfig(), false, false)
	if err == nil {
		t.Fatal("expected error for missing image")
	}
	if !strings.Contains(err.Error(), "failed to load image") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunOnceWithImageWithConfig_ExecutesSingleTurn(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	imagePath := filepath.Join(t.TempDir(), "test.png")
	if err := os.WriteFile(imagePath, []byte("fake png data for testing"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	provider := &imageOnceProvider{}
	err := RunOnceWithImageWithConfig("describe", "test-model", provider, imagePath, newProjectMapDisabledConfig(), false, false)
	if err != nil {
		t.Fatalf("RunOnceWithImageWithConfig() error = %v", err)
	}
	if provider.imageCalls != 1 {
		t.Fatalf("ChatWithImage() called %d times, want 1", provider.imageCalls)
	}
}

func TestRunOnceWithImageWithConfig_QuietSuppressesStatusOutput(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	imagePath := filepath.Join(t.TempDir(), "test.png")
	if err := os.WriteFile(imagePath, []byte("fake png data for testing"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	oldStdout := os.Stdout
	oldColorOutput := color.Output
	r, w, _ := os.Pipe()
	os.Stdout = w
	color.Output = w
	t.Cleanup(func() {
		os.Stdout = oldStdout
		color.Output = oldColorOutput
	})

	provider := &imageOnceProvider{}
	err := RunOnceWithImageWithConfig("describe", "test-model", provider, imagePath, newProjectMapDisabledConfig(), false, true)

	_ = w.Close()
	os.Stdout = oldStdout
	color.Output = oldColorOutput

	if err != nil {
		t.Fatalf("RunOnceWithImageWithConfig() error = %v", err)
	}
	if provider.imageCalls != 1 {
		t.Fatalf("ChatWithImage() called %d times, want 1", provider.imageCalls)
	}

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()
	if strings.Contains(output, "Image loaded:") {
		t.Fatalf("quiet output should not contain image status, got %q", output)
	}
	if strings.Contains(output, "Sending image:") {
		t.Fatalf("quiet output should not contain image send status, got %q", output)
	}
}
