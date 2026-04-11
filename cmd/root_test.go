package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/agent"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

type headlessSequenceProvider struct {
	name      string
	responses []string
	index     int
}

func (p *headlessSequenceProvider) Name() string {
	return p.name
}

func (p *headlessSequenceProvider) SupportsImages() bool {
	return false
}

func (p *headlessSequenceProvider) IsFunctionCallingEnabled() bool {
	return true
}

func (p *headlessSequenceProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	if p.index >= len(p.responses) {
		return "done", nil
	}
	resp := p.responses[p.index]
	p.index++
	return resp, nil
}

func (p *headlessSequenceProvider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	return "", fmt.Errorf("not supported")
}

func resetRootFlagsForTest() {
	resume = false
	once = false
	interactive = false
	quiet = false
	providerFlag = ""
	modelFlag = ""
	autoApprove = false
	loopThreshold = 0
	diffLines = -1
	outputFormat = "text"
	headless = false
	noUpdateCheck = false
	imageFlag = ""
	noTUI = false
}

func TestRootCommand_PositionalQueryDefaultsToOnce(t *testing.T) {
	resetRootFlagsForTest()

	origRunInteractive := runInteractive
	origRunInteractiveWithResume := runInteractiveWithResume
	origRunHeadless := runHeadless
	origRunOnce := runOnce
	origRunOnceWithImage := runOnceWithImage
	t.Cleanup(func() {
		runInteractive = origRunInteractive
		runInteractiveWithResume = origRunInteractiveWithResume
		runHeadless = origRunHeadless
		runOnce = origRunOnce
		runOnceWithImage = origRunOnceWithImage
		resetRootFlagsForTest()
	})

	interactiveCalled := false
	onceCalled := false
	runInteractive = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
		interactiveCalled = true
	}
	runInteractiveWithResume = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
		interactiveCalled = true
	}
	runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config) *agent.HeadlessResult {
		interactiveCalled = true
		return agent.NewSuccessResult(provider.Name(), model, "", nil, 0)
	}
	runOnceWithImage = func(query string, model string, provider api.Provider, imagePath string, cfg *config.Config, autoApprove bool) {
		interactiveCalled = true
	}
	runOnce = func(query string, model string, provider api.Provider, cfg *config.Config, autoApprove bool, quiet bool) error {
		onceCalled = true
		if query != "hello" {
			t.Fatalf("query = %q, want hello", query)
		}
		return nil
	}

	rootCmd.SetArgs([]string{"--provider", "ollama", "--no-update-check", "hello"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !onceCalled {
		t.Fatal("expected positional query to use one-shot path")
	}
	if interactiveCalled {
		t.Fatal("interactive path must not be executed for positional one-shot query")
	}
}

func TestRootCommand_OnceExecutesSingleTurn(t *testing.T) {
	resetRootFlagsForTest()

	origRunInteractive := runInteractive
	origRunInteractiveWithResume := runInteractiveWithResume
	origRunHeadless := runHeadless
	origRunOnce := runOnce
	origRunOnceWithImage := runOnceWithImage
	t.Cleanup(func() {
		runInteractive = origRunInteractive
		runInteractiveWithResume = origRunInteractiveWithResume
		runHeadless = origRunHeadless
		runOnce = origRunOnce
		runOnceWithImage = origRunOnceWithImage
		resetRootFlagsForTest()
	})

	interactiveCalled := false
	onceCalled := false
	runInteractive = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
		interactiveCalled = true
	}
	runInteractiveWithResume = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
		interactiveCalled = true
	}
	runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config) *agent.HeadlessResult {
		interactiveCalled = true
		return nil
	}
	runOnceWithImage = func(query string, model string, provider api.Provider, imagePath string, cfg *config.Config, autoApprove bool) {
		interactiveCalled = true
	}
	runOnce = func(query string, model string, provider api.Provider, cfg *config.Config, autoApprove bool, quiet bool) error {
		onceCalled = true
		if query != "hello" {
			t.Fatalf("query = %q, want hello", query)
		}
		return nil
	}

	rootCmd.SetArgs([]string{"--once", "--provider", "ollama", "--no-update-check", "hello"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !onceCalled {
		t.Fatal("expected --once path to be executed")
	}
	if interactiveCalled {
		t.Fatal("interactive path must not be executed when --once is set")
	}
}

func TestRootCommand_InteractiveFlagForcesREPLWithPositionalQuery(t *testing.T) {
	resetRootFlagsForTest()

	origRunInteractive := runInteractive
	origRunTUI := runTUI
	origRunOnce := runOnce
	t.Cleanup(func() {
		runInteractive = origRunInteractive
		runTUI = origRunTUI
		runOnce = origRunOnce
		resetRootFlagsForTest()
	})

	interactiveCalled := false
	onceCalled := false
	runInteractive = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
		interactiveCalled = true
	}
	runTUI = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
		interactiveCalled = true
	}
	runOnce = func(query string, model string, provider api.Provider, cfg *config.Config, autoApprove bool, quiet bool) error {
		onceCalled = true
		return nil
	}

	rootCmd.SetArgs([]string{"--interactive", "--provider", "ollama", "--no-update-check", "hello"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !interactiveCalled {
		t.Fatal("expected --interactive to use interactive path")
	}
	if onceCalled {
		t.Fatal("one-shot path must not be executed when --interactive is set")
	}
}

func TestRootCommand_InteractiveWithOnceReturnsError(t *testing.T) {
	resetRootFlagsForTest()
	rootCmd.SetArgs([]string{"--interactive", "--once", "--provider", "ollama", "--no-update-check", "hello"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for --interactive with --once")
	}
	if !strings.Contains(err.Error(), "--interactive cannot be used with --once") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestRootCommand_OnceRequiresQuery(t *testing.T) {
	resetRootFlagsForTest()
	rootCmd.SetArgs([]string{"--once", "--no-update-check"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when --once has no query")
	}
}

func TestRootCommand_OnceWithResumeReturnsError(t *testing.T) {
	resetRootFlagsForTest()
	rootCmd.SetArgs([]string{"--once", "--resume", "--no-update-check", "hello"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for --once with --resume")
	}
}

func TestRootCommand_ResumeWithPositionalQueryReturnsError(t *testing.T) {
	resetRootFlagsForTest()
	rootCmd.SetArgs([]string{"--resume", "--provider", "ollama", "--no-update-check", "hello"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for --resume with positional query")
	}
	if !strings.Contains(err.Error(), "--resume cannot be used with query arguments") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestRootCommand_OnceWithHeadlessReturnsError(t *testing.T) {
	resetRootFlagsForTest()
	rootCmd.SetArgs([]string{"--once", "--headless", "--no-update-check", "hello"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for --once with --headless")
	}
}

func TestRootCommand_QuietWithoutOnceReturnsError(t *testing.T) {
	resetRootFlagsForTest()
	rootCmd.SetArgs([]string{"--quiet", "--no-update-check"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for --quiet without one-shot execution")
	}
	if !strings.Contains(err.Error(), "--quiet can only be used with one-shot execution") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestRootCommand_InteractiveWithQuietReturnsError(t *testing.T) {
	resetRootFlagsForTest()
	rootCmd.SetArgs([]string{"--interactive", "--quiet", "--provider", "ollama", "--no-update-check", "hello"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for --interactive with --quiet")
	}
	if !strings.Contains(err.Error(), "--quiet can only be used with one-shot execution") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestRootCommand_QuietWithPositionalQueryUsesImplicitOnce(t *testing.T) {
	resetRootFlagsForTest()

	origRunOnce := runOnce
	t.Cleanup(func() {
		runOnce = origRunOnce
		resetRootFlagsForTest()
	})

	called := false
	runOnce = func(query string, model string, provider api.Provider, cfg *config.Config, autoApprove bool, quiet bool) error {
		called = true
		if !quiet {
			t.Fatal("expected quiet to be passed to one-shot execution")
		}
		if query != "hello" {
			t.Fatalf("query = %q, want hello", query)
		}
		return nil
	}

	rootCmd.SetArgs([]string{"--quiet", "--provider", "ollama", "--no-update-check", "hello"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !called {
		t.Fatal("expected implicit one-shot execution")
	}
}

func TestRootCommand_OnceMultiWordQuery(t *testing.T) {
	resetRootFlagsForTest()

	origRunOnce := runOnce
	t.Cleanup(func() {
		runOnce = origRunOnce
		resetRootFlagsForTest()
	})

	var gotQuery string
	runOnce = func(query string, model string, provider api.Provider, cfg *config.Config, autoApprove bool, quiet bool) error {
		gotQuery = query
		return nil
	}

	rootCmd.SetArgs([]string{"--once", "--provider", "ollama", "--no-update-check", "fix", "this", "bug"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	want := "fix this bug"
	if gotQuery != want {
		t.Fatalf("query = %q, want %q", gotQuery, want)
	}
}

func TestRootCommand_OnceErrorPropagation(t *testing.T) {
	resetRootFlagsForTest()

	origRunOnce := runOnce
	t.Cleanup(func() {
		runOnce = origRunOnce
		resetRootFlagsForTest()
	})

	runOnce = func(query string, model string, provider api.Provider, cfg *config.Config, autoApprove bool, quiet bool) error {
		return fmt.Errorf("something went wrong")
	}

	rootCmd.SetArgs([]string{"--once", "--provider", "ollama", "--no-update-check", "hello"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error to propagate from runOnce")
	}
	if !strings.Contains(err.Error(), "something went wrong") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRootCommand_PositionalQueryUsesHeadlessInJSONMode(t *testing.T) {
	resetRootFlagsForTest()

	origRunHeadless := runHeadless
	origRunOnce := runOnce
	origRunInteractive := runInteractive
	t.Cleanup(func() {
		runHeadless = origRunHeadless
		runOnce = origRunOnce
		runInteractive = origRunInteractive
		resetRootFlagsForTest()
	})

	headlessCalled := false
	onceCalled := false
	interactiveCalled := false
	runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config) *agent.HeadlessResult {
		headlessCalled = true
		if query != "hello" {
			t.Fatalf("query = %q, want hello", query)
		}
		return agent.NewSuccessResult(provider.Name(), model, "ok", nil, 0)
	}
	runOnce = func(query string, model string, provider api.Provider, cfg *config.Config, autoApprove bool, quiet bool) error {
		onceCalled = true
		return nil
	}
	runInteractive = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
		interactiveCalled = true
	}

	rootCmd.SetArgs([]string{"--output-format", "json", "--provider", "ollama", "--no-update-check", "hello"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !headlessCalled {
		t.Fatal("expected JSON mode to use headless path")
	}
	if onceCalled {
		t.Fatal("one-shot path must not be executed in JSON mode")
	}
	if interactiveCalled {
		t.Fatal("interactive path must not be executed in JSON mode")
	}
}

func TestRootCommand_HeadlessJSONStdoutIsPureJSON(t *testing.T) {
	resetRootFlagsForTest()
	t.Setenv("HOME", t.TempDir())

	origRunHeadless := runHeadless
	t.Cleanup(func() {
		runHeadless = origRunHeadless
		resetRootFlagsForTest()
	})

	tempDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	targetPath := filepath.Join(tempDir, "input.txt")
	if err := os.WriteFile(targetPath, []byte("hello from headless\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	provider := &headlessSequenceProvider{
		name: "headless-seq",
		responses: []string{
			fmt.Sprintf(`{"tool":"gather_context","args":{"query":%q}}`, targetPath),
			"done",
		},
	}

	runHeadless = func(ctx context.Context, query string, model string, providerArg api.Provider, cfg *config.Config) *agent.HeadlessResult {
		if cfg == nil {
			cfg = config.DefaultConfig()
		} else {
			cloned := *cfg
			cfg = &cloned
		}
		cfg.ProjectMap.Enabled = false
		return agent.RunHeadlessWithConfig(ctx, query, model, provider, cfg)
	}

	oldStdout := os.Stdout
	oldColorOutput := color.Output
	r, w, _ := os.Pipe()
	os.Stdout = w
	color.Output = w

	rootCmd.SetArgs([]string{"--output-format", "json", "--provider", "ollama", "--no-update-check", "hello"})
	execErr := rootCmd.Execute()

	_ = w.Close()
	os.Stdout = oldStdout
	color.Output = oldColorOutput

	if execErr != nil {
		t.Fatalf("Execute() error = %v", execErr)
	}

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := strings.TrimSpace(buf.String())

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("stdout is not pure JSON: %v\noutput=%q", err, output)
	}
	if parsed["status"] != "success" {
		t.Fatalf("status = %v, want success", parsed["status"])
	}
	if parsed["response"] != "done" {
		t.Fatalf("response = %v, want done", parsed["response"])
	}
}

func TestRootCommand_ImageFlagPreservesImagePath(t *testing.T) {
	resetRootFlagsForTest()

	origRunOnce := runOnce
	origRunInteractive := runInteractive
	origRunOnceWithImage := runOnceWithImage
	t.Cleanup(func() {
		runOnce = origRunOnce
		runInteractive = origRunInteractive
		runOnceWithImage = origRunOnceWithImage
		resetRootFlagsForTest()
	})

	imageCalled := false
	onceCalled := false
	interactiveCalled := false
	runOnce = func(query string, model string, provider api.Provider, cfg *config.Config, autoApprove bool, quiet bool) error {
		onceCalled = true
		return nil
	}
	runInteractive = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
		interactiveCalled = true
	}
	runOnceWithImage = func(query string, model string, provider api.Provider, imagePath string, cfg *config.Config, autoApprove bool) {
		imageCalled = true
		if query != "describe" {
			t.Fatalf("query = %q, want describe", query)
		}
		if imagePath != "/tmp/image.png" {
			t.Fatalf("imagePath = %q, want /tmp/image.png", imagePath)
		}
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
