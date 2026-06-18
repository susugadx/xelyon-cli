package agent

import (
	"context"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	promptnormal "github.com/susugadx/xelyon-cli/internal/prompt/normal"
)

func TestCompressHistory_DetachesResponseContextOnSuccess(t *testing.T) {
	provider := &compressionTestProvider{name: "openai", summary: "summary"}
	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", config.DefaultConfig())
	agent.History = []api.Message{
		{Role: "user", Content: "old"},
		{Role: "assistant", Content: "older"},
		{Role: "user", Content: "latest"},
	}
	setCompressionTestResponseContext(agent, provider, "resp_old")

	if err := agent.CompressHistory(1); err != nil {
		t.Fatalf("CompressHistory() error = %v", err)
	}

	if provider.capturedChatResponseID != "" {
		t.Fatalf("summary request response ID = %q, want empty", provider.capturedChatResponseID)
	}
	assertCompressionTestResponseContextCleared(t, agent, provider)
}

func TestCompressHistory_PersistsCompressedHistoryBeforeClearingResponseContext(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	provider := &compressionTestProvider{name: "openai", summary: "persisted summary"}
	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", config.DefaultConfig())
	agent.History = []api.Message{
		{Role: "user", Content: "old full context"},
		{Role: "assistant", Content: "older full answer"},
		{Role: "user", Content: "latest question"},
	}
	for _, msg := range agent.History {
		agent.session.AddMessageFromAPI(msg, agent.CurrentModel)
	}
	setCompressionTestResponseContext(agent, provider, "resp_old")
	agent.persistSession()

	if err := agent.CompressHistory(1); err != nil {
		t.Fatalf("CompressHistory() error = %v", err)
	}

	loaded, err := agent.storage.Load(agent.session.ID)
	if err != nil {
		t.Fatalf("storage.Load() error = %v", err)
	}
	loadedMessages := loaded.ToAPIMessages()
	if len(loadedMessages) != 2 {
		t.Fatalf("len(loaded.ToAPIMessages()) = %d, want 2", len(loadedMessages))
	}
	if loadedMessages[0].Role != "assistant" || !strings.Contains(loadedMessages[0].Content, "persisted summary") {
		t.Fatalf("loaded first message = %#v, want assistant continuation data", loadedMessages[0])
	}
	if strings.Contains(loadedMessages[0].Content, "old full context") {
		t.Fatalf("loaded summary contains original full context: %q", loadedMessages[0].Content)
	}
	if loadedMessages[1].Content != "latest question" {
		t.Fatalf("loaded latest message = %q, want latest question", loadedMessages[1].Content)
	}
	if loaded.ResponseID != "" {
		t.Fatalf("loaded.ResponseID = %q, want empty", loaded.ResponseID)
	}
}

func TestCompressHistory_PersistsSessionHistoryWithoutNormalModePrompt(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	provider := &compressionTestProvider{name: "openai", summary: "persisted summary"}
	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", config.DefaultConfig())
	rawOld := "old full context"
	rawLatest := "latest question"
	agent.History = []api.Message{
		{Role: "user", Content: rawOld + promptnormal.NormalModePrompt},
		{Role: "assistant", Content: "older full answer"},
		{Role: "user", Content: rawLatest + promptnormal.NormalModePrompt},
	}
	agent.session.AddMessage("user", rawOld, agent.CurrentModel)
	agent.session.AddMessage("assistant", "older full answer", agent.CurrentModel)
	agent.session.AddMessage("user", rawLatest, agent.CurrentModel)
	agent.persistSession()

	if err := agent.CompressHistory(1); err != nil {
		t.Fatalf("CompressHistory() error = %v", err)
	}

	if len(provider.capturedChatHistory) != 1 {
		t.Fatalf("len(capturedChatHistory) = %d, want 1", len(provider.capturedChatHistory))
	}
	if strings.Contains(provider.capturedChatHistory[0].Content, promptnormal.NormalModePrompt) {
		t.Fatalf("summary prompt contains NormalModePrompt: %q", provider.capturedChatHistory[0].Content)
	}
	if len(agent.History) != 2 || agent.History[1].Content != rawLatest+promptnormal.NormalModePrompt {
		t.Fatalf("runtime History = %#v, want latest runtime prompt retained", agent.History)
	}

	loaded, err := agent.storage.Load(agent.session.ID)
	if err != nil {
		t.Fatalf("storage.Load() error = %v", err)
	}
	loadedMessages := loaded.ToAPIMessages()
	if len(loadedMessages) != 2 {
		t.Fatalf("len(loaded.ToAPIMessages()) = %d, want 2", len(loadedMessages))
	}
	if strings.Contains(loadedMessages[0].Content, promptnormal.NormalModePrompt) {
		t.Fatalf("loaded summary contains NormalModePrompt: %q", loadedMessages[0].Content)
	}
	if loadedMessages[1].Content != rawLatest {
		t.Fatalf("loaded latest message = %q, want %q", loadedMessages[1].Content, rawLatest)
	}
}

func TestCompressHistory_PersistsCompressedHistoryUTF8(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	provider := &compressionTestProvider{name: "openai", summary: strings.Repeat("要約", 120)}
	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", config.DefaultConfig())
	agent.History = []api.Message{
		{Role: "user", Content: strings.Repeat("古い文脈", 120)},
		{Role: "assistant", Content: "古い応答"},
		{Role: "user", Content: "最新の質問"},
	}
	for _, msg := range agent.History {
		agent.session.AddMessageFromAPI(msg, agent.CurrentModel)
	}
	agent.persistSession()

	if err := agent.CompressHistory(1); err != nil {
		t.Fatalf("CompressHistory() error = %v", err)
	}

	loaded, err := agent.storage.Load(agent.session.ID)
	if err != nil {
		t.Fatalf("storage.Load() error = %v", err)
	}
	loadedMessages := loaded.ToAPIMessages()
	if len(loadedMessages) == 0 {
		t.Fatal("loaded messages empty, want compressed summary")
	}
	if !utf8.ValidString(loadedMessages[0].Content) {
		t.Fatalf("loaded compressed summary is invalid UTF-8: %q", loadedMessages[0].Content)
	}
}

func TestMaybeAutoCompress_PersistsSessionHistoryWithoutNormalModePrompt(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	provider := &compressionTestProvider{name: "openai", summary: "auto persisted summary"}
	cfg := config.DefaultConfig()
	cfg.Compression.TokenThreshold = 1
	cfg.Compression.KeepRecent = 1
	cfg.Compression.PreferCompactAPI = false
	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", cfg)
	rawOld := "old full context"
	rawLatest := "latest question"
	agent.History = []api.Message{
		{Role: "user", Content: rawOld + promptnormal.NormalModePrompt},
		{Role: "assistant", Content: "older full answer"},
		{Role: "user", Content: rawLatest + promptnormal.NormalModePrompt},
	}
	agent.session.AddMessage("user", rawOld, agent.CurrentModel)
	agent.session.AddMessage("assistant", "older full answer", agent.CurrentModel)
	agent.session.AddMessage("user", rawLatest, agent.CurrentModel)
	agent.persistSession()

	if !agent.maybeAutoCompress() {
		t.Fatal("maybeAutoCompress() = false, want true")
	}

	loaded, err := agent.storage.Load(agent.session.ID)
	if err != nil {
		t.Fatalf("storage.Load() error = %v", err)
	}
	loadedMessages := loaded.ToAPIMessages()
	if len(loadedMessages) != 2 {
		t.Fatalf("len(loaded.ToAPIMessages()) = %d, want 2", len(loadedMessages))
	}
	if strings.Contains(loadedMessages[0].Content, promptnormal.NormalModePrompt) {
		t.Fatalf("loaded summary contains NormalModePrompt: %q", loadedMessages[0].Content)
	}
	if loadedMessages[1].Content != rawLatest {
		t.Fatalf("loaded latest message = %q, want %q", loadedMessages[1].Content, rawLatest)
	}
}

func TestCompressHistory_RestoresResponseContextOnSummaryError(t *testing.T) {
	provider := &compressionTestProvider{name: "openai", chatErr: errors.New("boom")}
	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", config.DefaultConfig())
	agent.History = []api.Message{
		{Role: "user", Content: "old"},
		{Role: "assistant", Content: "older"},
		{Role: "user", Content: "latest"},
	}
	setCompressionTestResponseContext(agent, provider, "resp_old")

	if err := agent.CompressHistory(1); err == nil {
		t.Fatal("CompressHistory() error = nil, want summary error")
	}

	if provider.capturedChatResponseID != "" {
		t.Fatalf("summary request response ID = %q, want empty", provider.capturedChatResponseID)
	}
	if provider.GetResponseID() != "resp_old" {
		t.Fatalf("provider response ID = %q, want restored resp_old", provider.GetResponseID())
	}
	if agent.session.ResponseID != "resp_old" {
		t.Fatalf("session.ResponseID = %q, want preserved resp_old", agent.session.ResponseID)
	}
}

func TestCompressHistory_RestoresStateOnInvalidSummaryJSON(t *testing.T) {
	provider := &compressionTestProvider{name: "openai", summary: `{bad json`}
	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", config.DefaultConfig())
	originalHistory := []api.Message{
		{Role: "user", Content: "old"},
		{Role: "assistant", Content: "older"},
		{Role: "user", Content: "latest"},
	}
	agent.History = append([]api.Message(nil), originalHistory...)
	for _, msg := range originalHistory {
		agent.session.AddMessageFromAPI(msg, agent.CurrentModel)
	}
	setCompressionTestResponseContext(agent, provider, "resp_old")

	if err := agent.CompressHistory(1); err == nil || !strings.Contains(err.Error(), "不正") {
		t.Fatalf("CompressHistory() error = %v, want invalid summary JSON error", err)
	}
	if provider.GetResponseID() != "resp_old" {
		t.Fatalf("provider response ID = %q, want restored resp_old", provider.GetResponseID())
	}
	if agent.session.ResponseID != "resp_old" {
		t.Fatalf("session.ResponseID = %q, want preserved resp_old", agent.session.ResponseID)
	}
	if len(agent.History) != len(originalHistory) {
		t.Fatalf("len(agent.History) = %d, want original %d", len(agent.History), len(originalHistory))
	}
	for i := range originalHistory {
		if agent.History[i].Role != originalHistory[i].Role || agent.History[i].Content != originalHistory[i].Content {
			t.Fatalf("agent.History[%d] = %#v, want %#v", i, agent.History[i], originalHistory[i])
		}
	}
}

func TestCompressWithCompactAPI_DetachesResponseContextOnSuccess(t *testing.T) {
	provider := &compressionTestProvider{name: "openai", supportsCompact: true}
	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", config.DefaultConfig())
	agent.History = []api.Message{{Role: "user", Content: "hello"}}
	setCompressionTestResponseContext(agent, provider, "resp_old")

	if err := agent.CompressWithCompactAPI(context.Background()); err != nil {
		t.Fatalf("CompressWithCompactAPI() error = %v", err)
	}

	if provider.capturedCompactResponseID != "" {
		t.Fatalf("compact request response ID = %q, want empty", provider.capturedCompactResponseID)
	}
	assertCompressionTestResponseContextCleared(t, agent, provider)
}

func TestCompressWithCompactAPI_PersistsCompactedStateBeforeClearingResponseContext(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	provider := &compressionTestProvider{
		name:            "openai",
		supportsCompact: true,
		compactOutput:   []api.InputItem{{Type: "compacted", Data: "compact-data"}},
	}
	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", config.DefaultConfig())
	agent.History = []api.Message{{Role: "user", Content: "old full context"}}
	agent.session.AddMessageFromAPI(agent.History[0], agent.CurrentModel)
	setCompressionTestResponseContext(agent, provider, "resp_old")
	agent.persistSession()

	if err := agent.CompressWithCompactAPI(context.Background()); err != nil {
		t.Fatalf("CompressWithCompactAPI() error = %v", err)
	}

	loaded, err := agent.storage.Load(agent.session.ID)
	if err != nil {
		t.Fatalf("storage.Load() error = %v", err)
	}
	if len(loaded.ToAPIMessages()) != 0 {
		t.Fatalf("len(loaded.ToAPIMessages()) = %d, want 0 after Compact API compression", len(loaded.ToAPIMessages()))
	}
	if !loaded.IsCompactedMode {
		t.Fatal("loaded.IsCompactedMode = false, want true")
	}
	if len(loaded.CompactedItems) != 1 || loaded.CompactedItems[0].Data != "compact-data" {
		t.Fatalf("loaded.CompactedItems = %#v, want compact-data", loaded.CompactedItems)
	}
	if loaded.ResponseID != "" {
		t.Fatalf("loaded.ResponseID = %q, want empty", loaded.ResponseID)
	}
}

func TestCompressWithCompactAPI_SendsInputWithoutNormalModePrompt(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	provider := &compressionTestProvider{
		name:            "openai",
		supportsCompact: true,
		compactOutput:   []api.InputItem{{Type: "compacted", Data: "compact-data"}},
	}
	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", config.DefaultConfig())
	raw := "normal request"
	agent.History = []api.Message{{Role: "user", Content: raw + promptnormal.NormalModePrompt}}
	agent.session.AddMessage("user", raw, agent.CurrentModel)
	agent.persistSession()

	if err := agent.CompressWithCompactAPI(context.Background()); err != nil {
		t.Fatalf("CompressWithCompactAPI() error = %v", err)
	}

	if len(provider.capturedCompactInput) != 1 {
		t.Fatalf("len(capturedCompactInput) = %d, want 1", len(provider.capturedCompactInput))
	}
	if provider.capturedCompactInput[0].Content != raw {
		t.Fatalf("captured compact input content = %#v, want %q", provider.capturedCompactInput[0].Content, raw)
	}
}

func setCompressionTestResponseContext(agent *Agent, provider *compressionTestProvider, responseID string) {
	provider.SetResponseID(responseID)
	agent.session.ResponseID = responseID
	agent.session.ResponseModel = agent.CurrentModel
	agent.session.ResponseProviderName = config.CanonicalProviderName(agent.ProviderName)
	agent.session.ResponseProviderConfigKey = agent.currentProviderConfigKey()
}

func assertCompressionTestResponseContextCleared(t *testing.T, agent *Agent, provider *compressionTestProvider) {
	t.Helper()

	if provider.GetResponseID() != "" {
		t.Fatalf("provider response ID = %q, want empty", provider.GetResponseID())
	}
	if agent.session.ResponseID != "" {
		t.Fatalf("session.ResponseID = %q, want empty", agent.session.ResponseID)
	}
	if agent.session.ResponseModel != "" {
		t.Fatalf("session.ResponseModel = %q, want empty", agent.session.ResponseModel)
	}
	if agent.session.ResponseProviderName != "" {
		t.Fatalf("session.ResponseProviderName = %q, want empty", agent.session.ResponseProviderName)
	}
	if agent.session.ResponseProviderConfigKey != "" {
		t.Fatalf("session.ResponseProviderConfigKey = %q, want empty", agent.session.ResponseProviderConfigKey)
	}
	if agent.session.ResponsePromptFingerprint != "" {
		t.Fatalf("session.ResponsePromptFingerprint = %q, want empty", agent.session.ResponsePromptFingerprint)
	}
}
