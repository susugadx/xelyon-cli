package agent

import (
	"os"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestIsSameToolCall(t *testing.T) {
	tests := []struct {
		name string
		tc1  *tools.ToolCall
		tc2  *tools.ToolCall
		want bool
	}{
		{
			name: "both nil",
			tc1:  nil,
			tc2:  nil,
			want: false,
		},
		{
			name: "first nil",
			tc1:  nil,
			tc2:  &tools.ToolCall{Tool: "test"},
			want: false,
		},
		{
			name: "second nil",
			tc1:  &tools.ToolCall{Tool: "test"},
			tc2:  nil,
			want: false,
		},
		{
			name: "different tool names",
			tc1:  &tools.ToolCall{Tool: "tool1"},
			tc2:  &tools.ToolCall{Tool: "tool2"},
			want: false,
		},
		{
			name: "same tool, no args",
			tc1:  &tools.ToolCall{Tool: "test", Args: map[string]string{}},
			tc2:  &tools.ToolCall{Tool: "test", Args: map[string]string{}},
			want: true,
		},
		{
			name: "same tool, same args",
			tc1: &tools.ToolCall{
				Tool: "read_file",
				Args: map[string]string{"path": "/test/file.txt"},
			},
			tc2: &tools.ToolCall{
				Tool: "read_file",
				Args: map[string]string{"path": "/test/file.txt"},
			},
			want: true,
		},
		{
			name: "same tool, different args",
			tc1: &tools.ToolCall{
				Tool: "read_file",
				Args: map[string]string{"path": "/test/file1.txt"},
			},
			tc2: &tools.ToolCall{
				Tool: "read_file",
				Args: map[string]string{"path": "/test/file2.txt"},
			},
			want: false,
		},
		{
			name: "same tool, different number of args",
			tc1: &tools.ToolCall{
				Tool: "test",
				Args: map[string]string{"a": "1"},
			},
			tc2: &tools.ToolCall{
				Tool: "test",
				Args: map[string]string{"a": "1", "b": "2"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSameToolCall(tt.tc1, tt.tc2)
			if got != tt.want {
				t.Errorf("isSameToolCall() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModelDisplayName(t *testing.T) {
	tests := []struct {
		name  string
		model string
		want  string
	}{
		{
			name:  "deepseek-chat",
			model: "deepseek-chat",
			want:  "DeepSeek V3 (balanced)",
		},
		{
			name:  "deepseek-coder",
			model: "deepseek-coder",
			want:  "DeepSeek Coder (code-focused)",
		},
		{
			name:  "deepseek-reasoner",
			model: "deepseek-reasoner",
			want:  "DeepSeek R1 (reasoning)",
		},
		{
			name:  "claude",
			model: "claude",
			want:  "Claude (Vertex AI)",
		},
		{
			name:  "unknown model",
			model: "gpt-4",
			want:  "gpt-4",
		},
		{
			name:  "custom model",
			model: "my-custom-model",
			want:  "my-custom-model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := modelDisplayName(tt.model)
			if got != tt.want {
				t.Errorf("modelDisplayName(%q) = %q, want %q", tt.model, got, tt.want)
			}
		})
	}
}

// TestNewAgent is skipped because NewAgent requires a valid provider
// and performs significant initialization (MCP, storage, etc.)
// These are better tested via integration tests.
func TestNewAgent(t *testing.T) {
	t.Skip("NewAgent requires a valid provider and performs I/O operations")
}

func TestIsAPIKeyAvailable(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		setup    func()
		cleanup  func()
		want     bool
	}{
		{
			name:     "Claude with API key",
			provider: "claude",
			setup: func() {
				os.Setenv("ANTHROPIC_API_KEY", "test-key")
			},
			cleanup: func() {
				os.Unsetenv("ANTHROPIC_API_KEY")
			},
			want: true,
		},
		{
			name:     "Claude without API key",
			provider: "claude",
			setup:    func() {},
			cleanup:  func() {},
			want:     false,
		},
		{
			name:     "OpenAI with API key",
			provider: "openai",
			setup: func() {
				os.Setenv("OPENAI_API_KEY", "test-key")
			},
			cleanup: func() {
				os.Unsetenv("OPENAI_API_KEY")
			},
			want: true,
		},
		{
			name:     "DeepSeek with API key",
			provider: "deepseek",
			setup: func() {
				os.Setenv("DEEPSEEK_API_KEY", "test-key")
			},
			cleanup: func() {
				os.Unsetenv("DEEPSEEK_API_KEY")
			},
			want: true,
		},
		{
			name:     "Unknown provider",
			provider: "unknown",
			setup:    func() {},
			cleanup:  func() {},
			want:     false,
		},
		{
			name:     "Gemini with API key",
			provider: "gemini",
			setup: func() {
				os.Setenv("GEMINI_API_KEY", "test-key")
			},
			cleanup: func() {
				os.Unsetenv("GEMINI_API_KEY")
			},
			want: true,
		},
		{
			name:     "Gemini without API key",
			provider: "gemini",
			setup:    func() {},
			cleanup:  func() {},
			want:     false,
		},
		{
			name:     "Groq with API key",
			provider: "groq",
			setup: func() {
				os.Setenv("GROQ_API_KEY", "test-key")
			},
			cleanup: func() {
				os.Unsetenv("GROQ_API_KEY")
			},
			want: true,
		},
		{
			name:     "Groq without API key",
			provider: "groq",
			setup:    func() {},
			cleanup:  func() {},
			want:     false,
		},
		{
			name:     "Ollama always available",
			provider: "ollama",
			setup:    func() {},
			cleanup:  func() {},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			defer tt.cleanup()

			got := IsAPIKeyAvailable(tt.provider)

			if got != tt.want {
				t.Errorf("IsAPIKeyAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseImageInput_NoImage(t *testing.T) {
	input := "Hello world"
	text, image := parseImageInput(input)

	if text != input {
		t.Errorf("parseImageInput() text = %q, want %q", text, input)
	}

	if image != nil {
		t.Error("parseImageInput() should return nil image when no image: prefix")
	}
}

func TestParseImageInput_ImagePrefixWithoutFile(t *testing.T) {
	input := "image:/nonexistent/file.png analyze this"
	text, image := parseImageInput(input)

	if text != input {
		t.Errorf("parseImageInput() text = %q, want %q", text, input)
	}

	if image != nil {
		t.Error("parseImageInput() should return nil image when file doesn't exist")
	}
}

func TestParseImageInput_EmptyTextWithImage(t *testing.T) {
	// image:だけでテキストがない場合のテスト
	// 実際のファイルが必要なのでスキップ
	t.Skip("Requires actual image file")
}

func TestAgent_SwitchProvider_NoAPIKey(t *testing.T) {
	// APIキーが設定されていない場合はスキップ
	t.Skip("Requires mock provider setup")
}

func TestLoadProjectConfig_NoFile(t *testing.T) {
	// xelyon.yaml も XELYON.md も存在しないディレクトリで実行
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalDir) }()

	_ = os.Chdir(tmpDir)

	pc := loadProjectConfig()
	if pc != nil {
		t.Errorf("loadProjectConfig() should return nil when no config file, got %+v", pc)
	}
}

func TestLoadProjectConfig_WithXELYONMD(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalDir) }()

	// XELYON.mdを作成
	content := "# Test Project Config\nThis is a test."
	_ = os.WriteFile(tmpDir+"/XELYON.md", []byte(content), 0644)

	_ = os.Chdir(tmpDir)

	pc := loadProjectConfig()
	if pc == nil {
		t.Fatal("loadProjectConfig() returned nil, want non-nil")
	}
	if !pc.IsLegacy {
		t.Error("expected IsLegacy=true for XELYON.md")
	}
	if pc.Context != content {
		t.Errorf("Context = %q, want %q", pc.Context, content)
	}
}

func TestLoadProjectConfig_WithXelyonYAML(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalDir) }()

	yamlContent := "context: \"test context\"\nrules:\n  - \"rule 1\"\n"
	_ = os.WriteFile(tmpDir+"/xelyon.yaml", []byte(yamlContent), 0644)

	_ = os.Chdir(tmpDir)

	pc := loadProjectConfig()
	if pc == nil {
		t.Fatal("loadProjectConfig() returned nil, want non-nil")
	}
	if pc.IsLegacy {
		t.Error("expected IsLegacy=false for xelyon.yaml")
	}
	if pc.Context != "test context" {
		t.Errorf("Context = %q, want %q", pc.Context, "test context")
	}
}

func TestAgent_AppendHistory(t *testing.T) {
	agent := &Agent{
		History: []api.Message{},
	}

	msg := api.Message{Role: "user", Content: "Hello"}
	agent.appendHistory(msg)

	if len(agent.History) != 1 {
		t.Errorf("appendHistory() history length = %d, want 1", len(agent.History))
	}
	if agent.History[0].Content != "Hello" {
		t.Errorf("appendHistory() content = %q, want %q", agent.History[0].Content, "Hello")
	}
}

func TestAgent_GetHistorySnapshot(t *testing.T) {
	agent := &Agent{
		History: []api.Message{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there"},
		},
	}

	snapshot := agent.getHistorySnapshot()

	if len(snapshot) != 2 {
		t.Errorf("getHistorySnapshot() length = %d, want 2", len(snapshot))
	}

	// Verify it's a copy
	snapshot[0].Content = "Modified"
	if agent.History[0].Content == "Modified" {
		t.Error("getHistorySnapshot() should return a copy, not a reference")
	}
}

func TestAgent_IncrementToolExecution(t *testing.T) {
	agent := &Agent{
		Stats: NewSessionStats("test"),
	}

	agent.incrementToolExecution("read_file")
	agent.incrementToolExecution("write_file")
	agent.incrementToolExecution("read_file")

	if agent.Stats.TotalToolExecutions() != 3 {
		t.Errorf("incrementToolExecution() total = %d, want 3", agent.Stats.TotalToolExecutions())
	}
}

func TestAgent_IncrementToolExecution_NilStats(t *testing.T) {
	agent := &Agent{
		Stats: nil,
	}

	// Should not panic
	agent.incrementToolExecution("read_file")
}

func TestAgent_IncrementAssistantMessages(t *testing.T) {
	agent := &Agent{
		Stats: NewSessionStats("test"),
	}

	agent.incrementAssistantMessages()
	agent.incrementAssistantMessages()

	if agent.Stats.AssistantMessages != 2 {
		t.Errorf("incrementAssistantMessages() count = %d, want 2", agent.Stats.AssistantMessages)
	}
}

func TestAgent_IncrementAssistantMessages_NilStats(t *testing.T) {
	agent := &Agent{
		Stats: nil,
	}

	// Should not panic
	agent.incrementAssistantMessages()
}

func TestAgent_AppendChange(t *testing.T) {
	agent := &Agent{
		changeStack: []tools.FileChange{},
	}

	change := tools.FileChange{
		FilePath: "/test/file.txt",
		Tool:     "write_file",
	}
	agent.appendChange(change)

	if len(agent.changeStack) != 1 {
		t.Errorf("appendChange() stack length = %d, want 1", len(agent.changeStack))
	}
	if agent.changeStack[0].FilePath != "/test/file.txt" {
		t.Errorf("appendChange() path = %q, want %q", agent.changeStack[0].FilePath, "/test/file.txt")
	}
}

func TestRemoveToolsSection(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "removes tools section",
			input: `## Core Identity
Some identity text.

## Available Tools
- tool1
- tool2

## Workflow Rules
Some rules.`,
			want: `## Core Identity
Some identity text.

## Workflow Rules
Some rules.`,
		},
		{
			name: "no tools section",
			input: `## Core Identity
Some identity text.

## Workflow Rules
Some rules.`,
			want: `## Core Identity
Some identity text.

## Workflow Rules
Some rules.`,
		},
		{
			name: "no workflow rules",
			input: `## Core Identity
Some identity text.

## Available Tools
- tool1`,
			want: `## Core Identity
Some identity text.

## Available Tools
- tool1`,
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removeToolsSection(tt.input)
			if got != tt.want {
				t.Errorf("removeToolsSection() = %q, want %q", got, tt.want)
			}
		})
	}
}
