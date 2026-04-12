package agent

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/stdio"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func withTempWorkdir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("os.Chdir() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })
	return dir
}

func withDefaultSTDIO(t *testing.T, in *strings.Reader, out *bytes.Buffer) {
	t.Helper()
	stdio.SetDefaults(in, out, out)
	t.Cleanup(func() {
		stdio.SetDefaults(nil, nil, nil)
	})
}

func TestRunInteractiveWithConfig_ProcessesSingleInputAndCleansUp(t *testing.T) {
	disableColors(t)
	withTempWorkdir(t)
	t.Setenv("HOME", t.TempDir())

	var out bytes.Buffer
	withDefaultSTDIO(t, strings.NewReader("hello\n"), &out)

	var cleanupCount atomic.Int32
	cleanupHook = func() { cleanupCount.Add(1) }
	defer func() { cleanupHook = nil }()

	provider := &scriptedChatProvider{name: "openai", functionCalling: true}
	cfg := newProjectMapDisabledConfig()
	cfg.Paste.BracketedPaste = false

	RunInteractiveWithConfig("test-model", provider, cfg, true)

	if provider.callCount != 1 {
		t.Fatalf("provider.callCount = %d, want 1", provider.callCount)
	}
	if cleanupCount.Load() != 1 {
		t.Fatalf("cleanup count = %d, want 1", cleanupCount.Load())
	}

	got := out.String()
	for _, fragment := range []string{
		"Provider: openai | Model: test-model",
		"Mode: Auto-approve",
		"Context size:",
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("RunInteractiveWithConfig() output missing %q:\n%s", fragment, got)
		}
	}
}

func TestRunInteractiveWithImageWithConfig_SendsInitialImageTurn(t *testing.T) {
	disableColors(t)
	workdir := withTempWorkdir(t)
	t.Setenv("HOME", t.TempDir())

	imagePath := filepath.Join(workdir, "test.png")
	if err := os.WriteFile(imagePath, []byte("fake png data for testing"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var out bytes.Buffer
	withDefaultSTDIO(t, strings.NewReader(""), &out)

	var cleanupCount atomic.Int32
	cleanupHook = func() { cleanupCount.Add(1) }
	defer func() { cleanupHook = nil }()

	provider := &imageOnceProvider{}
	cfg := newProjectMapDisabledConfig()
	cfg.Paste.BracketedPaste = false

	if err := RunInteractiveWithImageWithConfig("describe", "test-model", provider, imagePath, cfg, false); err != nil {
		t.Fatalf("RunInteractiveWithImageWithConfig() error = %v", err)
	}
	if provider.imageCalls != 1 {
		t.Fatalf("provider.imageCalls = %d, want 1", provider.imageCalls)
	}
	if cleanupCount.Load() != 1 {
		t.Fatalf("cleanup count = %d, want 1", cleanupCount.Load())
	}
	if !strings.Contains(out.String(), "Image loaded:") || !strings.Contains(out.String(), "Sending image:") {
		t.Fatalf("RunInteractiveWithImageWithConfig() output = %q, want image status lines", out.String())
	}
}

func TestRunInteractiveWithImageWithConfig_UnsupportedProviderReturnsError(t *testing.T) {
	disableColors(t)
	withTempWorkdir(t)
	t.Setenv("HOME", t.TempDir())

	var out bytes.Buffer
	withDefaultSTDIO(t, strings.NewReader(""), &out)

	cfg := newProjectMapDisabledConfig()
	cfg.Paste.BracketedPaste = false

	err := RunInteractiveWithImageWithConfig("describe", "test-model", &mockProvider{name: "ollama"}, "missing.png", cfg, false)
	if err == nil {
		t.Fatal("expected unsupported provider error")
	}
	if !strings.Contains(err.Error(), "does not support image input") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunInteractiveWithImageWithConfig_EmptyQueryUsesDefaultPrompt(t *testing.T) {
	disableColors(t)
	workdir := withTempWorkdir(t)
	t.Setenv("HOME", t.TempDir())

	imagePath := filepath.Join(workdir, "test.png")
	if err := os.WriteFile(imagePath, []byte("fake png data for testing"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var out bytes.Buffer
	withDefaultSTDIO(t, strings.NewReader(""), &out)

	cfg := newProjectMapDisabledConfig()
	cfg.Paste.BracketedPaste = false
	provider := &imageCapableChatProvider{name: "openai"}

	if err := RunInteractiveWithImageWithConfig("", "test-model", provider, imagePath, cfg, false); err != nil {
		t.Fatalf("RunInteractiveWithImageWithConfig() error = %v", err)
	}
	if provider.imageCalls != 1 {
		t.Fatalf("provider.imageCalls = %d, want 1", provider.imageCalls)
	}
	if !strings.Contains(provider.lastMessage, "Please analyze this image.") {
		t.Fatalf("provider.lastMessage = %q, want default image prompt", provider.lastMessage)
	}
}

func TestRunInteractiveWithResumeWithConfig_ResumesLastSession(t *testing.T) {
	disableColors(t)
	withTempWorkdir(t)
	t.Setenv("HOME", t.TempDir())

	storage, err := history.NewStorage()
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}
	session := history.NewSession("test-model")
	session.AddMessage("user", "previous question", "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	var out bytes.Buffer
	withDefaultSTDIO(t, strings.NewReader("follow up\n"), &out)

	var cleanupCount atomic.Int32
	cleanupHook = func() { cleanupCount.Add(1) }
	defer func() { cleanupHook = nil }()

	provider := &scriptedChatProvider{name: "openai", functionCalling: true}
	cfg := newProjectMapDisabledConfig()
	cfg.Paste.BracketedPaste = false

	RunInteractiveWithResumeWithConfig("test-model", provider, cfg, false)

	if provider.callCount != 1 {
		t.Fatalf("provider.callCount = %d, want 1", provider.callCount)
	}
	if cleanupCount.Load() != 1 {
		t.Fatalf("cleanup count = %d, want 1", cleanupCount.Load())
	}
	if !strings.Contains(out.String(), "Resumed session") {
		t.Fatalf("RunInteractiveWithResumeWithConfig() output = %q, want resume message", out.String())
	}
}

func TestRunInteractiveWithResumeWithConfig_FallsBackWhenNoSessionExists(t *testing.T) {
	disableColors(t)
	withTempWorkdir(t)
	t.Setenv("HOME", t.TempDir())

	var out bytes.Buffer
	withDefaultSTDIO(t, strings.NewReader("hello\n"), &out)

	var cleanupCount atomic.Int32
	cleanupHook = func() { cleanupCount.Add(1) }
	defer func() { cleanupHook = nil }()

	provider := &scriptedChatProvider{name: "openai", functionCalling: true}
	cfg := newProjectMapDisabledConfig()
	cfg.Paste.BracketedPaste = false

	RunInteractiveWithResumeWithConfig("test-model", provider, cfg, false)

	if provider.callCount != 1 {
		t.Fatalf("provider.callCount = %d, want 1", provider.callCount)
	}
	if cleanupCount.Load() != 1 {
		t.Fatalf("cleanup count = %d, want 1", cleanupCount.Load())
	}
	if !strings.Contains(out.String(), "No previous session found") {
		t.Fatalf("RunInteractiveWithResumeWithConfig() output = %q, want fallback message", out.String())
	}
}

func TestRunInteractiveWithResumeWithConfig_FallsBackWhenStorageInitFails(t *testing.T) {
	disableColors(t)
	withTempWorkdir(t)

	homeFile := filepath.Join(t.TempDir(), "home-file")
	if err := os.WriteFile(homeFile, []byte("not a dir"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Setenv("HOME", homeFile)

	var out bytes.Buffer
	withDefaultSTDIO(t, strings.NewReader("hello\n"), &out)

	var cleanupCount atomic.Int32
	cleanupHook = func() { cleanupCount.Add(1) }
	defer func() { cleanupHook = nil }()

	provider := &scriptedChatProvider{name: "openai", functionCalling: true}
	cfg := newProjectMapDisabledConfig()
	cfg.Paste.BracketedPaste = false

	RunInteractiveWithResumeWithConfig("test-model", provider, cfg, false)

	if provider.callCount != 1 {
		t.Fatalf("provider.callCount = %d, want 1", provider.callCount)
	}
	if cleanupCount.Load() != 1 {
		t.Fatalf("cleanup count = %d, want 1", cleanupCount.Load())
	}
	if !strings.Contains(out.String(), "Failed to initialize storage") {
		t.Fatalf("RunInteractiveWithResumeWithConfig() output = %q, want storage init failure", out.String())
	}
}

func TestRunInteractiveWithResumeWithConfig_FallsBackWhenSessionLoadFails(t *testing.T) {
	disableColors(t)
	withTempWorkdir(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	storage, err := history.NewStorage()
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}
	session := history.NewSession("test-model")
	session.AddMessage("user", "previous question", "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	sessionPath := filepath.Join(home, ".xelyon", "history", session.ID+".jsonl")
	if err := os.Remove(sessionPath); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	var out bytes.Buffer
	withDefaultSTDIO(t, strings.NewReader("hello\n"), &out)

	var cleanupCount atomic.Int32
	cleanupHook = func() { cleanupCount.Add(1) }
	defer func() { cleanupHook = nil }()

	provider := &scriptedChatProvider{name: "openai", functionCalling: true}
	cfg := newProjectMapDisabledConfig()
	cfg.Paste.BracketedPaste = false

	RunInteractiveWithResumeWithConfig("test-model", provider, cfg, false)

	if provider.callCount != 1 {
		t.Fatalf("provider.callCount = %d, want 1", provider.callCount)
	}
	if cleanupCount.Load() != 1 {
		t.Fatalf("cleanup count = %d, want 1", cleanupCount.Load())
	}
	if !strings.Contains(out.String(), "Failed to load session") {
		t.Fatalf("RunInteractiveWithResumeWithConfig() output = %q, want load failure", out.String())
	}
}

func TestSetupSignalHandler_HelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_XELYON_AGENT_SIGNAL_HELPER") != "1" {
		return
	}

	cleanupHook = func() {
		fmt.Fprintln(os.Stdout, "cleanup-hook")
		cleanupHook = nil
	}

	agent := &Agent{
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader(""), os.Stdout, os.Stdout),
		},
	}
	setupSignalHandler(agent)

	time.Sleep(100 * time.Millisecond)
	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		fmt.Fprintln(os.Stdout, err)
		os.Exit(2)
	}
	_ = proc.Signal(os.Interrupt)
	time.Sleep(200 * time.Millisecond)
	_ = proc.Signal(os.Interrupt)
	time.Sleep(500 * time.Millisecond)
	os.Exit(3)
}

func TestSetupSignalHandler_InterruptTwiceTriggersCleanupAndExit(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}

	cmd := exec.Command(exe, "-test.run=TestSetupSignalHandler_HelperProcess")
	cmd.Env = append(os.Environ(), "GO_WANT_XELYON_AGENT_SIGNAL_HELPER=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 3 {
			t.Fatalf("helper process did not exit from signal handler:\n%s", string(output))
		}
		t.Fatalf("helper process failed: %v\n%s", err, string(output))
	}

	got := string(output)
	for _, fragment := range []string{
		"Interrupted. Press Ctrl+C again within 3 seconds to exit.",
		"Gracefully shutting down...",
		"cleanup-hook",
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("signal helper output missing %q:\n%s", fragment, got)
		}
	}
}
