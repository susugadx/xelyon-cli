package tui

import (
	"strings"
	"testing"
	"time"
)

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

func TestModel_AppendNonBlockingToolErrorKeepsAgentActivityWorking(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.beginAgentActivity()

	updated, _, handled := m.handleStreamMessage(AppendToolResultMsg{
		Tool: ToolResult{
			ID:               "review:probe:probe-1:0",
			Name:             "probe host_readonly",
			Target:           "· go test ./internal/tui",
			Status:           ToolStatusError,
			NonBlockingError: true,
			Duration:         120 * time.Millisecond,
		},
	})
	if !handled {
		t.Fatal("AppendToolResultMsg should be handled")
	}
	m = updated

	if m.agentActivity.status != agentActivityStatusWorking {
		t.Fatalf("agentActivity status = %q, want working for non-blocking error", m.agentActivity.status)
	}
	plain := plainRawTranscript(m)
	for _, want := range []string{
		"working",
		"✕ [tool error] probe host_readonly failed · 120ms",
	} {
		if !strings.Contains(plain, want) {
			t.Fatalf("transcript missing %q:\n%s", want, plain)
		}
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
	if plain := plainRawTranscript(m); !strings.Contains(plain, "  ✓ read_file internal/tui/model.go") {
		t.Fatalf("fallback tool block should render, transcript:\n%s", plain)
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
	for _, fragment := range []string{"── agent · blocked ·", "✕ [tool error] read_file failed · 38ms", "! inspect tool output or adjust the request"} {
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
	for _, fragment := range []string{"── agent · blocked ·", "✕ [tool error]", "! inspect tool output or adjust the request"} {
		if strings.Contains(plain, fragment) {
			t.Fatalf("recovered error tool should not leave final blocked marker %q, transcript:\n%s", fragment, plain)
		}
	}
	if m.agentActivity.errorText != "" {
		t.Fatalf("agentActivity.errorText = %q, want empty after recovered turn", m.agentActivity.errorText)
	}
}
