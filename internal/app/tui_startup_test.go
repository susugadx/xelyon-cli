package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	agentpkg "github.com/susugadx/xelyon-cli/internal/agent"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/tui"
	"github.com/susugadx/xelyon-cli/internal/tuiagent"
)

type blockingTUIImageProvider struct {
	started chan struct{}
	release chan struct{}
	calls   atomic.Int32
}

func newBlockingTUIImageProvider() *blockingTUIImageProvider {
	return &blockingTUIImageProvider{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
}

func (p *blockingTUIImageProvider) Name() string { return "openai" }

func (p *blockingTUIImageProvider) SupportsImages() bool { return true }

func (p *blockingTUIImageProvider) IsFunctionCallingEnabled() bool { return true }

func (p *blockingTUIImageProvider) ChatWithTools(context.Context, string, []api.Message, string) (string, error) {
	return "", fmt.Errorf("ChatWithTools should not be called for initial TUI image")
}

func (p *blockingTUIImageProvider) ChatWithImage(ctx context.Context, _ string, _ []api.Message, _ string, image *api.ImageData, _ string) (string, error) {
	if image == nil {
		return "", fmt.Errorf("image is required")
	}
	if p.calls.Add(1) == 1 {
		close(p.started)
	}
	select {
	case <-p.release:
		return "mock image response", nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func runBlockingStartupSubmission(t *testing.T, startup *tui.StartupSubmission, provider *blockingTUIImageProvider) tea.Msg {
	t.Helper()

	if startup == nil {
		t.Fatal("expected startup submission")
	}
	if startup.Cmd == nil {
		t.Fatal("expected startup command")
	}

	msgCh := make(chan tea.Msg, 1)
	go func() {
		msgCh <- startup.Cmd()
	}()
	select {
	case <-provider.started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for startup image turn to start")
	}

	if provider.calls.Load() != 1 {
		t.Fatalf("provider calls after startup = %d, want 1", provider.calls.Load())
	}
	close(provider.release)

	select {
	case msg := <-msgCh:
		return msg
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for startup image turn to finish")
	}
	return nil
}

func TestRunTUIWithConfig_InitializesTUIAgentAndHeader(t *testing.T) {
	disableColors(t)
	t.Setenv("HOME", t.TempDir())

	originalRunner := runTUIProgram
	defer func() { runTUIProgram = originalRunner }()
	originalAdapterFactory := newTUIAdapter
	defer func() { newTUIAdapter = originalAdapterFactory }()

	var called atomic.Int32
	var gotAdapter *tuiagent.TUIAdapter
	var gotAgent *agentpkg.Agent
	var gotInitialContent string
	runTUIProgram = func(agent tui.AgentInterface, initialContent string, onProgram func(*tea.Program)) {
		called.Add(1)
		typed, ok := agent.(*tuiagent.TUIAdapter)
		if !ok {
			t.Fatalf("agent type = %T, want *tuiagent.TUIAdapter", agent)
		}
		gotAdapter = typed
		gotInitialContent = initialContent
		onProgram(nil)
	}
	newTUIAdapter = func(ag *agentpkg.Agent, sendMsg func(tui.AppendMessageMsg)) *tuiagent.TUIAdapter {
		gotAgent = ag
		return tuiagent.NewTUIAdapter(ag, sendMsg)
	}

	var cleanupCount atomic.Int32
	originalCleanup := cleanupTUIAgent
	cleanupTUIAgent = func(ag *agentpkg.Agent) {
		cleanupCount.Add(1)
		ag.Cleanup()
	}
	defer func() { cleanupTUIAgent = originalCleanup }()

	RunTUIWithConfig("test-model", &mockProvider{name: "openai"}, newProjectMapDisabledConfig(), false)

	if called.Load() != 1 {
		t.Fatalf("runTUIProgram called %d times, want 1", called.Load())
	}
	if gotAdapter == nil {
		t.Fatal("expected TUI adapter to be passed to runner")
	}
	if gotAgent == nil {
		t.Fatal("expected adapter.agent to be initialized")
	}
	if gotAgent.AutoApprove {
		t.Fatal("expected TUI agent to preserve disabled auto-approve")
	}
	if cleanupCount.Load() != 1 {
		t.Fatalf("cleanup count = %d, want 1", cleanupCount.Load())
	}

	stripped := stripANSI(gotInitialContent)
	for _, fragment := range []string{"code-guided agent runtime", "Ready · / opens commands"} {
		if !strings.Contains(stripped, fragment) {
			t.Fatalf("initialContent missing %q:\n%s", fragment, stripped)
		}
	}
}

func TestRunTUIWithConfig_PreservesAutoApproveArgument(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	originalRunner := runTUIProgram
	defer func() { runTUIProgram = originalRunner }()
	originalAdapterFactory := newTUIAdapter
	defer func() { newTUIAdapter = originalAdapterFactory }()

	var gotAgent *agentpkg.Agent
	runTUIProgram = func(agent tui.AgentInterface, _ string, onProgram func(*tea.Program)) {
		if _, ok := agent.(*tuiagent.TUIAdapter); !ok {
			t.Fatalf("agent type = %T, want *tuiagent.TUIAdapter", agent)
		}
		onProgram(nil)
	}
	newTUIAdapter = func(ag *agentpkg.Agent, sendMsg func(tui.AppendMessageMsg)) *tuiagent.TUIAdapter {
		gotAgent = ag
		return tuiagent.NewTUIAdapter(ag, sendMsg)
	}

	RunTUIWithConfig("test-model", &mockProvider{name: "openai"}, newProjectMapDisabledConfig(), true)

	if gotAgent == nil {
		t.Fatal("expected TUI agent")
	}
	if !gotAgent.AutoApprove {
		t.Fatal("expected TUI agent to preserve enabled auto-approve")
	}
}

func TestRunTUIWithResumeWithConfig_LoadsLastSession(t *testing.T) {
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

	originalRunner := runTUIProgram
	defer func() { runTUIProgram = originalRunner }()
	originalAdapterFactory := newTUIAdapter
	defer func() { newTUIAdapter = originalAdapterFactory }()

	var gotAgent *agentpkg.Agent
	var gotInitialContent string
	runTUIProgram = func(agent tui.AgentInterface, initialContent string, onProgram func(*tea.Program)) {
		if _, ok := agent.(*tuiagent.TUIAdapter); !ok {
			t.Fatalf("agent type = %T, want *tuiagent.TUIAdapter", agent)
		}
		gotInitialContent = initialContent
		onProgram(nil)
	}
	newTUIAdapter = func(ag *agentpkg.Agent, sendMsg func(tui.AppendMessageMsg)) *tuiagent.TUIAdapter {
		gotAgent = ag
		return tuiagent.NewTUIAdapter(ag, sendMsg)
	}

	if err := RunTUIWithResumeWithConfig("test-model", &mockProvider{name: "openai"}, newProjectMapDisabledConfig(), false); err != nil {
		t.Fatalf("RunTUIWithResumeWithConfig() error = %v", err)
	}

	if gotAgent == nil {
		t.Fatal("expected TUI adapter and agent")
	}
	if len(gotAgent.History) != 1 || gotAgent.History[0].Content != "previous question" {
		t.Fatalf("agent.History = %#v, want restored session history", gotAgent.History)
	}
	if !strings.Contains(stripANSI(gotInitialContent), "Resumed session") {
		t.Fatalf("initial content missing resume message:\n%s", stripANSI(gotInitialContent))
	}
}

func TestRunTUIWithResumeWithConfig_NoSessionsStartsBlankTUI(t *testing.T) {
	disableColors(t)
	withTempWorkdir(t)
	t.Setenv("HOME", t.TempDir())

	originalRunner := runTUIProgram
	defer func() { runTUIProgram = originalRunner }()
	originalAdapterFactory := newTUIAdapter
	defer func() { newTUIAdapter = originalAdapterFactory }()

	var runnerCalled atomic.Int32
	var gotInitialContent string
	runTUIProgram = func(agent tui.AgentInterface, initialContent string, onProgram func(*tea.Program)) {
		runnerCalled.Add(1)
		gotInitialContent = initialContent
	}
	newTUIAdapter = func(ag *agentpkg.Agent, sendMsg func(tui.AppendMessageMsg)) *tuiagent.TUIAdapter {
		return tuiagent.NewTUIAdapter(ag, sendMsg)
	}

	if err := RunTUIWithResumeWithConfig("test-model", &mockProvider{name: "openai"}, newProjectMapDisabledConfig(), false); err != nil {
		t.Fatalf("RunTUIWithResumeWithConfig() error = %v", err)
	}
	if runnerCalled.Load() != 1 {
		t.Fatalf("runTUIProgram called %d times, want 1", runnerCalled.Load())
	}
	if !strings.Contains(stripANSI(gotInitialContent), "No previous session found") {
		t.Fatalf("initial content missing no-session fallback:\n%s", stripANSI(gotInitialContent))
	}
}

func TestRunTUIWithResumeWithConfig_LoadFailureReturnsErrorBeforeStartingTUI(t *testing.T) {
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
	if err := os.Remove(filepath.Join(home, ".xelyon", "history", session.ID+".jsonl")); err != nil {
		t.Fatalf("Remove(session body) error = %v", err)
	}

	originalRunner := runTUIProgram
	defer func() { runTUIProgram = originalRunner }()
	originalAdapterFactory := newTUIAdapter
	defer func() { newTUIAdapter = originalAdapterFactory }()
	originalCleanup := cleanupTUIAgent
	defer func() { cleanupTUIAgent = originalCleanup }()

	var runnerCalled atomic.Int32
	var adapterCalled atomic.Int32
	var cleanupCalled atomic.Int32
	runTUIProgram = func(agent tui.AgentInterface, initialContent string, onProgram func(*tea.Program)) {
		runnerCalled.Add(1)
	}
	newTUIAdapter = func(ag *agentpkg.Agent, sendMsg func(tui.AppendMessageMsg)) *tuiagent.TUIAdapter {
		adapterCalled.Add(1)
		return tuiagent.NewTUIAdapter(ag, sendMsg)
	}
	cleanupTUIAgent = func(ag *agentpkg.Agent) {
		cleanupCalled.Add(1)
		ag.Cleanup()
	}

	err = RunTUIWithResumeWithConfig("test-model", &mockProvider{name: "openai"}, newProjectMapDisabledConfig(), false)
	if err == nil {
		t.Fatal("expected resume load error")
	}
	if !strings.Contains(err.Error(), "failed to resume session") || !strings.Contains(err.Error(), "load session") {
		t.Fatalf("error = %v, want failed resume load session", err)
	}
	if runnerCalled.Load() != 0 {
		t.Fatalf("runTUIProgram called %d times, want 0", runnerCalled.Load())
	}
	if adapterCalled.Load() != 0 {
		t.Fatalf("newTUIAdapter called %d times, want 0", adapterCalled.Load())
	}
	if cleanupCalled.Load() != 1 {
		t.Fatalf("cleanupTUIAgent called %d times, want 1", cleanupCalled.Load())
	}
}

func TestRunTUIWithResumeSessionWithConfig_ReturnsErrorBeforeStartingTUI(t *testing.T) {
	disableColors(t)
	withTempWorkdir(t)
	t.Setenv("HOME", t.TempDir())

	originalRunner := runTUIProgram
	defer func() { runTUIProgram = originalRunner }()
	originalAdapterFactory := newTUIAdapter
	defer func() { newTUIAdapter = originalAdapterFactory }()

	var runnerCalled atomic.Int32
	var adapterCalled atomic.Int32
	var cleanupCalled atomic.Int32
	runTUIProgram = func(agent tui.AgentInterface, initialContent string, onProgram func(*tea.Program)) {
		runnerCalled.Add(1)
	}
	newTUIAdapter = func(ag *agentpkg.Agent, sendMsg func(tui.AppendMessageMsg)) *tuiagent.TUIAdapter {
		adapterCalled.Add(1)
		return tuiagent.NewTUIAdapter(ag, sendMsg)
	}
	originalCleanup := cleanupTUIAgent
	defer func() { cleanupTUIAgent = originalCleanup }()
	cleanupTUIAgent = func(ag *agentpkg.Agent) {
		cleanupCalled.Add(1)
		ag.Cleanup()
	}

	err := RunTUIWithResumeSessionWithConfig("test-model", &mockProvider{name: "openai"}, newProjectMapDisabledConfig(), false, "missing-session")
	if err == nil {
		t.Fatal("expected resume error")
	}
	if !strings.Contains(err.Error(), "failed to resume session") {
		t.Fatalf("error = %v, want failed to resume session", err)
	}
	if runnerCalled.Load() != 0 {
		t.Fatalf("runTUIProgram called %d times, want 0", runnerCalled.Load())
	}
	if adapterCalled.Load() != 0 {
		t.Fatalf("newTUIAdapter called %d times, want 0", adapterCalled.Load())
	}
	if cleanupCalled.Load() != 1 {
		t.Fatalf("cleanupTUIAgent called %d times, want 1", cleanupCalled.Load())
	}
	storage, err := history.NewStorage()
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}
	sessions, err := storage.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("len(ListSessions()) = %d, want 0 after failed direct resume", len(sessions))
	}
}

func TestRunTUIWithImageWithConfig_RunsInitialImageTurn(t *testing.T) {
	disableColors(t)
	workdir := withTempWorkdir(t)
	t.Setenv("HOME", t.TempDir())

	imagePath := filepath.Join(workdir, "test.png")
	if err := os.WriteFile(imagePath, []byte("fake png data for testing"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	originalRunner := runTUIProgramWithStartupSubmission
	defer func() { runTUIProgramWithStartupSubmission = originalRunner }()

	var gotInitialContent string
	provider := newBlockingTUIImageProvider()
	runTUIProgramWithStartupSubmission = func(agent tui.AgentInterface, initialContent string, startupSubmission *tui.StartupSubmission, onProgram func(*tea.Program)) {
		gotInitialContent = initialContent
		if startupSubmission == nil {
			t.Fatal("expected initial image startup submission")
		}
		if startupSubmission.UserMessage != "describe" {
			t.Fatalf("startup user message = %q, want describe", startupSubmission.UserMessage)
		}
		if provider.calls.Load() != 0 {
			t.Fatalf("image turn ran before TUI startup: calls = %d", provider.calls.Load())
		}
		onProgram(nil)

		msg := runBlockingStartupSubmission(t, startupSubmission, provider)
		if _, ok := msg.(tui.AgentDoneMsg); !ok {
			t.Fatalf("startup command returned %T, want tui.AgentDoneMsg", msg)
		}
	}

	if err := RunTUIWithImageWithConfig("describe", "test-model", provider, imagePath, newProjectMapDisabledConfig(), false); err != nil {
		t.Fatalf("RunTUIWithImageWithConfig() error = %v", err)
	}

	if provider.calls.Load() != 1 {
		t.Fatalf("provider calls = %d, want 1", provider.calls.Load())
	}
	stripped := stripANSI(gotInitialContent)
	if !strings.Contains(stripped, "Image loaded:") {
		t.Fatalf("initial content missing image loaded line:\n%s", stripped)
	}
	for _, fragment := range []string{"Sending image:", "mock image response"} {
		if strings.Contains(stripped, fragment) {
			t.Fatalf("initial content should not include pre-TUI image turn output %q:\n%s", fragment, stripped)
		}
	}
}
