package tools

import "testing"

func TestParseToolCalls_InMarkdownCodeBlock(t *testing.T) {
	input := "Here is an example:\n```json\n{\"tool\": \"read_file\", \"args\": {\"path\": \"test.go\"}}\n```\nThis is just an example."

	result := ParseToolCalls(input)
	if len(result) != 0 {
		t.Errorf("ParseToolCalls() returned %d calls from markdown block, want 0", len(result))
	}
}

func TestParseToolCalls_CodeBlockWithRealToolCall(t *testing.T) {
	// Geminiパターン: コードブロック内のJSONは無視し、実際のツール呼び出しのみ抽出
	input := "I'll run the command:\n```json\n{\"tool\": \"bash\", \"args\": {\"command\": \"make ci-check\"}}\n```\n\n{\"tool\": \"bash\", \"args\": {\"command\": \"make ci-check\"}}"

	result := ParseToolCalls(input)
	if len(result) != 1 {
		t.Errorf("ParseToolCalls() returned %d calls, want 1 (only the real one)", len(result))
	}
	if len(result) > 0 && result[0].Tool != "bash" {
		t.Errorf("Tool = %q, want 'bash'", result[0].Tool)
	}
}

func TestParseToolCalls_OnlyCodeBlock(t *testing.T) {
	input := "Here's what I would run:\n```\n{\"tool\": \"bash\", \"args\": {\"command\": \"go test ./...\"}}\n```\nTask completed!"

	result := ParseToolCalls(input)
	if len(result) != 0 {
		t.Errorf("ParseToolCalls() returned %d calls, want 0 (all in code block)", len(result))
	}
}

func TestParseToolCalls_MultipleCodeBlocks(t *testing.T) {
	input := "Example 1:\n```json\n{\"tool\": \"read_file\", \"args\": {\"path\": \"a.go\"}}\n```\n\n{\"tool\": \"bash\", \"args\": {\"command\": \"ls\"}}\n\nExample 2:\n```\n{\"tool\": \"write_file\", \"args\": {\"path\": \"b.go\", \"content\": \"test\"}}\n```"

	result := ParseToolCalls(input)
	if len(result) != 1 {
		t.Errorf("ParseToolCalls() returned %d calls, want 1", len(result))
	}
	if len(result) > 0 && result[0].Tool != "bash" {
		t.Errorf("Tool = %q, want 'bash'", result[0].Tool)
	}
}

func TestParseToolCalls_UnclosedCodeBlock(t *testing.T) {
	tests := []struct {
		name      string
		response  string
		wantCount int
		wantTools []string
	}{
		{
			name:      "unclosed code block before tool (CURRENT BEHAVIOR - may cause issues)",
			response:  "```json\nsome example\n{\"tool\": \"read_file\", \"args\": {}}",
			wantCount: 0,
			wantTools: []string{},
		},
		{
			name:      "tool after properly closed code block",
			response:  "```json\nexample\n```\n{\"tool\": \"read_file\", \"args\": {}}",
			wantCount: 1,
			wantTools: []string{"read_file"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseToolCalls(tt.response)
			if len(got) != tt.wantCount {
				t.Fatalf("ParseToolCalls() returned %d tools, want %d", len(got), tt.wantCount)
			}
			for i, tc := range got {
				if tc.Tool != tt.wantTools[i] {
					t.Errorf("ParseToolCalls()[%d].Tool = %v, want %v", i, tc.Tool, tt.wantTools[i])
				}
			}
		})
	}
}

func TestParseToolCalls_IncompleteJSONInCodeBlock_DoesNotBlockOutsideJSON(t *testing.T) {
	input := "```json\n{\"tool\": \"write_file\", \"args\": {\"path\": \"a.go\"\n```\n\n{\"tool\": \"bash\", \"args\": {\"command\": \"echo ok\"}}"

	result := ParseToolCalls(input)
	if len(result) != 1 {
		t.Fatalf("ParseToolCalls() returned %d calls, want 1", len(result))
	}
	if result[0].Tool != "bash" {
		t.Errorf("Tool = %q, want 'bash'", result[0].Tool)
	}
}
