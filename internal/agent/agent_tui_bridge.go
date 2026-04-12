package agent

import (
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tui"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type tuiProgramBridge struct {
	adapter       *TUIAdapter
	agent         *Agent
	toolResultCh  <-chan tools.ToolResultInfo
	send          func(tea.Msg)
	debugLog      func(string, ...any)
	messageQueue  chan tui.AppendMessageMsg
	closed        atomic.Bool
	droppedEvents atomic.Int64
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
}

func (b *tuiProgramBridge) shutdown() {
	if b == nil {
		return
	}
	b.closed.Store(true)
	if b.agent != nil {
		b.agent.tuiToolResultClosed.Store(true)
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
	bridge := newTUIProgramBridge(adapter, ag, toolResultCh, func(msg tea.Msg) {
		sendTUIProgramMessage(program, msg)
	}, tui.DebugLog)
	bridge.start()
	registerTUIBridgeOnExit(registerTUIOnExit, bridge.shutdown)
	return bridge
}
