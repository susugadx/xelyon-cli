package tools

import (
	"testing"
)

// NOTE: ParseToolCall/ParseToolCalls tests are in parsing_test.go
// NOTE: Backup tests are in common/backup_test.go
// NOTE: Truncate, NormalizeLeadingWhitespace, MinMax tests are in common/helpers_test.go
// NOTE: TestDetectTestFramework etc. are in dev/ package

// ===== Tests for parsing.go internal functions =====

func TestFindCodeBlockRanges(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		wantRanges int
	}{
		{
			name:       "no code blocks",
			text:       "plain text without code blocks",
			wantRanges: 0,
		},
		{
			name:       "single code block",
			text:       "text ```code``` more",
			wantRanges: 1,
		},
		{
			name:       "two code blocks",
			text:       "```a``` middle ```b```",
			wantRanges: 2,
		},
		{
			name:       "unclosed code block",
			text:       "```start of code block",
			wantRanges: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findCodeBlockRanges(tt.text)
			if len(got) != tt.wantRanges {
				t.Errorf("findCodeBlockRanges() returned %d ranges, want %d", len(got), tt.wantRanges)
			}
		})
	}
}

// TestParseToolCalls_UnclosedCodeBlock はコードブロックが閉じていない場合の挙動をテスト
// この問題が起こりうる: AIがコードブロックを開始して閉じ忘れた場合、
// その後のすべてのツールJSONがコードブロック内として扱われてスキップされる
func TestParseToolCalls_UnclosedCodeBlock(t *testing.T) {
	tests := []struct {
		name      string
		response  string
		wantCount int
		wantTools []string
	}{
		{
			name: "unclosed code block before tool (CURRENT BEHAVIOR - may cause issues)",
			// 現在の実装では、閉じていないコードブロック以降がすべてスキップされる
			response:  "```json\nsome example\n{\"tool\": \"read_file\", \"args\": {}}",
			wantCount: 0, // 現在の挙動: コードブロック内なのでスキップ
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
