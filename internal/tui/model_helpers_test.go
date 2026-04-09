package tui

import (
	"fmt"
	"strings"
	"sync"

	"github.com/susugadx/xelyon-cli/internal/config"
)

type stubAgent struct {
	mu                sync.RWMutex
	processing        bool
	cancelCalls       int
	cleanupCalls      int
	copyCalls         int
	copyTexts         []string
	chatInputs        []string
	handledInputs     []string
	handledCommands   map[string]bool
	statusLine        string
	saveStatusLine    string
	providerName      string
	providerConfigKey string
	saveErr           error          // non-nil にすると SaveAndSyncConfig が失敗する
	lastSavedConfig   *config.Config // SaveAndSyncConfig で受け取った最後の Config
}

func (s *stubAgent) Chat(input string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.chatInputs = append(s.chatInputs, input)
}
func (s *stubAgent) HandleCommand(cmd string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handledInputs = append(s.handledInputs, cmd)
	return s.handledCommands[cmd]
}
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
func (s *stubAgent) LoadConfigForEdit() (*config.Config, error) {
	return config.DefaultConfig(), nil
}
func (s *stubAgent) SaveAndSyncConfig(cfg *config.Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastSavedConfig = config.CloneConfig(cfg)
	if s.saveErr == nil && s.saveStatusLine != "" {
		s.statusLine = s.saveStatusLine
	}
	return s.saveErr
}
func (s *stubAgent) GetProviderName() string {
	if s.providerName != "" {
		return s.providerName
	}
	return "deepseek"
}
func (s *stubAgent) GetProviderConfigKey() string {
	if s.providerConfigKey != "" {
		return s.providerConfigKey
	}
	return s.GetProviderName()
}
func (s *stubAgent) ResolveAlias(cmd string) string {
	// テスト用 alias
	switch cmd {
	case "/c":
		return "/config"
	case "/q":
		return "/quit"
	case "/cp":
		return "/copy"
	}
	return cmd
}

func (s *stubAgent) lastChatInput() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.chatInputs) == 0 {
		return ""
	}
	return s.chatInputs[len(s.chatInputs)-1]
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
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
}

func newModelWithViewport(agent AgentInterface) Model {
	m := NewModel(agent, "")
	m.ready = true
	m.width = 80
	m.height = 30
	m.vp = lightViewport{width: 80, height: m.height - m.footerHeight()}
	m.padLineCache = strings.Repeat(" ", 80)
	return m
}
