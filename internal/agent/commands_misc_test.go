package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestIsCodexModel(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{"gpt-5.2-codex", true},
		{"gpt-5.1-codex", true},
		{"gpt-5.1-codex-max", true},
		{"gpt-5-codex", true},
		{"GPT-5.2-CODEX", true}, // case insensitive
		{"gpt-5.2-Codex", true}, // mixed case
		{"gpt-4o", false},
		{"gpt-5.2", false},
		{"claude-3-opus", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := isCodexModel(tt.model)
			if got != tt.want {
				t.Errorf("isCodexModel(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}

func TestExtractCodeBlocks(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{
			name: "single code block with language",
			text: "```go\nfunc main() {}\n```",
			want: []string{"func main() {}"},
		},
		{
			name: "single code block without language",
			text: "```\ncode here\n```",
			want: []string{"code here"},
		},
		{
			name: "multiple code blocks",
			text: "```go\nfunc foo() {}\n```\n\nSome text\n\n```python\ndef bar():\n    pass\n```",
			want: []string{"func foo() {}", "def bar():\n    pass"},
		},
		{
			name: "no code blocks",
			text: "Just some regular text without any code blocks",
			want: []string{},
		},
		{
			name: "empty string",
			text: "",
			want: []string{},
		},
		{
			name: "code block with extra whitespace",
			text: "```js\n  const x = 1;  \n```",
			want: []string{"const x = 1;"},
		},
		{
			name: "nested backticks in text",
			text: "Use `inline code` like this\n```go\nfunc test() {}\n```",
			want: []string{"func test() {}"},
		},
		{
			name: "code block with multiline content",
			text: "```typescript\ninterface User {\n  name: string;\n  age: number;\n}\n```",
			want: []string{"interface User {\n  name: string;\n  age: number;\n}"},
		},
		{
			name: "unclosed code block",
			text: "```go\nfunc incomplete()",
			want: []string{},
		},
		{
			name: "code block with different languages",
			text: "```rust\nfn main() {}\n```\n```cpp\nint main() {}\n```",
			want: []string{"fn main() {}", "int main() {}"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCodeBlocks(tt.text)
			if len(got) != len(tt.want) {
				t.Errorf("extractCodeBlocks() returned %d blocks, want %d", len(got), len(tt.want))
				t.Errorf("got: %v", got)
				t.Errorf("want: %v", tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("extractCodeBlocks()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestRequestCacheMode(t *testing.T) {
	tests := []struct {
		name  string
		usage api.Usage
		want  string
	}{
		{name: "none", usage: api.Usage{}, want: "none"},
		{name: "read", usage: api.Usage{InputTokens: 100, CachedInputTokens: 40}, want: "read"},
		{name: "create", usage: api.Usage{InputTokens: 100, CacheCreationTokens: 40}, want: "create"},
		{name: "read and create", usage: api.Usage{InputTokens: 100, CachedInputTokens: 20, CacheCreationTokens: 30}, want: "read + create"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := requestCacheMode(tt.usage); got != tt.want {
				t.Fatalf("requestCacheMode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRequestCacheHitRate(t *testing.T) {
	usage := api.Usage{InputTokens: 200, CachedInputTokens: 50}
	if got := requestCacheHitRate(usage); got != 25.0 {
		t.Fatalf("requestCacheHitRate() = %.1f, want 25.0", got)
	}
}
