package agent

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func newSessionCommandTestAgent(out *bytes.Buffer) *Agent {
	return &Agent{
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader(""), out, out),
		},
	}
}

func TestSessionCommands_ReportMissingStorage(t *testing.T) {
	var out bytes.Buffer
	agent := newSessionCommandTestAgent(&out)

	if !handleSaveCommand(agent) {
		t.Fatal("handleSaveCommand() = false, want true")
	}
	if !handleLoadCommand(agent, nil) {
		t.Fatal("handleLoadCommand() = false, want true")
	}
	if !handleSessionsCommand(agent) {
		t.Fatal("handleSessionsCommand() = false, want true")
	}

	if count := strings.Count(out.String(), "History storage not available"); count != 3 {
		t.Fatalf("output = %q, want 3 missing-storage warnings", out.String())
	}
}

func TestHandleLoadCommand_NoLastSessionAndMissingSession(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	storage, err := history.NewStorage()
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}

	t.Run("no last session", func(t *testing.T) {
		var out bytes.Buffer
		agent := newSessionCommandTestAgent(&out)
		agent.storage = storage

		if !handleLoadCommand(agent, nil) {
			t.Fatal("handleLoadCommand() = false, want true")
		}
		if !strings.Contains(out.String(), "No sessions found") {
			t.Fatalf("output = %q, want no sessions message", out.String())
		}
	})

	t.Run("explicit missing session reports load error", func(t *testing.T) {
		var out bytes.Buffer
		agent := newSessionCommandTestAgent(&out)
		agent.storage = storage

		if !handleLoadCommand(agent, []string{"missing-session"}) {
			t.Fatal("handleLoadCommand() = false, want true")
		}
		if !strings.Contains(out.String(), "Failed to load session") {
			t.Fatalf("output = %q, want load error", out.String())
		}
	})
}

func TestHandleLoadCommand_RestoresResponseIDAndListsEmptySessions(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	storage, err := history.NewStorage()
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}

	session := history.NewSession("test-model")
	session.ResponseID = "resp_123"
	session.ProviderName = "openai"
	session.ProviderConfigKey = "openai"
	session.AddMessage("user", strings.Repeat("preview", 20), "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	var out bytes.Buffer
	agent := newSessionCommandTestAgent(&out)
	agent.storage = storage
	agent.CurrentModel = "test-model"
	agent.ProviderName = "openai"
	agent.ProviderConfigKey = "openai"
	agent.CurrentProvider = &mockResponseIDProvider{mockProvider: mockProvider{name: "openai"}}

	if !handleLoadCommand(agent, nil) {
		t.Fatal("handleLoadCommand() = false, want true")
	}
	if ridProvider, ok := agent.CurrentProvider.(*mockResponseIDProvider); !ok || ridProvider.responseID != "resp_123" {
		t.Fatalf("responseID = %#v, want resp_123", agent.CurrentProvider)
	}

	out.Reset()
	t.Setenv("HOME", t.TempDir())
	emptyStorage, err := history.NewStorage()
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}
	agent.storage = emptyStorage
	if !handleSessionsCommand(agent) {
		t.Fatal("handleSessionsCommand() = false, want true")
	}
	if !strings.Contains(out.String(), "No sessions found") {
		t.Fatalf("output = %q, want no sessions message", out.String())
	}
}

func TestHandleLoadCommand_RepairsVersion1GuessedOpenAIOwnerForAliasRuntime(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	storage, err := history.NewStorage()
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}

	session := history.NewSession("test-model")
	session.ResponseID = "resp_v1"
	session.ProviderName = "openai"
	session.ProviderConfigKey = "openai-alt"
	session.ResponseModel = "test-model"
	session.ResponseProviderName = "openai"
	session.ResponseProviderConfigKey = "openai"
	session.AddMessage("user", "hello", "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	metaPath := filepath.Join(os.Getenv("HOME"), ".xelyon", "history", "metadata", session.ID+".json")
	raw, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("ReadFile(metadata) error = %v", err)
	}

	var meta history.SessionMetadata
	if err := json.Unmarshal(raw, &meta); err != nil {
		t.Fatalf("Unmarshal(metadata) error = %v", err)
	}
	meta.ResponseContextVersion = 1
	version1Raw, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent(metadata) error = %v", err)
	}
	if err := os.WriteFile(metaPath, version1Raw, 0o600); err != nil {
		t.Fatalf("WriteFile(metadata) error = %v", err)
	}

	var out bytes.Buffer
	agent := newSessionCommandTestAgent(&out)
	agent.storage = storage
	agent.CurrentModel = "test-model"
	agent.ProviderName = "openai"
	agent.ProviderConfigKey = "openai-alt"
	agent.CurrentProvider = &mockResponseIDProvider{mockProvider: mockProvider{name: "openai"}}

	if !handleLoadCommand(agent, []string{session.ID}) {
		t.Fatal("handleLoadCommand() = false, want true")
	}
	if ridProvider, ok := agent.CurrentProvider.(*mockResponseIDProvider); !ok || ridProvider.responseID != "resp_v1" {
		t.Fatalf("responseID = %#v, want resp_v1", agent.CurrentProvider)
	}
}

func TestHandleSessionsCommand_TruncatesPreview(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	storage, err := history.NewStorage()
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}
	session := history.NewSession("test-model")
	session.AddMessage("user", strings.Repeat("long-preview-", 20), "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	var out bytes.Buffer
	agent := newSessionCommandTestAgent(&out)
	agent.storage = storage

	if !handleSessionsCommand(agent) {
		t.Fatal("handleSessionsCommand() = false, want true")
	}
	if !strings.Contains(out.String(), "...") {
		t.Fatalf("output = %q, want truncated preview ellipsis", out.String())
	}
}
