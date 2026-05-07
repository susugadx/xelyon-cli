package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func TestModel_ChatSubmissionAppendsAgentActivityAfterUserMessage(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.textInput.SetValue("hello")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd == nil {
		t.Fatal("chat submission should return send command")
	}
	if !m.agentActivity.active {
		t.Fatal("agent activity should be active after chat submission")
	}
	plain := plainRawTranscript(m)
	userIdx := strings.Index(plain, "── user ·")
	agentIdx := strings.Index(plain, "── agent · working · 00:00 ──")
	if userIdx < 0 || agentIdx < 0 || agentIdx <= userIdx {
		t.Fatalf("agent activity should follow user turn, transcript:\n%s", plain)
	}
}

func TestModel_StartupSubmissionAppendsAgentActivityAfterUserMessage(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	startup := StartupSubmission{
		UserMessage: "describe image",
		Cmd: func() tea.Msg {
			return AgentDoneMsg{}
		},
	}
	m := newModelWithViewport(agent)

	updated, cmd := m.handleStartupSubmissionMsg(startupSubmissionMsg{submission: startup})
	m = updated

	if cmd == nil {
		t.Fatal("startup submission should return startup command")
	}
	if !m.agentActivity.active {
		t.Fatal("agent activity should be active after startup submission")
	}
	plain := plainRawTranscript(m)
	userIdx := strings.Index(plain, "── user ·")
	agentIdx := strings.Index(plain, "── agent · working · 00:00 ──")
	if userIdx < 0 || agentIdx < 0 || agentIdx <= userIdx {
		t.Fatalf("agent activity should follow startup user turn, transcript:\n%s", plain)
	}
}

func TestModel_AgentActivitySpinnerTickReplacesTrackedBlock(t *testing.T) {
	agent := &stubAgent{
		statusLine: "processing",
		processing: true,
		statusSnapshot: StatusSnapshot{
			Provider:   "openai",
			Model:      "gpt-5.4",
			Tokens:     "12.3k",
			Cost:       "~$0.123",
			LegacyLine: "processing",
		},
	}
	m := newModelWithViewport(agent)
	m.beginAgentActivity()
	m.agentActivity.startedAt = time.Now().Add(-65 * time.Second)
	initialLen := len(m.rawLines)
	blockStart := m.agentActivity.block.lineStart

	updated, _ := m.Update(spinner.TickMsg{})
	m = updated.(Model)

	if len(m.rawLines) != initialLen {
		t.Fatalf("spinner tick should replace agent block, rawLines len = %d, want %d", len(m.rawLines), initialLen)
	}
	if m.agentActivity.block.lineStart != blockStart {
		t.Fatalf("agent block start = %d, want %d", m.agentActivity.block.lineStart, blockStart)
	}
	plain := plainRawTranscript(m)
	for _, fragment := range []string{"── agent · working · 01:05 ──", "openai/gpt-5.4", "12.3k tok", "~$0.123"} {
		if !strings.Contains(plain, fragment) {
			t.Fatalf("agent activity missing %q, transcript:\n%s", fragment, plain)
		}
	}
}

func TestModel_AppendToolResultMsgUpsertsToolInsideActiveAgentActivity(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.beginAgentActivity()

	updated, _, handled := m.handleStreamMessage(AppendToolResultMsg{
		Tool: ToolResult{
			ID:     "tool-1",
			Name:   "read_file",
			Target: "internal/tui/model.go",
			Status: ToolStatusRunning,
		},
	})
	if !handled {
		t.Fatal("AppendToolResultMsg should be handled")
	}
	m = updated
	runningLen := len(m.rawLines)

	if len(m.toolBlocks) != 0 {
		t.Fatalf("active agent activity should not append fallback toolBlocks, got %d", len(m.toolBlocks))
	}
	if got := len(m.agentActivity.tools); got != 1 {
		t.Fatalf("agent activity tools = %d, want 1", got)
	}
	if plain := plainRawTranscript(m); !strings.Contains(plain, "● running read_file internal/tui/model.go") {
		t.Fatalf("running tool should render inside agent activity, transcript:\n%s", plain)
	}

	updated, _, handled = m.handleStreamMessage(AppendToolResultMsg{
		Tool: ToolResult{
			ID:       "tool-1",
			Name:     "read_file",
			Target:   "internal/tui/model.go",
			Status:   ToolStatusOK,
			Duration: 38 * time.Millisecond,
		},
	})
	if !handled {
		t.Fatal("AppendToolResultMsg completion should be handled")
	}
	m = updated

	if len(m.rawLines) != runningLen {
		t.Fatalf("tool completion should update existing activity row, rawLines len = %d, want %d", len(m.rawLines), runningLen)
	}
	plain := plainRawTranscript(m)
	if !strings.Contains(plain, "✓ read_file internal/tui/model.go · 38ms") || strings.Contains(plain, "● running read_file") {
		t.Fatalf("tool completion should replace running row, transcript:\n%s", plain)
	}
}

func TestModel_AppendToolResultMsgWithoutActiveAgentActivityUsesFallbackToolBlock(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	updated, _, handled := m.handleStreamMessage(AppendToolResultMsg{
		Tool: ToolResult{Name: "read_file", Summary: "✓ read_file internal/tui/model.go", Detail: "ok", Collapsed: true},
	})
	if !handled {
		t.Fatal("AppendToolResultMsg should be handled")
	}
	m = updated

	if len(m.toolBlocks) != 1 {
		t.Fatalf("fallback toolBlocks len = %d, want 1", len(m.toolBlocks))
	}
	if plain := plainRawTranscript(m); !strings.Contains(plain, "▶ ✓ read_file internal/tui/model.go") {
		t.Fatalf("fallback tool block should render, transcript:\n%s", plain)
	}
}

func TestModel_AgentCommandSubmissionKeepsToolUpdatesInActivityUntilDone(t *testing.T) {
	agent := &stubAgent{
		statusLine:      "ready",
		handledCommands: map[string]bool{"/thinking high": true},
	}
	m := newModelWithViewport(agent)

	updated, cmd := m.handleCommandSubmission(composerSubmission{
		kind:         composerSubmissionCommand,
		commandInput: "/thinking high",
		payload:      "/thinking high",
	})
	m = updated.(Model)

	if cmd == nil {
		t.Fatal("handled agent command should return completion command")
	}
	if !m.agentActivity.active {
		t.Fatal("agent activity should stay active until completion command is handled")
	}
	plain := plainRawTranscript(m)
	userIdx := strings.Index(plain, "── user ·")
	agentIdx := strings.Index(plain, "── agent · working")
	if userIdx < 0 || agentIdx < 0 || agentIdx <= userIdx {
		t.Fatalf("handled command activity should follow command user turn, transcript:\n%s", plain)
	}

	streamUpdated, _, handled := m.handleStreamMessage(AppendToolResultMsg{
		Tool: ToolResult{
			ID:       "tool-1",
			Name:     "read_file",
			Target:   "a.go",
			Status:   ToolStatusOK,
			Duration: 38 * time.Millisecond,
		},
	})
	if !handled {
		t.Fatal("AppendToolResultMsg should be handled")
	}
	m = streamUpdated
	if len(m.toolBlocks) != 0 {
		t.Fatalf("active command activity should not append fallback toolBlocks, got %d", len(m.toolBlocks))
	}
	if plain := plainRawTranscript(m); !strings.Contains(plain, "✓ read_file a.go · 38ms") {
		t.Fatalf("command tool result should render inside agent activity, transcript:\n%s", plain)
	}

	done := cmd()
	if _, ok := done.(AgentDoneMsg); !ok {
		t.Fatalf("completion command returned %T, want AgentDoneMsg", done)
	}
	updatedModel, _ := m.Update(done)
	m = updatedModel.(Model)

	if m.agentActivity.active {
		t.Fatal("agent activity should close after completion command")
	}
	if plain := plainRawTranscript(m); !strings.Contains(plain, "│ ✓ 1 tools") {
		t.Fatalf("done command activity should compact tool summary, transcript:\n%s", plain)
	}
}

func TestModel_ChatSubmissionWhileActivityActiveDoesNotOverwriteTrackedActivity(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.textInput.SetValue("first")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("first chat submission should return send command")
	}
	firstBlock := m.agentActivity.block

	m.textInput.SetValue("second")
	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("second chat submission while active should not return send command, got %v", cmd)
	}
	if m.agentActivity.block != firstBlock {
		t.Fatalf("agent activity block changed from %#v to %#v", firstBlock, m.agentActivity.block)
	}
	if got := len(m.messages); got != 1 {
		t.Fatalf("messages len = %d, want 1", got)
	}
	if got := m.textInput.Value(); got != "second" {
		t.Fatalf("textInput after rejected submission = %q, want second", got)
	}
	if plain := plainRawTranscript(m); strings.Count(plain, "── agent · working") != 1 {
		t.Fatalf("rejected second submission should not append another activity, transcript:\n%s", plain)
	}
	if m.transientStatus != agentTurnBusyStatus {
		t.Fatalf("transientStatus = %q, want %q", m.transientStatus, agentTurnBusyStatus)
	}
}

func TestModel_AgentCommandSubmissionWhileActivityActiveHandlesWithoutOverwritingTrackedActivity(t *testing.T) {
	agent := &stubAgent{
		statusLine:      "ready",
		handledCommands: map[string]bool{"/thinking high": true},
	}
	m := newModelWithViewport(agent)
	m.beginAgentActivity()
	firstBlock := m.agentActivity.block

	updated, cmd := m.handleCommandSubmission(composerSubmission{
		kind:         composerSubmissionCommand,
		commandInput: "/thinking high",
		payload:      "/thinking high",
	})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("agent command while active should not return completion command, got %v", cmd)
	}
	if got := len(agent.handledInputs); got != 1 {
		t.Fatalf("handledInputs len = %d, want 1", got)
	}
	if m.agentActivity.block != firstBlock {
		t.Fatalf("agent activity block changed from %#v to %#v", firstBlock, m.agentActivity.block)
	}
	if plain := plainRawTranscript(m); strings.Count(plain, "── agent · working") != 1 {
		t.Fatalf("handled command should not append another activity, transcript:\n%s", plain)
	}
	if m.transientStatus == agentTurnBusyStatus {
		t.Fatalf("handled command should not show busy transient, got %q", m.transientStatus)
	}
}

func TestModel_AgentCommandFallbackUsesOnlyChatActivity(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	updated, cmd := m.handleCommandSubmission(composerSubmission{
		kind:         composerSubmissionCommand,
		commandInput: "/unknown",
		payload:      "/unknown",
	})
	m = updated.(Model)

	if cmd == nil {
		t.Fatal("unhandled agent command should fall back to chat command")
	}
	plain := plainRawTranscript(m)
	if got := strings.Count(plain, "── agent · working"); got != 1 {
		t.Fatalf("fallback should keep only chat activity, got %d activity blocks, transcript:\n%s", got, plain)
	}
}

func TestModel_AgentDoneCompactsActivityAndAssistantContinuesAsSeparateTurn(t *testing.T) {
	agent := &stubAgent{
		statusLine: "ready",
		statusSnapshot: StatusSnapshot{
			Provider:   "openai",
			Model:      "gpt-5.4",
			Tokens:     "12",
			Cost:       "~$0.01",
			LegacyLine: "ready",
		},
	}
	m := newModelWithViewport(agent)
	m.beginAgentActivity()
	m.upsertAgentActivityTool(ToolResult{
		ID:       "tool-1",
		Name:     "read_file",
		Target:   "a.go",
		Status:   ToolStatusOK,
		Duration: 38 * time.Millisecond,
	})

	updated, _, handled := m.handleStreamMessage(AgentDoneMsg{})
	if !handled {
		t.Fatal("AgentDoneMsg should be handled")
	}
	m = updated

	plain := plainRawTranscript(m)
	for _, fragment := range []string{"── agent · done ·", "│ ✓ 1 tools · 12 tok · ~$0.01"} {
		if !strings.Contains(plain, fragment) {
			t.Fatalf("done activity missing %q, transcript:\n%s", fragment, plain)
		}
	}
	if strings.Contains(plain, "read_file a.go") {
		t.Fatalf("done activity should compact tool rows, transcript:\n%s", plain)
	}

	updated, _, handled = m.handleStreamMessage(AppendMessageMsg{
		Message: ChatMessage{Role: "assistant", Content: "done"},
	})
	if !handled {
		t.Fatal("AppendMessageMsg should be handled")
	}
	m = updated

	plain = plainRawTranscript(m)
	doneIdx := strings.Index(plain, "── agent · done ·")
	assistantIdx := strings.Index(plain, "── assistant ·")
	if doneIdx < 0 || assistantIdx < 0 || assistantIdx <= doneIdx {
		t.Fatalf("assistant turn should follow compact agent activity, transcript:\n%s", plain)
	}
}

func TestModel_AgentDoneWithErrorBlocksActivity(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.beginAgentActivity()

	updated, _, handled := m.handleStreamMessage(AgentDoneMsg{Error: errors.New("network down")})
	if !handled {
		t.Fatal("AgentDoneMsg should be handled")
	}
	m = updated

	plain := plainRawTranscript(m)
	for _, fragment := range []string{"── agent · blocked ·", "✕ network down", "! user action may be needed"} {
		if !strings.Contains(plain, fragment) {
			t.Fatalf("blocked activity missing %q, transcript:\n%s", fragment, plain)
		}
	}
}

func TestModel_ChatSubmissionErrorCompletesActivityAsBlocked(t *testing.T) {
	agent := &stubAgent{statusLine: "ready", chatErr: errors.New("provider down")}
	m := newModelWithViewport(agent)
	m.textInput.SetValue("hello")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("chat submission should return send command")
	}

	msg := cmd()
	done, ok := msg.(AgentDoneMsg)
	if !ok {
		t.Fatalf("send command returned %T, want AgentDoneMsg", msg)
	}
	if done.Error == nil || !strings.Contains(done.Error.Error(), "provider down") {
		t.Fatalf("AgentDoneMsg.Error = %v, want provider down", done.Error)
	}

	updatedModel, _ := m.Update(done)
	m = updatedModel.(Model)

	if m.agentActivity.active {
		t.Fatal("agent activity should close after failed chat submission")
	}
	plain := plainRawTranscript(m)
	for _, fragment := range []string{"── agent · blocked ·", "✕ provider down", "! user action may be needed"} {
		if !strings.Contains(plain, fragment) {
			t.Fatalf("failed chat activity missing %q, transcript:\n%s", fragment, plain)
		}
	}
}

func TestModel_RecoveredErrorToolDoesNotBlockActivityOnDone(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.beginAgentActivity()
	m.upsertAgentActivityTool(ToolResult{
		ID:       "tool-1",
		Name:     "read_file",
		Detail:   "permission denied\nmore detail",
		Status:   ToolStatusError,
		Error:    true,
		Duration: 38 * time.Millisecond,
	})

	activePlain := plainRawTranscript(m)
	for _, fragment := range []string{"── agent · blocked ·", "✕ read_file failed · 38ms", "! user action may be needed"} {
		if !strings.Contains(activePlain, fragment) {
			t.Fatalf("active error tool should render blocked, missing %q, transcript:\n%s", fragment, activePlain)
		}
	}

	updated, _, handled := m.handleStreamMessage(AgentDoneMsg{})
	if !handled {
		t.Fatal("AgentDoneMsg should be handled")
	}
	m = updated

	plain := plainRawTranscript(m)
	for _, fragment := range []string{"── agent · done ·", "│ ✓ 1 tools"} {
		if !strings.Contains(plain, fragment) {
			t.Fatalf("recovered error tool should finish done, missing %q, transcript:\n%s", fragment, plain)
		}
	}
	for _, fragment := range []string{"── agent · blocked ·", "✕ permission denied", "! user action may be needed"} {
		if strings.Contains(plain, fragment) {
			t.Fatalf("recovered error tool should not leave final blocked marker %q, transcript:\n%s", fragment, plain)
		}
	}
	if m.agentActivity.errorText != "" {
		t.Fatalf("agentActivity.errorText = %q, want empty after recovered turn", m.agentActivity.errorText)
	}
}

func TestModel_StartupSubmissionResultWithoutDoneBlocksActivity(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	startup := StartupSubmission{
		UserMessage: "describe image",
		Cmd: func() tea.Msg {
			return AppendMessageMsg{
				Message: ChatMessage{Role: ChatRoleAssistantChunk, Content: "partial output"},
			}
		},
	}
	m := newModelWithViewport(agent)

	updated, cmd := m.handleStartupSubmissionMsg(startupSubmissionMsg{submission: startup})
	m = updated
	if cmd == nil {
		t.Fatal("startup submission should return wrapped command")
	}

	result := cmd()
	updatedModel, _ := m.Update(result)
	m = updatedModel.(Model)

	if m.agentActivity.active {
		t.Fatal("startup activity should close when command returns without AgentDoneMsg")
	}
	plain := plainRawTranscript(m)
	for _, fragment := range []string{"partial output", "── agent · blocked ·", "startup command returned tui.AppendMessageMsg without completion"} {
		if !strings.Contains(plain, fragment) {
			t.Fatalf("startup missing-done activity missing %q, transcript:\n%s", fragment, plain)
		}
	}
}

func TestModel_StartupSubmissionPanicBlocksActivity(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	startup := StartupSubmission{
		UserMessage: "describe image",
		Cmd: func() tea.Msg {
			panic("boom")
		},
	}
	m := newModelWithViewport(agent)

	updated, cmd := m.handleStartupSubmissionMsg(startupSubmissionMsg{submission: startup})
	m = updated
	if cmd == nil {
		t.Fatal("startup submission should return wrapped command")
	}

	result := cmd()
	updatedModel, _ := m.Update(result)
	m = updatedModel.(Model)

	if m.agentActivity.active {
		t.Fatal("startup activity should close when command panics")
	}
	if plain := plainRawTranscript(m); !strings.Contains(plain, "startup command failed: boom") {
		t.Fatalf("startup panic should block activity with error, transcript:\n%s", plain)
	}
}

func TestModel_AgentActivityLiveUpdatesRespectViewportFollowState(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	for i := 0; i < 40; i++ {
		m.appendContentLines("line")
	}
	m.beginAgentActivity()
	if !m.vp.atBottom() {
		t.Fatal("viewport should start at bottom")
	}

	updated, _, _ := m.handleStreamMessage(AppendToolResultMsg{
		Tool: ToolResult{ID: "tool-1", Name: "read_file", Target: "a.go", Status: ToolStatusRunning},
	})
	m = updated
	if !m.vp.atBottom() {
		t.Fatal("viewport should follow live activity updates while at bottom")
	}
	if m.newOutput {
		t.Fatal("newOutput should stay false while following bottom")
	}

	m.vp.gotoTop()
	savedOffset := m.vp.yOffset
	m.newOutput = false
	updated, _, _ = m.handleStreamMessage(AppendToolResultMsg{
		Tool: ToolResult{ID: "tool-1", Name: "read_file", Target: "a.go", Status: ToolStatusOK, Duration: 38 * time.Millisecond},
	})
	m = updated
	if m.vp.yOffset != savedOffset {
		t.Fatalf("viewport yOffset = %d, want %d while scrolled up", m.vp.yOffset, savedOffset)
	}
	if !m.newOutput {
		t.Fatal("newOutput should be set for live activity updates while scrolled up")
	}
}

func plainRawTranscript(m Model) string {
	return stripANSI(strings.Join(m.rawLines, "\n"))
}
