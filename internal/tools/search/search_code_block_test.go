package search

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func TestBuildMatchBlocks(t *testing.T) {
	matches := []Match{
		{LineNum: 1, Line: "ctx1", IsMatch: false, Type: MatchTypeUsage},
		{LineNum: 2, Line: "match1", IsMatch: true, Type: MatchTypeDefinition},
		{LineNum: 3, Line: "ctx2", IsMatch: false, Type: MatchTypeUsage},
		{LineNum: 4, Line: "ctx3", IsMatch: false, Type: MatchTypeUsage},
		{LineNum: 5, Line: "match2", IsMatch: true, Type: MatchTypeUsage},
		{LineNum: 6, Line: "ctx4", IsMatch: false, Type: MatchTypeUsage},
	}

	blocks := buildMatchBlocks(matches)

	if len(blocks) != 2 {
		t.Fatalf("Expected 2 blocks, got %d", len(blocks))
	}
	if blocks[0].typ != MatchTypeDefinition {
		t.Errorf("Block 0 typ = %d, want %d", blocks[0].typ, MatchTypeDefinition)
	}
	if len(blocks[0].matches) != 2 {
		t.Errorf("Block 0 has %d matches, want 2", len(blocks[0].matches))
	}
	if blocks[1].typ != MatchTypeUsage {
		t.Errorf("Block 1 typ = %d, want %d", blocks[1].typ, MatchTypeUsage)
	}
	if len(blocks[1].matches) != 4 {
		t.Errorf("Block 1 has %d matches, want 4", len(blocks[1].matches))
	}
}

func TestFindBlockForLine(t *testing.T) {
	ranges := []common.BlockRange{
		{Name: "func outer", StartLine: 1, EndLine: 20},
		{Name: "func inner", StartLine: 5, EndLine: 10},
	}

	b := findBlockForLine(ranges, 7)
	if b == nil || b.Name != "func inner" {
		t.Errorf("Expected innermost block 'func inner' for line 7, got: %v", b)
	}

	b = findBlockForLine(ranges, 15)
	if b == nil || b.Name != "func outer" {
		t.Errorf("Expected 'func outer' for line 15, got: %v", b)
	}

	b = findBlockForLine(ranges, 25)
	if b != nil {
		t.Errorf("Expected nil for line 25, got: %v", b)
	}

	b = findBlockForLine(nil, 1)
	if b != nil {
		t.Errorf("Expected nil for empty ranges, got: %v", b)
	}

	b = findBlockForLine(ranges, 5)
	if b == nil || b.Name != "func inner" {
		t.Errorf("Expected 'func inner' for start line 5, got: %v", b)
	}

	b = findBlockForLine(ranges, 10)
	if b == nil || b.Name != "func inner" {
		t.Errorf("Expected 'func inner' for end line 10, got: %v", b)
	}
}

func TestCollapseBlockMatches_ThreeMatchesSameBlock(t *testing.T) {
	block := &BlockInfo{Name: "func doStuff", StartLine: 10}
	results := []SearchResult{{
		FilePath:   "a.go",
		MatchCount: 3,
		Matches: []Match{
			{LineNum: 11, Line: "ctx line before", IsMatch: false},
			{LineNum: 12, Line: "match1", IsMatch: true, Type: MatchTypeUsage, Block: block},
			{LineNum: 13, Line: "ctx between 1-2", IsMatch: false},
			{LineNum: 14, Line: "match2", IsMatch: true, Type: MatchTypeUsage, Block: block},
			{LineNum: 15, Line: "ctx between 2-3", IsMatch: false},
			{LineNum: 16, Line: "match3", IsMatch: true, Type: MatchTypeUsage, Block: block},
			{LineNum: 17, Line: "ctx line after", IsMatch: false},
		},
	}}

	collapseBlockMatches(results)

	matches := results[0].Matches
	if len(matches) != 5 {
		t.Fatalf("expected 5 entries, got %d: %+v", len(matches), matches)
	}
	if matches[0].LineNum != 11 {
		t.Errorf("expected pre-context at line 11, got %d", matches[0].LineNum)
	}
	if matches[1].LineNum != 12 || !matches[1].IsMatch {
		t.Errorf("expected first match at line 12")
	}
	if matches[2].LineNum != -1 {
		t.Errorf("expected collapse marker (LineNum=-1), got %d", matches[2].LineNum)
	}
	if !strings.Contains(matches[2].Line, "+1 more match") {
		t.Errorf("expected collapse marker text, got %q", matches[2].Line)
	}
	if matches[3].LineNum != 16 || !matches[3].IsMatch {
		t.Errorf("expected last match at line 16")
	}
	if matches[4].LineNum != 17 {
		t.Errorf("expected post-context at line 17, got %d", matches[4].LineNum)
	}
}

func TestCollapseBlockMatches_TwoMatchesNoCollapse(t *testing.T) {
	block := &BlockInfo{Name: "func foo", StartLine: 5}
	results := []SearchResult{{
		FilePath:   "b.go",
		MatchCount: 2,
		Matches: []Match{
			{LineNum: 6, Line: "match1", IsMatch: true, Type: MatchTypeUsage, Block: block},
			{LineNum: 7, Line: "match2", IsMatch: true, Type: MatchTypeUsage, Block: block},
		},
	}}

	collapseBlockMatches(results)

	if len(results[0].Matches) != 2 {
		t.Fatalf("expected 2 entries (no collapse), got %d", len(results[0].Matches))
	}
}

func TestCollapseBlockMatches_NilBlockNoCollapse(t *testing.T) {
	results := []SearchResult{{
		FilePath:   "c.go",
		MatchCount: 3,
		Matches: []Match{
			{LineNum: 1, Line: "match1", IsMatch: true, Type: MatchTypeUsage, Block: nil},
			{LineNum: 2, Line: "match2", IsMatch: true, Type: MatchTypeUsage, Block: nil},
			{LineNum: 3, Line: "match3", IsMatch: true, Type: MatchTypeUsage, Block: nil},
		},
	}}

	collapseBlockMatches(results)

	if len(results[0].Matches) != 3 {
		t.Fatalf("expected 3 entries (nil block, no collapse), got %d", len(results[0].Matches))
	}
}

func TestCollapseBlockMatches_DifferentBlocks(t *testing.T) {
	blockA := &BlockInfo{Name: "func a", StartLine: 10}
	blockB := &BlockInfo{Name: "func b", StartLine: 30}
	results := []SearchResult{{
		FilePath:   "d.go",
		MatchCount: 4,
		Matches: []Match{
			{LineNum: 12, Line: "a1", IsMatch: true, Type: MatchTypeUsage, Block: blockA},
			{LineNum: 14, Line: "a2", IsMatch: true, Type: MatchTypeUsage, Block: blockA},
			{LineNum: 32, Line: "b1", IsMatch: true, Type: MatchTypeUsage, Block: blockB},
			{LineNum: 34, Line: "b2", IsMatch: true, Type: MatchTypeUsage, Block: blockB},
		},
	}}

	collapseBlockMatches(results)

	if len(results[0].Matches) != 4 {
		t.Fatalf("expected 4 entries (different blocks <3), got %d", len(results[0].Matches))
	}
}

func TestCollapseBlockMatches_FiveMatches(t *testing.T) {
	block := &BlockInfo{Name: "func big", StartLine: 1}
	results := []SearchResult{{
		FilePath:   "e.go",
		MatchCount: 5,
		Matches: []Match{
			{LineNum: 2, Line: "m1", IsMatch: true, Type: MatchTypeUsage, Block: block},
			{LineNum: 4, Line: "m2", IsMatch: true, Type: MatchTypeUsage, Block: block},
			{LineNum: 6, Line: "m3", IsMatch: true, Type: MatchTypeUsage, Block: block},
			{LineNum: 8, Line: "m4", IsMatch: true, Type: MatchTypeUsage, Block: block},
			{LineNum: 10, Line: "m5", IsMatch: true, Type: MatchTypeUsage, Block: block},
		},
	}}

	collapseBlockMatches(results)

	matches := results[0].Matches
	if len(matches) != 3 {
		t.Fatalf("expected 3 entries, got %d: %+v", len(matches), matches)
	}
	if matches[0].LineNum != 2 {
		t.Errorf("expected first match at line 2")
	}
	if matches[1].LineNum != -1 || !strings.Contains(matches[1].Line, "+3 more match") {
		t.Errorf("expected collapse marker with +3, got %q", matches[1].Line)
	}
	if matches[2].LineNum != 10 {
		t.Errorf("expected last match at line 10")
	}
}

func TestCollapseBlockMatches_FormatterRendering(t *testing.T) {
	block := &BlockInfo{Name: "func render", StartLine: 5}
	results := []SearchResult{{
		FilePath:   "f.go",
		MatchCount: 3,
		Matches: []Match{
			{LineNum: 6, Line: "first", IsMatch: true, Type: MatchTypeUsage, Block: block},
			{LineNum: 7, Line: "middle", IsMatch: true, Type: MatchTypeUsage, Block: block},
			{LineNum: 8, Line: "last", IsMatch: true, Type: MatchTypeUsage, Block: block},
		},
	}}

	collapseBlockMatches(results)
	out := formatSearchResults(results, false, 3000)

	if !strings.Contains(out, "+1 more match") {
		t.Errorf("expected collapse marker in formatted output, got:\n%s", out)
	}
	if strings.Contains(out, "middle") {
		t.Errorf("expected middle match to be collapsed, got:\n%s", out)
	}
}
