package tools

import (
	"testing"
)

// NOTE: Tool registration tests are in each subpackage:
// - dev/bash_test.go tests BashTool
// - file/read_test.go tests ReadFileTool
// - git/commit_test.go tests GitCommitTool
// - search/code_test.go tests SearchCodeTool
// - lsp/tools_test.go tests LSP tools

// ===== parseInt Tests =====

func TestParseInt(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{
			name:    "positive integer",
			input:   "42",
			want:    42,
			wantErr: false,
		},
		{
			name:    "zero",
			input:   "0",
			want:    0,
			wantErr: false,
		},
		{
			name:    "negative integer",
			input:   "-10",
			want:    -10,
			wantErr: false,
		},
		{
			name:    "large positive",
			input:   "999999",
			want:    999999,
			wantErr: false,
		},
		{
			name:    "large negative",
			input:   "-123456",
			want:    -123456,
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			want:    0,
			wantErr: true,
		},
		{
			name:    "non-numeric string",
			input:   "abc",
			want:    0,
			wantErr: true,
		},
		{
			name:    "float string",
			input:   "3.14",
			want:    0,
			wantErr: true,
		},
		{
			name:    "string with spaces",
			input:   " 10",
			want:    0,
			wantErr: true,
		},
		{
			name:    "mixed alphanumeric",
			input:   "123abc",
			want:    0,
			wantErr: true,
		},
		{
			name:    "plus sign prefix",
			input:   "+5",
			want:    5,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseInt(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseInt(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseInt(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// ===== Registry Tests =====

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewRegistry()

	// Create a mock tool
	mock := &mockTool{name: "test_tool"}
	r.Register(mock)

	// Test GetTool
	got := r.GetTool("test_tool")
	if got == nil {
		t.Fatal("GetTool() returned nil")
	}
	if got.Name() != "test_tool" {
		t.Errorf("GetTool().Name() = %v, want 'test_tool'", got.Name())
	}

	// Test GetTool for non-existent tool
	got = r.GetTool("nonexistent")
	if got != nil {
		t.Errorf("GetTool('nonexistent') should return nil, got %v", got)
	}
}

func TestRegistry_RegisterMultiple(t *testing.T) {
	r := NewRegistry()

	mock1 := &mockTool{name: "tool_a"}
	mock2 := &mockTool{name: "tool_b"}
	r.Register(mock1)
	r.Register(mock2)

	// Verify both tools can be retrieved
	if got := r.GetTool("tool_a"); got == nil {
		t.Error("GetTool('tool_a') returned nil")
	}
	if got := r.GetTool("tool_b"); got == nil {
		t.Error("GetTool('tool_b') returned nil")
	}
}

// mockTool implements Tool interface for testing
type mockTool struct {
	name string
}

func (m *mockTool) Name() string { return m.name }

func (m *mockTool) Run(args map[string]string) (string, *FileChange, error) {
	return "mock output", nil, nil
}
