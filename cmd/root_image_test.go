package cmd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestRootCommand_ImageFlagPreservesImagePath(t *testing.T) {
	withRootCommandTest(t)

	imageCalled := false
	onceCalled := false
	interactiveCalled := false
	runOnce = func(query string, model string, provider api.Provider, cfg *config.Config, autoApprove bool, quiet bool) error {
		onceCalled = true
		return nil
	}
	runLegacyInteractive = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
		interactiveCalled = true
	}
	runOnceWithImage = func(query string, model string, provider api.Provider, imagePath string, cfg *config.Config, autoApprove bool, quiet bool) error {
		imageCalled = true
		if query != "describe" {
			t.Fatalf("query = %q, want describe", query)
		}
		if imagePath != "/tmp/image.png" {
			t.Fatalf("imagePath = %q, want /tmp/image.png", imagePath)
		}
		return nil
	}

	rootCmd.SetArgs([]string{"--image", "/tmp/image.png", "--provider", "ollama", "--no-update-check", "describe"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !imageCalled {
		t.Fatal("expected image path to be executed when --image is set")
	}
	if onceCalled {
		t.Fatal("text one-shot path must not be executed when --image is set")
	}
	if interactiveCalled {
		t.Fatal("interactive path must not be executed when --image is set")
	}
}

func TestRootCommand_OnceWithImageUsesImageOneShotPath(t *testing.T) {
	withRootCommandTest(t)

	imageCalled := false
	onceCalled := false
	runOnce = func(query string, model string, provider api.Provider, cfg *config.Config, autoApprove bool, quiet bool) error {
		onceCalled = true
		return nil
	}
	runOnceWithImage = func(query string, model string, provider api.Provider, imagePath string, cfg *config.Config, autoApprove bool, quiet bool) error {
		imageCalled = true
		if query != "describe" {
			t.Fatalf("query = %q, want describe", query)
		}
		if imagePath != "/tmp/image.png" {
			t.Fatalf("imagePath = %q, want /tmp/image.png", imagePath)
		}
		return nil
	}

	rootCmd.SetArgs([]string{"--once", "--image", "/tmp/image.png", "--provider", "ollama", "--no-update-check", "describe"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !imageCalled {
		t.Fatal("expected --once --image to use image one-shot path")
	}
	if onceCalled {
		t.Fatal("text one-shot path must not be executed for --once --image")
	}
}

func TestRootCommand_OnceWithImageWithoutQueryUsesImageOneShotPath(t *testing.T) {
	withRootCommandTest(t)

	imageCalled := false
	runOnce = func(query string, model string, provider api.Provider, cfg *config.Config, autoApprove bool, quiet bool) error {
		t.Fatal("text one-shot path must not be executed for --once --image without query")
		return nil
	}
	runOnceWithImage = func(query string, model string, provider api.Provider, imagePath string, cfg *config.Config, autoApprove bool, quiet bool) error {
		imageCalled = true
		if query != "" {
			t.Fatalf("query = %q, want empty string so image path can apply its default prompt", query)
		}
		if imagePath != "/tmp/image.png" {
			t.Fatalf("imagePath = %q, want /tmp/image.png", imagePath)
		}
		return nil
	}

	rootCmd.SetArgs([]string{"--once", "--image", "/tmp/image.png", "--provider", "ollama", "--no-update-check"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !imageCalled {
		t.Fatal("expected --once --image without query to use image one-shot path")
	}
}

func TestRootCommand_InteractiveWithImageUsesInteractiveImagePath(t *testing.T) {
	withRootCommandTest(t)

	imageInteractiveCalled := false
	imageOnceCalled := false
	runTUIWithImage = func(query string, model string, provider api.Provider, imagePath string, cfg *config.Config, autoApprove bool) error {
		imageInteractiveCalled = true
		if query != "describe this screenshot" {
			t.Fatalf("query = %q, want %q", query, "describe this screenshot")
		}
		if imagePath != "/tmp/image.png" {
			t.Fatalf("imagePath = %q, want /tmp/image.png", imagePath)
		}
		return nil
	}
	runOnceWithImage = func(query string, model string, provider api.Provider, imagePath string, cfg *config.Config, autoApprove bool, quiet bool) error {
		imageOnceCalled = true
		return nil
	}

	rootCmd.SetArgs([]string{"--interactive", "--image", "/tmp/image.png", "--provider", "ollama", "--no-update-check", "describe", "this", "screenshot"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !imageInteractiveCalled {
		t.Fatal("expected --interactive --image to use interactive image path")
	}
	if imageOnceCalled {
		t.Fatal("image one-shot path must not be executed for --interactive --image")
	}
}

func TestRootCommand_InteractiveImageMissingProviderCredentialUsesPlaceholder(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OPENAI_API_KEY", "")

	imageInteractiveCalled := false
	runTUIWithImage = func(query string, model string, provider api.Provider, imagePath string, cfg *config.Config, autoApprove bool) error {
		imageInteractiveCalled = true
		if !api.IsProviderSetupRequired(provider) {
			t.Fatalf("provider = %T, want setup placeholder", provider)
		}
		if imagePath != "/tmp/image.png" {
			t.Fatalf("imagePath = %q, want /tmp/image.png", imagePath)
		}
		return nil
	}
	runOnceWithImage = func(query string, model string, provider api.Provider, imagePath string, cfg *config.Config, autoApprove bool, quiet bool) error {
		t.Fatal("image one-shot path must not run for --interactive --image")
		return nil
	}

	rootCmd.SetArgs([]string{"--interactive", "--image", "/tmp/image.png", "--provider", "openai", "--no-update-check", "describe"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !imageInteractiveCalled {
		t.Fatal("expected interactive image path")
	}
}

func TestRootCommand_NoTUIInteractiveImageUsesLegacyPath(t *testing.T) {
	withRootCommandTest(t)

	tuiCalled := false
	legacyCalled := false
	runTUIWithImage = func(query string, model string, provider api.Provider, imagePath string, cfg *config.Config, autoApprove bool) error {
		tuiCalled = true
		return nil
	}
	runLegacyInteractiveWithImage = func(query string, model string, provider api.Provider, imagePath string, cfg *config.Config, autoApprove bool) error {
		legacyCalled = true
		return nil
	}

	rootCmd.SetArgs([]string{"--interactive", "--no-tui", "--image", "/tmp/image.png", "--provider", "ollama", "--no-update-check", "describe"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !legacyCalled {
		t.Fatal("expected --no-tui --interactive --image to use legacy image path")
	}
	if tuiCalled {
		t.Fatal("TUI image path must not be executed when --no-tui is set")
	}
}

func TestRootCommand_ImageErrorPropagation(t *testing.T) {
	withRootCommandTest(t)

	runOnceWithImage = func(query string, model string, provider api.Provider, imagePath string, cfg *config.Config, autoApprove bool, quiet bool) error {
		return fmt.Errorf("image failed")
	}

	rootCmd.SetArgs([]string{"--image", "/tmp/image.png", "--provider", "ollama", "--no-update-check", "describe"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error to propagate from image one-shot execution")
	}
	if !strings.Contains(err.Error(), "image failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRootCommand_QuietWithImageUsesOneShotImagePath(t *testing.T) {
	withRootCommandTest(t)

	called := false
	runOnceWithImage = func(query string, model string, provider api.Provider, imagePath string, cfg *config.Config, autoApprove bool, quiet bool) error {
		called = true
		if !quiet {
			t.Fatal("expected quiet to be passed to image one-shot execution")
		}
		if query != "describe" {
			t.Fatalf("query = %q, want describe", query)
		}
		if imagePath != "/tmp/image.png" {
			t.Fatalf("imagePath = %q, want /tmp/image.png", imagePath)
		}
		return nil
	}

	rootCmd.SetArgs([]string{"--quiet", "--image", "/tmp/image.png", "--provider", "ollama", "--no-update-check", "describe"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !called {
		t.Fatal("expected quiet image one-shot execution")
	}
}

func TestRootCommand_ResumeWithImageReturnsError(t *testing.T) {
	withRootCommandTest(t)
	rootCmd.SetArgs([]string{"--resume", "--image", "/tmp/image.png", "--provider", "ollama", "--no-update-check"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for --resume with image")
	}
	if !strings.Contains(err.Error(), "--resume cannot be used with --image") {
		t.Fatalf("unexpected error: %v", err)
	}
}
