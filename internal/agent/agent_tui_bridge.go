package agent

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tui"
	"github.com/susugadx/xelyon-cli/internal/tui/lifecycle"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type tuiProgramBridge struct {
	adapter         *TUIAdapter
	agent           *Agent
	toolResultCh    <-chan tools.ToolResultInfo
	send            func(tea.Msg)
	debugLog        func(string, ...any)
	outgoing        chan tea.Msg
	toolFlush       chan chan struct{}
	toolForwardDone chan struct{}
	promptBridge    *tuiPromptBridge
	closed          atomic.Bool
	droppedEvents   atomic.Int64
}

type tuiBridgeFlushMsg struct {
	done chan struct{}
}

type tuiPromptBridge struct {
	send   func(tea.Msg)
	mu     sync.Mutex
	closed chan struct{}
	once   sync.Once
	nextID atomic.Uint64
}

func (*tuiPromptBridge) isTUIRuntimePrompter() {}

func newTUIPromptBridge(send func(tea.Msg)) *tuiPromptBridge {
	return &tuiPromptBridge{
		send:   send,
		closed: make(chan struct{}),
	}
}

func (b *tuiPromptBridge) Prompt(ctx context.Context, req ui.PromptRequest) (ui.PromptResponse, error) {
	if b == nil || b.send == nil {
		return cancelPromptResponse(req), nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	select {
	case <-b.closed:
		return cancelPromptResponse(req), nil
	default:
	}

	id := b.nextID.Add(1)
	respCh := make(chan ui.PromptResponse, 1)
	b.send(tui.OpenPromptMsg{
		ID:      id,
		Request: req,
		Respond: respCh,
	})

	select {
	case resp := <-respCh:
		return resp, nil
	case <-ctx.Done():
		b.send(tui.CancelPromptMsg{ID: id})
		return cancelPromptResponse(req), nil
	case <-b.closed:
		return cancelPromptResponse(req), nil
	}
}

func (b *tuiPromptBridge) close() {
	if b == nil {
		return
	}
	b.once.Do(func() {
		close(b.closed)
	})
}

func cancelPromptResponse(req ui.PromptRequest) ui.PromptResponse {
	if req.Kind == ui.PromptKindConfirm {
		return ui.PromptResponse{Action: ui.PromptActionNo, Cancelled: true}
	}
	return ui.PromptResponse{Cancelled: true}
}

func newTUIProgramBridge(
	adapter *TUIAdapter,
	agent *Agent,
	toolResultCh <-chan tools.ToolResultInfo,
	send func(tea.Msg),
	debugLog func(string, ...any),
) *tuiProgramBridge {
	if debugLog == nil {
		debugLog = func(string, ...any) {}
	}
	return &tuiProgramBridge{
		adapter:         adapter,
		agent:           agent,
		toolResultCh:    toolResultCh,
		send:            send,
		debugLog:        debugLog,
		outgoing:        make(chan tea.Msg, 4096),
		toolFlush:       make(chan chan struct{}),
		toolForwardDone: make(chan struct{}),
	}
}

func buildTUIToolResult(info tools.ToolResultInfo) tui.ToolResult {
	displayInfo := ui.ToolDisplayInfo{
		ToolName: info.ToolName,
		Args:     info.Args,
		Result:   info.Result,
		Error:    info.Error,
	}
	status := tuiToolStatus(info)
	target := ui.ToolTarget(displayInfo)
	summary := formatTUIToolSummary(status, info.ToolName, target, info.Duration)
	return tui.ToolResult{
		Name:      info.ToolName,
		Summary:   summary,
		Detail:    info.Result,
		Collapsed: defaultToolCollapsed(info.ToolName, info.Result, info.Error),
		Error:     info.Error,
		ID:        info.ID,
		Status:    status,
		Target:    target,
		StartedAt: info.StartedAt,
		Duration:  info.Duration,
	}
}

func tuiToolStatus(info tools.ToolResultInfo) tui.ToolStatus {
	switch info.Status {
	case tools.ToolStatusRunning:
		return tui.ToolStatusRunning
	case tools.ToolStatusError:
		return tui.ToolStatusError
	case tools.ToolStatusOK:
		return tui.ToolStatusOK
	default:
		if info.Error {
			return tui.ToolStatusError
		}
		return tui.ToolStatusOK
	}
}

func formatTUIToolSummary(status tui.ToolStatus, toolName, target string, duration time.Duration) string {
	parts := []string{toolStatusLabel(status), toolName}
	if strings.TrimSpace(target) != "" {
		parts = append(parts, target)
	}
	if status != tui.ToolStatusRunning && duration > 0 {
		parts = append(parts, ui.FormatParallelElapsed(duration))
	}
	return strings.Join(parts, " ")
}

func toolStatusLabel(status tui.ToolStatus) string {
	switch status {
	case tui.ToolStatusRunning:
		return "● running"
	case tui.ToolStatusError:
		return "✕ error"
	default:
		return "✓ ok"
	}
}

func (b *tuiProgramBridge) start() {
	if b == nil || b.adapter == nil {
		return
	}

	go b.forwardOutgoingMessages()
	go b.forwardToolResults()

	b.adapter.sendMsg = b.enqueueCapturedMessage
	b.adapter.sendToolResult = b.enqueueToolResult
	b.adapter.sendReviewProgress = b.enqueueReviewProgress
	b.adapter.setTUIEventFlush(b.flushPendingTUIEvents)
	b.adapter.SetOutputCapture()
	if b.agent != nil && b.send != nil {
		b.promptBridge = newTUIPromptBridge(b.send)
		b.agent.ui().SetPrompter(b.promptBridge)
	}
}

func (b *tuiProgramBridge) forwardOutgoingMessages() {
	for msg := range b.outgoing {
		if flush, ok := msg.(tuiBridgeFlushMsg); ok {
			close(flush.done)
			continue
		}
		if b.send != nil {
			b.send(msg)
		}
	}
}

func (b *tuiProgramBridge) forwardToolResults() {
	defer close(b.toolForwardDone)
	for {
		select {
		case info, ok := <-b.toolResultCh:
			if !ok {
				return
			}
			if !b.enqueueToolResultInfo(info) {
				return
			}
		case done := <-b.toolFlush:
			b.drainToolResults()
			b.flushOutgoingMessages()
			close(done)
		}
	}
}

func (b *tuiProgramBridge) enqueueToolResultInfo(info tools.ToolResultInfo) bool {
	if b.closed.Load() {
		return false
	}
	if b.send == nil {
		return true
	}
	b.enqueueToolResult(tui.AppendToolResultMsg{
		Tool: buildTUIToolResult(info),
	})
	return true
}

func (b *tuiProgramBridge) drainToolResults() {
	for {
		select {
		case info, ok := <-b.toolResultCh:
			if !ok {
				return
			}
			if !b.enqueueToolResultInfo(info) {
				return
			}
		default:
			return
		}
	}
}

func (b *tuiProgramBridge) enqueueCapturedMessage(msg tui.AppendMessageMsg) {
	if b.closed.Load() {
		return
	}
	select {
	case b.outgoing <- msg:
	default:
		b.droppedEvents.Add(1)
	}
}

func (b *tuiProgramBridge) enqueueToolResult(msg tui.AppendToolResultMsg) {
	if b.closed.Load() {
		return
	}
	b.outgoing <- msg
}

func (b *tuiProgramBridge) enqueueReviewProgress(msg tui.ReviewProgressMsg) {
	if b.closed.Load() {
		return
	}
	b.outgoing <- msg
}

func (b *tuiProgramBridge) flushPendingTUIEvents() {
	if b == nil || b.closed.Load() {
		return
	}
	done := make(chan struct{})
	select {
	case b.toolFlush <- done:
	case <-b.toolForwardDone:
		b.flushOutgoingMessages()
		return
	}
	select {
	case <-done:
	case <-b.toolForwardDone:
	}
}

func (b *tuiProgramBridge) flushOutgoingMessages() {
	if b == nil || b.closed.Load() {
		return
	}
	done := make(chan struct{})
	b.outgoing <- tuiBridgeFlushMsg{done: done}
	<-done
}

func (b *tuiProgramBridge) shutdown() {
	if b == nil {
		return
	}
	b.closed.Store(true)
	if b.adapter != nil {
		b.adapter.sendToolResult = nil
		b.adapter.sendReviewProgress = nil
		b.adapter.setTUIEventFlush(nil)
	}
	if b.agent != nil {
		b.agent.tuiToolResultClosed.Store(true)
		if b.promptBridge != nil {
			b.promptBridge.close()
			b.agent.ui().SetPrompter(nil)
		}
	}
	if n := b.droppedEvents.Load(); n > 0 {
		b.debugLog("TUI message channel: %d messages dropped", n)
	}
}

func sendTUIProgramMessage(program *tea.Program, msg tea.Msg) {
	if program != nil {
		program.Send(msg)
	}
}

func registerTUIBridgeOnExit(register func(func()), shutdown func()) {
	if register != nil && shutdown != nil {
		register(shutdown)
	}
}

func bindTUIProgram(adapter *TUIAdapter, ag *Agent, toolResultCh <-chan tools.ToolResultInfo, program *tea.Program) *tuiProgramBridge {
	var send func(tea.Msg)
	if program != nil {
		send = func(msg tea.Msg) {
			sendTUIProgramMessage(program, msg)
		}
	}
	bridge := newTUIProgramBridge(adapter, ag, toolResultCh, send, lifecycle.DebugLog)
	bridge.start()
	registerTUIBridgeOnExit(registerTUIOnExit, bridge.shutdown)
	return bridge
}
