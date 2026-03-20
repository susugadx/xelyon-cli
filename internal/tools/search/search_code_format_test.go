package search

import (
	"strings"
	"testing"
)

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

func TestTruncateToTokenBudget_SmartFlagDoesNotCollapse(t *testing.T) {
	var results []SearchResult
	for i := 0; i < 8; i++ {
		var matches []Match
		for j := 0; j < 8; j++ {
			matches = append(matches, Match{
				LineNum: j + 1,
				Line:    "test line",
				IsMatch: true,
			})
		}
		results = append(results, SearchResult{
			FilePath:   "file_" + string(rune('A'+i)) + ".go",
			Matches:    matches,
			MatchCount: 8,
		})
	}

	kept, truncated := truncateToTokenBudget(results, 10000, true)
	if truncated {
		t.Errorf("expected not to be truncated by budget")
	}

	if len(kept) != 8 {
		t.Fatalf("expected 8 files, got %d", len(kept))
	}

	for i := 0; i < 8; i++ {
		if len(kept[i].Matches) != 8 {
			t.Errorf("file %d: expected 8 code matches, got %d", i, len(kept[i].Matches))
		}
		if kept[i].MatchCount != 8 {
			t.Errorf("file %d: expected match count to remain 8, got %d", i, kept[i].MatchCount)
		}
	}
}

func TestTruncateToTokenBudget_ExplicitFullNoCollapse(t *testing.T) {
	var results []SearchResult
	for i := 0; i < 6; i++ {
		var matches []Match
		for j := 0; j < 5; j++ {
			matches = append(matches, Match{
				LineNum: j + 1,
				Line:    "test code line for full mode",
				IsMatch: true,
			})
		}
		results = append(results, SearchResult{
			FilePath:   "pkg_" + string(rune('A'+i)) + ".go",
			Matches:    matches,
			MatchCount: 5,
		})
	}

	kept, _ := truncateToTokenBudget(results, 10000, false)

	if len(kept) != 6 {
		t.Fatalf("expected 6 files, got %d", len(kept))
	}

	for i := 0; i < 6; i++ {
		if len(kept[i].Matches) == 0 {
			t.Errorf("file %d: expected code matches to be preserved in full mode, got 0", i)
		}
	}
}

func TestTruncateToTokenBudget_LightSearchNotCollapsed(t *testing.T) {
	var results []SearchResult
	for i := 0; i < 4; i++ {
		results = append(results, SearchResult{
			FilePath: "light_" + string(rune('A'+i)) + ".go",
			Matches: []Match{
				{LineNum: 10, Line: "match line", IsMatch: true},
			},
			MatchCount: 1,
		})
	}

	kept, _ := truncateToTokenBudget(results, 10000, true)

	if len(kept) != 4 {
		t.Fatalf("expected 4 files, got %d", len(kept))
	}

	for i := 0; i < 4; i++ {
		if len(kept[i].Matches) == 0 {
			t.Errorf("file %d: expected code matches to be preserved for light search, got 0", i)
		}
	}
}

func TestTruncateToTokenBudget_BroadSearchDoesNotCollapse(t *testing.T) {
	var results []SearchResult
	for i := 0; i < 10; i++ {
		var matches []Match
		for j := 0; j < 5; j++ {
			matches = append(matches, Match{
				LineNum: j + 1,
				Line:    "broad match line content",
				IsMatch: true,
			})
		}
		results = append(results, SearchResult{
			FilePath:   "broad_" + string(rune('A'+i)) + ".go",
			Matches:    matches,
			MatchCount: 5,
		})
	}

	kept, _ := truncateToTokenBudget(results, 10000, true)

	if len(kept) != 10 {
		t.Fatalf("expected 10 files, got %d", len(kept))
	}

	for i := 0; i < 10; i++ {
		if len(kept[i].Matches) == 0 {
			t.Errorf("file %d: expected code matches to remain visible, got 0", i)
		}
		if kept[i].MatchCount != 5 {
			t.Errorf("file %d: expected match count preserved as 5, got %d", i, kept[i].MatchCount)
		}
	}
}
