package search

import (
	"strings"
	"testing"
)

func TestTruncateLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{
			name:     "500 characters or less",
			line:     strings.Repeat("a", 500),
			expected: strings.Repeat("a", 500),
		},
		{
			name:     "501 characters",
			line:     strings.Repeat("a", 501),
			expected: strings.Repeat("a", 500) + "...",
		},
		{
			name:     "empty line",
			line:     "",
			expected: "",
		},
		{
			name:     "multibyte characters (501 runes)",
			line:     strings.Repeat("あ", 501),
			expected: strings.Repeat("あ", 500) + "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateLine(tt.line)
			if result != tt.expected {
				t.Errorf("truncateLine() len=%d, expected len=%d", len([]rune(result)), len([]rune(tt.expected)))
			}
		})
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected int
	}{
		{
			name:     "English text",
			line:     "hello world",
			expected: 9,
		},
		{
			name:     "Japanese text",
			line:     "これはテストです",
			expected: 10,
		},
		{
			name:     "Mixed text",
			line:     "hello これはテスト world",
			expected: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := estimateTokens(tt.line)
			if result != tt.expected {
				t.Errorf("estimateTokens(%q) = %d, expected %d", tt.line, result, tt.expected)
			}
		})
	}
}

func TestFormatManifestResults_WithBlocks(t *testing.T) {
	results := []SearchResult{
		{
			FilePath:   "internal/agent/agent.go",
			MatchCount: 5,
			Matches: []Match{
				{LineNum: 10, Line: "x", IsMatch: true, Type: MatchTypeUsage, Block: &BlockInfo{Name: "func NewAgent", StartLine: 5}},
				{LineNum: 20, Line: "x", IsMatch: true, Type: MatchTypeUsage, Block: &BlockInfo{Name: "func Cleanup", StartLine: 15}},
				{LineNum: 30, Line: "x", IsMatch: true, Type: MatchTypeUsage, Block: &BlockInfo{Name: "func handleRequest", StartLine: 25}},
				{LineNum: 40, Line: "x", IsMatch: true, Type: MatchTypeUsage, Block: &BlockInfo{Name: "func fourthBlock", StartLine: 35}},
				{LineNum: 50, Line: "x", IsMatch: true, Type: MatchTypeUsage, Block: nil},
			},
		},
		{
			FilePath:   "internal/tools/execute.go",
			MatchCount: 2,
			Matches: []Match{
				{LineNum: 5, Line: "y", IsMatch: true, Type: MatchTypeUsage, Block: nil},
				{LineNum: 15, Line: "y", IsMatch: true, Type: MatchTypeUsage, Block: nil},
			},
		},
	}

	out := formatManifestResults(results)

	if !strings.Contains(out, "Found 7 matches in 2 files") {
		t.Errorf("expected header, got:\n%s", out)
	}
	if !strings.Contains(out, "agent.go") {
		t.Errorf("expected agent.go in output")
	}
	if !strings.Contains(out, "func NewAgent") || !strings.Contains(out, "func Cleanup") || !strings.Contains(out, "func handleRequest") {
		t.Errorf("expected first 3 block names, got:\n%s", out)
	}
	if strings.Contains(out, "fourthBlock") {
		t.Errorf("should not include 4th block name, got:\n%s", out)
	}
	if !strings.Contains(out, "execute.go") || !strings.Contains(out, "2 matches") {
		t.Errorf("expected execute.go with 2 matches, got:\n%s", out)
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "│") {
			t.Errorf("manifest mode should not contain code snippets, got line: %s", line)
		}
	}
}

func TestFormatManifestResults_NoBlocks(t *testing.T) {
	results := []SearchResult{{
		FilePath:   "main.go",
		MatchCount: 3,
		Matches: []Match{
			{LineNum: 1, Line: "a", IsMatch: true, Type: MatchTypeUsage},
			{LineNum: 2, Line: "b", IsMatch: true, Type: MatchTypeUsage},
			{LineNum: 3, Line: "c", IsMatch: true, Type: MatchTypeUsage},
		},
	}}

	out := formatManifestResults(results)
	if !strings.Contains(out, "3 matches") {
		t.Errorf("expected 3 matches, got:\n%s", out)
	}
	if strings.Contains(out, "(") {
		t.Errorf("should not have block names in parens, got:\n%s", out)
	}
}

func TestFormatManifestMultiResults(t *testing.T) {
	collected := []patternResult{
		{
			Pattern: "handleSSE",
			Results: []SearchResult{{
				FilePath:   "stream.go",
				MatchCount: 3,
				Matches: []Match{
					{LineNum: 10, Line: "x", IsMatch: true, Type: MatchTypeUsage, Block: &BlockInfo{Name: "func handleSSE", StartLine: 5}},
					{LineNum: 20, Line: "x", IsMatch: true, Type: MatchTypeUsage, Block: &BlockInfo{Name: "func handleSSE", StartLine: 5}},
					{LineNum: 30, Line: "x", IsMatch: true, Type: MatchTypeUsage, Block: &BlockInfo{Name: "func parseEvent", StartLine: 25}},
				},
			}},
		},
		{
			Pattern: "badPattern",
			Error:   "regex syntax error",
		},
	}

	out := formatManifestMultiResults(collected)
	if !strings.Contains(out, "handleSSE") {
		t.Errorf("expected pattern name")
	}
	if !strings.Contains(out, "stream.go") {
		t.Errorf("expected file path")
	}
	if !strings.Contains(out, "⚠️ Error: regex syntax error") {
		t.Errorf("expected error for bad pattern, got:\n%s", out)
	}
}

func TestFormatMultiResults_WithPatternError(t *testing.T) {
	collected := []patternResult{
		{Pattern: "ok", Results: []SearchResult{{FilePath: "a.go", MatchCount: 1, Matches: []Match{{LineNum: 1, Line: "ok", IsMatch: true, Type: MatchTypeUsage}}}}},
		{Pattern: "bad", Error: "regex error"},
	}
	out := formatMultiResults(collected, 3000)
	if !strings.Contains(out, "⚠️ Error: regex error") {
		t.Fatalf("expected pattern error to be shown, got: %s", out)
	}
}
