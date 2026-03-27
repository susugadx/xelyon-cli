package tui

import (
	"fmt"
	"strings"
	"sync"
)

type stubAgent struct {
	mu           sync.RWMutex
	processing   bool
	cancelCalls  int
	cleanupCalls int
	copyCalls    int
	copyTexts    []string
	statusLine   string
}

func (s *stubAgent) Chat(input string)             {}
func (s *stubAgent) HandleCommand(cmd string) bool { return false }
func (s *stubAgent) GetStatusLine() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.statusLine
}
func (s *stubAgent) Cancel() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cancelCalls++
}
func (s *stubAgent) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupCalls++
}
func (s *stubAgent) IsProcessing() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.processing
}
func (s *stubAgent) CopyLastOutput() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.copyCalls++
	return "Copied 5 lines", nil
}
func (s *stubAgent) CopyText(text string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.copyCalls++
	s.copyTexts = append(s.copyTexts, text)
	return nil
}

func (s *stubAgent) setProcessing(processing bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.processing = processing
}

func (s *stubAgent) setStatusLine(status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statusLine = status
}

func setModelRawLines(m *Model, count int) {
	lines := make([]string, count)
	for i := 0; i < count; i++ {
		lines[i] = fmt.Sprintf("line%d", i)
	}
	m.rawLines = append([]string(nil), lines...)
	m.rebuildRenderedLines()
	m.vp.setLines(m.renderedLines)
}

func newModelWithViewport(agent AgentInterface) Model {
	m := NewModel(agent, "")
	m.ready = true
	m.width = 80
	m.height = 30
	m.vp = lightViewport{width: 80, height: 26}
	m.padLineCache = strings.Repeat(" ", 80)
	return m
}
