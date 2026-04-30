package agent

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tui"
)

type dummyTeaModel struct{}

func (dummyTeaModel) Init() tea.Cmd                           { return nil }
func (dummyTeaModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return dummyTeaModel{}, tea.Quit }
func (dummyTeaModel) View() string                            { return "" }

func waitForTUIMessageCount(t *testing.T, msgs *[]tea.Msg, mu *sync.Mutex, want int) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		got := len(*msgs)
		mu.Unlock()
		if got >= want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	mu.Lock()
	got := len(*msgs)
	mu.Unlock()
	t.Fatalf("message count = %d, want at least %d", got, want)
}

func TestTUIProgramBridge_StartCapturesAssistantAndToolResults(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	agent := newChatRequestTestAgent(t, &scriptedChatProvider{name: "openai", functionCalling: true}, &out)
	adapter := NewTUIAdapter(agent, nil)
	toolResultCh := make(chan tools.ToolResultInfo, 1)

	var (
		mu   sync.Mutex
		msgs []tea.Msg
	)
	bridge := newTUIProgramBridge(adapter, agent, toolResultCh, func(msg tea.Msg) {
		mu.Lock()
		msgs = append(msgs, msg)
		mu.Unlock()
	}, nil)

	bridge.start()

	fmt.Fprintln(agent.output(), "assistant line")
	toolResultCh <- tools.ToolResultInfo{ToolName: "bash", Result: "ok", Error: false}
	close(toolResultCh)

	waitForTUIMessageCount(t, &msgs, &mu, 2)

	mu.Lock()
	defer mu.Unlock()

	var (
		appendMsg tui.AppendMessageMsg
		toolMsg   tui.AppendToolResultMsg
		foundText bool
		foundTool bool
	)
	for _, msg := range msgs {
		switch typed := msg.(type) {
		case tui.AppendMessageMsg:
			appendMsg = typed
			foundText = true
		case tui.AppendToolResultMsg:
			toolMsg = typed
			foundTool = true
		}
	}
	if !foundText {
		t.Fatalf("expected captured assistant message, got %#v", msgs)
	}
	if !strings.Contains(appendMsg.Message.Content, "assistant line") {
		t.Fatalf("assistant message = %#v, want captured output", appendMsg.Message)
	}
	if !foundTool {
		t.Fatalf("expected tool result message, got %#v", msgs)
	}
	if toolMsg.Tool.Name != "bash" {
		t.Fatalf("tool name = %q, want %q", toolMsg.Tool.Name, "bash")
	}
	if toolMsg.Tool.Detail != "ok" {
		t.Fatalf("tool detail = %q, want %q", toolMsg.Tool.Detail, "ok")
	}
	if !toolMsg.Tool.Collapsed {
		t.Fatalf("tool collapsed = false, want true for bash success")
	}

	bridge.shutdown()
	if !agent.tuiToolResultClosed.Load() {
		t.Fatal("expected bridge shutdown to mark tool result channel closed")
	}
}

func TestTUIProgramBridge_ShutdownLogsDroppedEvents(t *testing.T) {
	disableColors(t)

	agent := newChatRequestTestAgent(t, &scriptedChatProvider{name: "openai", functionCalling: true}, &bytes.Buffer{})
	bridge := newTUIProgramBridge(NewTUIAdapter(agent, nil), agent, make(chan tools.ToolResultInfo), nil, nil)

	var logged string
	bridge.debugLog = func(format string, args ...any) {
		logged = fmt.Sprintf(format, args...)
	}
	bridge.droppedEvents.Add(2)

	bridge.shutdown()

	if !strings.Contains(logged, "2 messages dropped") {
		t.Fatalf("shutdown log = %q, want dropped-event summary", logged)
	}
}

func TestTUIProgramBridge_SendMsgDropsWhenQueueIsFullAndIgnoresAfterShutdown(t *testing.T) {
	disableColors(t)

	agent := newChatRequestTestAgent(t, &scriptedChatProvider{name: "openai", functionCalling: true}, &bytes.Buffer{})
	adapter := NewTUIAdapter(agent, nil)
	toolResultCh := make(chan tools.ToolResultInfo)

	sendStarted := make(chan struct{}, 1)
	releaseSend := make(chan struct{})
	bridge := newTUIProgramBridge(adapter, agent, toolResultCh, func(tea.Msg) {
		select {
		case sendStarted <- struct{}{}:
		default:
		}
		<-releaseSend
	}, nil)
	bridge.messageQueue = make(chan tui.AppendMessageMsg, 1)
	bridge.start()

	adapter.sendMsg(tui.AppendMessageMsg{Message: tui.ChatMessage{Role: "assistant", Content: "first"}})
	<-sendStarted
	adapter.sendMsg(tui.AppendMessageMsg{Message: tui.ChatMessage{Role: "assistant", Content: "second"}})
	adapter.sendMsg(tui.AppendMessageMsg{Message: tui.ChatMessage{Role: "assistant", Content: "third"}})

	if got := bridge.droppedEvents.Load(); got != 1 {
		t.Fatalf("droppedEvents = %d, want 1", got)
	}

	bridge.shutdown()
	adapter.sendMsg(tui.AppendMessageMsg{Message: tui.ChatMessage{Role: "assistant", Content: "ignored"}})
	if got := bridge.droppedEvents.Load(); got != 1 {
		t.Fatalf("droppedEvents after shutdown = %d, want 1", got)
	}

	close(toolResultCh)
	close(releaseSend)
	close(bridge.messageQueue)
}

func TestTUIProgramBridge_StartHandlesNilReceiverAndNilSend(t *testing.T) {
	var nilBridge *tuiProgramBridge
	nilBridge.start()
	nilBridge.shutdown()

	agent := newChatRequestTestAgent(t, &scriptedChatProvider{name: "openai", functionCalling: true}, &bytes.Buffer{})
	toolResultCh := make(chan tools.ToolResultInfo, 1)
	noAdapterBridge := newTUIProgramBridge(nil, agent, toolResultCh, nil, nil)
	noAdapterBridge.start()
	bridge := newTUIProgramBridge(NewTUIAdapter(agent, nil), agent, toolResultCh, nil, nil)
	bridge.start()
	bridge.adapter.sendMsg(tui.AppendMessageMsg{Message: tui.ChatMessage{Role: "assistant", Content: "ignored"}})

	toolResultCh <- tools.ToolResultInfo{ToolName: "bash", Result: "ok", Error: false}
	close(toolResultCh)
	close(bridge.messageQueue)
	bridge.shutdown()
}

func TestTUIProgramBridge_StartStopsForwardingWhenClosed(t *testing.T) {
	disableColors(t)

	agent := newChatRequestTestAgent(t, &scriptedChatProvider{name: "openai", functionCalling: true}, &bytes.Buffer{})
	toolResultCh := make(chan tools.ToolResultInfo, 1)

	var sendCount atomic.Int32
	bridge := newTUIProgramBridge(NewTUIAdapter(agent, nil), agent, toolResultCh, func(tea.Msg) {
		sendCount.Add(1)
	}, nil)
	bridge.start()
	bridge.closed.Store(true)

	toolResultCh <- tools.ToolResultInfo{ToolName: "bash", Result: "ok", Error: false}
	close(toolResultCh)
	time.Sleep(50 * time.Millisecond)

	if sendCount.Load() != 0 {
		t.Fatalf("sendCount = %d, want 0", sendCount.Load())
	}
}

func TestSendTUIProgramMessage_AllowsNilAndProgramInstances(t *testing.T) {
	sendTUIProgramMessage(nil, tui.AppendMessageMsg{})

	program := tea.NewProgram(dummyTeaModel{}, tea.WithInput(nil), tea.WithOutput(io.Discard))
	done := make(chan error, 1)
	go func() {
		_, err := program.Run()
		done <- err
	}()
	time.Sleep(50 * time.Millisecond)
	sendTUIProgramMessage(program, tui.AppendMessageMsg{
		Message: tui.ChatMessage{Role: "assistant", Content: "hello"},
	})

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("program.Run() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for program.Run() to exit")
	}
}

func TestRegisterTUIBridgeOnExit_RegistersWhenBothFunctionsExist(t *testing.T) {
	var called atomic.Int32
	var registered func()

	registerTUIBridgeOnExit(func(fn func()) {
		registered = fn
	}, func() {
		called.Add(1)
	})

	if registered == nil {
		t.Fatal("expected shutdown callback to be registered")
	}
	registered()
	if called.Load() != 1 {
		t.Fatalf("called = %d, want 1", called.Load())
	}

	registerTUIBridgeOnExit(nil, func() {})
	registerTUIBridgeOnExit(func(func()) {}, nil)
}

func TestBindTUIProgram_ReturnsBridgeAndRegistersShutdown(t *testing.T) {
	disableColors(t)

	originalRegister := registerTUIOnExit
	defer func() { registerTUIOnExit = originalRegister }()

	var registered func()
	registerTUIOnExit = func(fn func()) {
		registered = fn
	}

	agent := newChatRequestTestAgent(t, &scriptedChatProvider{name: "openai", functionCalling: true}, &bytes.Buffer{})
	adapter := NewTUIAdapter(agent, nil)
	toolResultCh := make(chan tools.ToolResultInfo, 1)

	bridge := bindTUIProgram(adapter, agent, toolResultCh, nil)
	if bridge == nil {
		t.Fatal("expected bindTUIProgram() to return bridge")
	}
	if registered == nil {
		t.Fatal("expected bindTUIProgram() to register shutdown callback")
	}
	if adapter.sendMsg == nil {
		t.Fatal("expected bindTUIProgram() to configure adapter.sendMsg")
	}

	adapter.sendMsg(tui.AppendMessageMsg{
		Message: tui.ChatMessage{Role: "assistant", Content: "hello"},
	})
	time.Sleep(20 * time.Millisecond)

	registered()
	close(toolResultCh)
	close(bridge.messageQueue)

	if !agent.tuiToolResultClosed.Load() {
		t.Fatal("expected registered shutdown callback to mark tool result channel closed")
	}
}
