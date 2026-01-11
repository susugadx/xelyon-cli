package agent

import (
	"os"
	"testing"

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
	// XELYON.mdが存在しないディレクトリで実行
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalDir) }()

	_ = os.Chdir(tmpDir)

	config := loadProjectConfig()
	if config != "" {
		t.Errorf("loadProjectConfig() should return empty string when no XELYON.md, got %q", config)
	}
}

func TestLoadProjectConfig_WithFile(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalDir) }()

	// XELYON.mdを作成
	content := "# Test Project Config\nThis is a test."
	_ = os.WriteFile(tmpDir+"/XELYON.md", []byte(content), 0644)

	_ = os.Chdir(tmpDir)

	config := loadProjectConfig()
	if config != content {
		t.Errorf("loadProjectConfig() = %q, want %q", config, content)
	}
}
