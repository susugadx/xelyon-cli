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
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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
	legacyNoTUI = false
	doctorDeploymentFlag = ""
	doctorCatalogModelFlag = ""
	doctorBedrockModelFlag = ""
	doctorClaudeModelFlag = ""
	doctorDeepSeekModelFlag = ""
	doctorGeminiModelFlag = ""
	doctorGroqModelFlag = ""
	doctorKimiModelFlag = ""
	doctorOllamaModelFlag = ""
	doctorOpenAIModelFlag = ""
	doctorOpenAISubscriptionModelFlag = ""
	doctorOpenRouterModelFlag = ""
	doctorSmokeFlag = false
	doctorToolSmokeFlag = false
	doctorCapabilitiesFlag = false
	doctorRequiredCapabilityFlags = nil
	doctorAzureRetentionSmokeFlag = false
	doctorOpenAIRetentionSmokeFlag = false
	doctorOpenAISubscriptionRetentionSmokeFlag = false
	doctorOpenAISubscriptionCacheSmokeFlag = false
	doctorOpenAISubscriptionCompactSmokeFlag = false
	doctorOpenAISubscriptionThinkingSmokeFlag = false
	doctorBedrockImageSmokeFlag = false
	doctorBedrockThinkingSmokeFlag = false
	doctorClaudeImageSmokeFlag = false
	doctorClaudeThinkingSmokeFlag = false
	doctorClaudeWebSearchSmokeFlag = false
	doctorGeminiImageSmokeFlag = false
	doctorGeminiWebSearchSmokeFlag = false
	doctorKimiImageSmokeFlag = false
	doctorKimiWebSearchSmokeFlag = false
	doctorTimeoutFlag = defaultDoctorTimeout
	doctorJSONFlag = false
	doctorPrintConfigFlag = false
	doctorPrintRequestFlag = false
	resetCommandFlagsForTest(rootCmd)
	doctorRequiredCapabilityFlags = nil
}

func newDoctorSubcommandTest(t *testing.T, newCommand func() *cobra.Command) (*cobra.Command, *bytes.Buffer) {
	t.Helper()
	resetRootFlagsForTest()
	t.Cleanup(resetRootFlagsForTest)

	var out bytes.Buffer
	cmd := newCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	return cmd, &out
}

func newRootCommandExecutionTest(t *testing.T) *bytes.Buffer {
	t.Helper()
	resetRootFlagsForTest()
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
		resetRootFlagsForTest()
	})

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	return &out
}

func resetCommandFlagsForTest(cmd *cobra.Command) {
	if cmd == nil {
		return
	}
	cmd.SilenceUsage = false
	resetFlagSetForTest(cmd.Flags())
	resetFlagSetForTest(cmd.PersistentFlags())
	resetFlagSetForTest(cmd.InheritedFlags())
	for _, child := range cmd.Commands() {
		resetCommandFlagsForTest(child)
	}
}

func resetFlagSetForTest(flags interface {
	VisitAll(fn func(*pflag.Flag))
}) {
	flags.VisitAll(func(flag *pflag.Flag) {
		flag.Changed = false
		_ = flag.Value.Set(flag.DefValue)
	})
}

type rootCommandRunners struct {
	runLegacyInteractive           func(string, api.Provider, *config.Config, bool)
	runLegacyInteractiveWithResume func(string, api.Provider, *config.Config, bool)
	runLegacyInteractiveWithImage  func(string, string, api.Provider, string, *config.Config, bool) error
	runTUI                         func(string, api.Provider, *config.Config, bool)
	runTUIWithResume               func(string, api.Provider, *config.Config, bool) error
	runTUIWithResumeDirect         func(string, api.Provider, *config.Config, bool, string) error
	runTUIWithResumePicker         func(string, api.Provider, *config.Config, bool, bool)
	runTUIWithImage                func(string, string, api.Provider, string, *config.Config, bool) error
	runHeadless                    func(context.Context, string, string, api.Provider, *config.Config) *agent.HeadlessResult
	runOnce                        func(string, string, api.Provider, *config.Config, bool, bool) error
	runOnceWithImage               func(string, string, api.Provider, string, *config.Config, bool, bool) error
}

func snapshotRootCommandRunners() rootCommandRunners {
	return rootCommandRunners{
		runLegacyInteractive:           runLegacyInteractive,
		runLegacyInteractiveWithResume: runLegacyInteractiveWithResume,
		runLegacyInteractiveWithImage:  runLegacyInteractiveWithImage,
		runTUI:                         runTUI,
		runTUIWithResume:               runTUIWithResume,
		runTUIWithResumeDirect:         runTUIWithResumeDirect,
		runTUIWithResumePicker:         runTUIWithResumePicker,
		runTUIWithImage:                runTUIWithImage,
		runHeadless:                    runHeadless,
		runOnce:                        runOnce,
		runOnceWithImage:               runOnceWithImage,
	}
}

func restoreRootCommandRunners(r rootCommandRunners) {
	runLegacyInteractive = r.runLegacyInteractive
	runLegacyInteractiveWithResume = r.runLegacyInteractiveWithResume
	runLegacyInteractiveWithImage = r.runLegacyInteractiveWithImage
	runTUI = r.runTUI
	runTUIWithResume = r.runTUIWithResume
	runTUIWithResumeDirect = r.runTUIWithResumeDirect
	runTUIWithResumePicker = r.runTUIWithResumePicker
	runTUIWithImage = r.runTUIWithImage
	runHeadless = r.runHeadless
	runOnce = r.runOnce
	runOnceWithImage = r.runOnceWithImage
}

func withRootCommandTest(t *testing.T) {
	t.Helper()

	originalRunners := snapshotRootCommandRunners()
	resetRootFlagsForTest()
	rootCmd.SetArgs(nil)

	t.Cleanup(func() {
		restoreRootCommandRunners(originalRunners)
		resetRootFlagsForTest()
		rootCmd.SetArgs(nil)
	})
}

func TestRootCommand_PositionalQueryDefaultsToOnce(t *testing.T) {
	withRootCommandTest(t)

	interactiveCalled := false
	onceCalled := false
	runLegacyInteractive = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
		interactiveCalled = true
	}
	runLegacyInteractiveWithResume = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
		interactiveCalled = true
	}
	runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config) *agent.HeadlessResult {
		interactiveCalled = true
		return agent.NewSuccessResult(provider.Name(), model, "", nil, 0)
	}
	runOnceWithImage = func(query string, model string, provider api.Provider, imagePath string, cfg *config.Config, autoApprove bool, quiet bool) error {
		interactiveCalled = true
		return nil
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
	withRootCommandTest(t)

	interactiveCalled := false
	onceCalled := false
	runLegacyInteractive = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
		interactiveCalled = true
	}
	runLegacyInteractiveWithResume = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
		interactiveCalled = true
	}
	runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config) *agent.HeadlessResult {
		interactiveCalled = true
		return nil
	}
	runOnceWithImage = func(query string, model string, provider api.Provider, imagePath string, cfg *config.Config, autoApprove bool, quiet bool) error {
		interactiveCalled = true
		return nil
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

func TestRootCommand_InteractiveFlagForcesTUIWithPositionalQuery(t *testing.T) {
	withRootCommandTest(t)

	interactiveCalled := false
	onceCalled := false
	runLegacyInteractive = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
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

func TestRootCommand_NoTUIInteractiveUsesLegacyPath(t *testing.T) {
	withRootCommandTest(t)

	legacyCalled := false
	tuiCalled := false
	runLegacyInteractive = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
		legacyCalled = true
	}
	runTUI = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
		tuiCalled = true
	}

	rootCmd.SetArgs([]string{"--interactive", "--no-tui", "--provider", "ollama", "--no-update-check"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !legacyCalled {
		t.Fatal("expected --no-tui --interactive to use legacy path")
	}
	if tuiCalled {
		t.Fatal("TUI path must not be executed when --no-tui is set")
	}
}

func TestRootCommand_InteractiveWithOnceReturnsError(t *testing.T) {
	withRootCommandTest(t)
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
	withRootCommandTest(t)
	rootCmd.SetArgs([]string{"--once", "--no-update-check"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when --once has no query")
	}
}

func TestRootCommand_OnceWithResumeReturnsError(t *testing.T) {
	withRootCommandTest(t)
	rootCmd.SetArgs([]string{"--once", "--resume", "--no-update-check", "hello"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for --once with --resume")
	}
}

func TestRootCommand_ResumeWithPositionalQueryReturnsError(t *testing.T) {
	withRootCommandTest(t)
	rootCmd.SetArgs([]string{"--resume", "--provider", "ollama", "--no-update-check", "hello"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for --resume with positional query")
	}
	if !strings.Contains(err.Error(), "--resume cannot be used with query arguments") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestRootCommand_ResumeUsesTUIPath(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())
	seedOllamaResumeSession(t, "qwen2.5-coder:14b")

	tuiCalled := false
	legacyCalled := false
	runTUIWithResume = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) error {
		tuiCalled = true
		return nil
	}
	runLegacyInteractiveWithResume = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
		legacyCalled = true
	}

	rootCmd.SetArgs([]string{"--resume", "--provider", "ollama", "--no-update-check"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !tuiCalled {
		t.Fatal("expected --resume to use TUI resume path")
	}
	if legacyCalled {
		t.Fatal("legacy resume path must not be executed by default")
	}
}

func TestRootCommand_ResumePropagatesTUIError(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())
	seedOllamaResumeSession(t, "qwen2.5-coder:14b")

	runTUIWithResume = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) error {
		return fmt.Errorf("resume failed")
	}

	rootCmd.SetArgs([]string{"--resume", "--provider", "ollama", "--no-update-check"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected --resume error")
	}
	if !strings.Contains(err.Error(), "resume failed") {
		t.Fatalf("error = %v, want resume failed", err)
	}
}

func TestRootCommand_NoTUIResumeUsesLegacyPath(t *testing.T) {
	withRootCommandTest(t)

	tuiCalled := false
	legacyCalled := false
	runTUIWithResume = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) error {
		tuiCalled = true
		return nil
	}
	runLegacyInteractiveWithResume = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
		legacyCalled = true
	}

	rootCmd.SetArgs([]string{"--resume", "--no-tui", "--provider", "ollama", "--no-update-check"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !legacyCalled {
		t.Fatal("expected --no-tui --resume to use legacy resume path")
	}
	if tuiCalled {
		t.Fatal("TUI resume path must not be executed when --no-tui is set")
	}
}

func TestResumeCommand_DefaultOpensPicker(t *testing.T) {
	withRootCommandTest(t)

	var pickerCalled bool
	var pickerAll bool
	runTUIWithResumePicker = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool, all bool) {
		pickerCalled = true
		pickerAll = all
	}

	rootCmd.SetArgs([]string{"resume", "--provider", "ollama", "--no-update-check"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !pickerCalled {
		t.Fatal("expected resume to open picker")
	}
	if pickerAll {
		t.Fatal("resume without --all should use cwd-scoped picker")
	}
}

func TestResumeCommand_AllOpensAllSessionPicker(t *testing.T) {
	withRootCommandTest(t)

	var pickerCalled bool
	var pickerAll bool
	runTUIWithResumePicker = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool, all bool) {
		pickerCalled = true
		pickerAll = all
	}

	rootCmd.SetArgs([]string{"resume", "--all", "--provider", "ollama", "--no-update-check"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !pickerCalled || !pickerAll {
		t.Fatalf("pickerCalled=%v pickerAll=%v, want all picker", pickerCalled, pickerAll)
	}
}

func TestResumeCommand_LastUsesResumePath(t *testing.T) {
	withRootCommandTest(t)

	var resumeCalled bool
	runTUIWithResume = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) error {
		resumeCalled = true
		return nil
	}

	rootCmd.SetArgs([]string{"resume", "--last", "--provider", "ollama", "--no-update-check"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !resumeCalled {
		t.Fatal("expected resume --last to use last-session path")
	}
}

func TestResumeCommand_LastPropagatesResumeError(t *testing.T) {
	withRootCommandTest(t)

	runTUIWithResume = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) error {
		return fmt.Errorf("resume last failed")
	}

	rootCmd.SetArgs([]string{"resume", "--last", "--provider", "ollama", "--no-update-check"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected resume --last error")
	}
	if !strings.Contains(err.Error(), "resume last failed") {
		t.Fatalf("error = %v, want resume last failed", err)
	}
}

func TestResumeCommand_SessionIDUsesDirectPath(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())
	sessionID := seedOllamaResumeSession(t, "qwen2.5-coder:14b")

	var gotSessionID string
	runTUIWithResumeDirect = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool, sessionID string) error {
		gotSessionID = sessionID
		return nil
	}

	rootCmd.SetArgs([]string{"resume", sessionID, "--provider", "ollama", "--no-update-check"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if gotSessionID != sessionID {
		t.Fatalf("sessionID = %q, want %q", gotSessionID, sessionID)
	}
}

func TestResumeCommand_DirectPathPropagatesError(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())
	sessionID := seedOllamaResumeSession(t, "qwen2.5-coder:14b")

	runTUIWithResumeDirect = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool, sessionID string) error {
		return fmt.Errorf("resume failed")
	}

	rootCmd.SetArgs([]string{"resume", sessionID, "--provider", "ollama", "--no-update-check"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected direct resume error")
	}
	if !strings.Contains(err.Error(), "resume failed") {
		t.Fatalf("error = %v, want resume failed", err)
	}
}

func TestResumeCommand_RejectsAllWithSessionID(t *testing.T) {
	withRootCommandTest(t)

	rootCmd.SetArgs([]string{"resume", "--all", "session-42", "--provider", "ollama", "--no-update-check"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for --all with session ID")
	}
	if !strings.Contains(err.Error(), "--all cannot be used with a session ID") {
		t.Fatalf("error = %v, want --all/session-id conflict", err)
	}
}

func TestRootCommand_OnceWithHeadlessReturnsError(t *testing.T) {
	withRootCommandTest(t)
	rootCmd.SetArgs([]string{"--once", "--headless", "--no-update-check", "hello"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for --once with --headless")
	}
}

func TestRootCommand_QuietWithoutOnceReturnsError(t *testing.T) {
	withRootCommandTest(t)
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
	withRootCommandTest(t)
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
	withRootCommandTest(t)

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
	withRootCommandTest(t)

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
	withRootCommandTest(t)

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
	withRootCommandTest(t)

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
	runLegacyInteractive = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
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

func TestRootCommand_OutputFormatIsCaseInsensitive(t *testing.T) {
	withRootCommandTest(t)

	headlessCalled := false
	runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config) *agent.HeadlessResult {
		headlessCalled = true
		if query != "hello" {
			t.Fatalf("query = %q, want hello", query)
		}
		return agent.NewSuccessResult(provider.Name(), model, "ok", nil, 0)
	}

	rootCmd.SetArgs([]string{"--output-format", "JSON", "--provider", "ollama", "--no-update-check", "hello"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !headlessCalled {
		t.Fatal("expected normalized JSON mode to use headless path")
	}
}

func TestRootCommand_InvalidOutputFormatReturnsError(t *testing.T) {
	withRootCommandTest(t)
	rootCmd.SetArgs([]string{"--output-format", "yaml", "--no-update-check", "hello"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid output format")
	}
	if !strings.Contains(err.Error(), "invalid --output-format") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestRootCommand_HeadlessJSONStdoutIsPureJSON(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())

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

func TestRootCommand_HeadlessErrorReturnsErrorAfterPrintingJSON(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())

	runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config) *agent.HeadlessResult {
		return agent.NewToolLoopLimitResult(provider.Name(), model, 10, nil, 0)
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = oldStdout
	})

	rootCmd.SetArgs([]string{"--headless", "--provider", "ollama", "--no-update-check", "hello"})
	execErr := rootCmd.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	if execErr == nil {
		t.Fatal("expected headless error status to return command error")
	}
	if !strings.Contains(execErr.Error(), "headless execution failed") {
		t.Fatalf("unexpected error: %v", execErr)
	}

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := strings.TrimSpace(buf.String())

	var parsed agent.HeadlessResult
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("stdout is not headless JSON: %v\noutput=%q", err, output)
	}
	if parsed.Status != agent.HeadlessStatusError {
		t.Fatalf("status = %q, want %q", parsed.Status, agent.HeadlessStatusError)
	}
	if parsed.Error == nil || parsed.Error.Type != agent.HeadlessErrorTypeToolLoopLimit {
		t.Fatalf("error = %+v, want %s", parsed.Error, agent.HeadlessErrorTypeToolLoopLimit)
	}
}

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

func TestRootCommand_ImageWithJSONReturnsError(t *testing.T) {
	withRootCommandTest(t)
	rootCmd.SetArgs([]string{"--image", "/tmp/image.png", "--output-format", "json", "--provider", "ollama", "--no-update-check", "describe"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for --image with JSON output")
	}
	if !strings.Contains(err.Error(), "--image cannot be used with --headless or --output-format json") {
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
