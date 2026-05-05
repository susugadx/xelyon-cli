package agent

import (
	"context"
	"sync"
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tui"
	"github.com/susugadx/xelyon-cli/internal/tui/lifecycle"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type tuiProgramBridge struct {
	adapter       *TUIAdapter
	agent         *Agent
	toolResultCh  <-chan tools.ToolResultInfo
	send          func(tea.Msg)
	debugLog      func(string, ...any)
	messageQueue  chan tui.AppendMessageMsg
	promptBridge  *tuiPromptBridge
	closed        atomic.Bool
	droppedEvents atomic.Int64
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
		adapter:      adapter,
		agent:        agent,
		toolResultCh: toolResultCh,
		send:         send,
		debugLog:     debugLog,
		messageQueue: make(chan tui.AppendMessageMsg, 4096),
	}
}

func (b *tuiProgramBridge) start() {
	if b == nil || b.adapter == nil {
		return
	}

	go func() {
		for msg := range b.messageQueue {
			if b.send != nil {
				b.send(msg)
			}
		}
	}()

	go func() {
		for info := range b.toolResultCh {
			if b.closed.Load() {
				return
			}
			if b.send == nil {
				continue
			}
			summary := ui.FormatToolLine(ui.ToolDisplayInfo{
				ToolName: info.ToolName,
				Args:     info.Args,
				Result:   info.Result,
				Error:    info.Error,
			})
			b.send(tui.AppendToolResultMsg{
				Tool: tui.ToolResult{
					Name:      info.ToolName,
					Summary:   summary,
					Detail:    info.Result,
					Collapsed: defaultToolCollapsed(info.ToolName, info.Result, info.Error),
					Error:     info.Error,
				},
			})
		}
	}()

	b.adapter.sendMsg = func(msg tui.AppendMessageMsg) {
		if b.closed.Load() {
			return
		}
		select {
		case b.messageQueue <- msg:
		default:
			b.droppedEvents.Add(1)
		}
	}
	b.adapter.SetOutputCapture()
	if b.agent != nil && b.send != nil {
		b.promptBridge = newTUIPromptBridge(b.send)
		b.agent.ui().SetPrompter(b.promptBridge)
	}
}

func (b *tuiProgramBridge) shutdown() {
	if b == nil {
		return
	}
	b.closed.Store(true)
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
