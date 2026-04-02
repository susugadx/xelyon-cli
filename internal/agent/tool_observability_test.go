package agent

import (
	"bytes"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

// ── isBatchPaths ──

func TestIsBatchPaths(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want bool
	}{
		{"slice 2 entries", []any{"a.go", "b.go"}, true},
		{"slice 3 entries", []any{"a.go", "b.go", "c.go"}, true},
		{"slice 1 entry", []any{"a.go"}, false},
		{"slice empty", []any{}, false},
		{"json 2 entries", `["a.go","b.go"]`, true},
		{"json 1 entry", `["a.go"]`, false},
		{"json empty", `[]`, false},
		{"json with leading newline", "\n[\"a.go\",\"b.go\"]\n", true},
		{"json with leading spaces", "  [\"a.go\",\"b.go\"]  ", true},
		{"json with mixed whitespace", " \t\n[\"a.go\",\"b.go\"]\n ", true},
		{"json 1 entry with whitespace", " \n[\"a.go\"]\n ", false},
		{"string non-json", "a.go", false},
		{"nil", nil, false},
		{"int", 42, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isBatchPaths(tt.val)
			if got != tt.want {
				t.Errorf("isBatchPaths(%v) = %v, want %v", tt.val, got, tt.want)
			}
		})
	}
}

// ── isMultiPatternArg ──

func TestIsMultiPatternArg(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want bool
	}{
		{"two patterns", "func_a,func_b", true},
		{"three patterns", "a,b,c", true},
		{"single pattern", "funcName", false},
		{"escaped comma", `hello\,world`, false},
		{"escaped + real comma", `hello\,world,other`, true},
		{"empty", "", false},
		{"non-string", 42, false},
		{"nil", nil, false},
		{"only commas", ",,", false},
		{"spaces around", " a , b ", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isMultiPatternArg(tt.val)
			if got != tt.want {
				t.Errorf("isMultiPatternArg(%v) = %v, want %v", tt.val, got, tt.want)
			}
		})
	}
}

// ── recordToolObservability (FC path: RawArgs あり) ──

func TestRecordToolObservability_BatchRead(t *testing.T) {
	a := &Agent{Stats: NewSessionStats("test")}
	a.recordToolObservability("read_file", map[string]any{
		"paths": []any{"a.go", "b.go"},
	}, nil, "file contents...")

	if a.Stats.ToolObs.ReadFileBatchCalls != 1 {
		t.Errorf("ReadFileBatchCalls = %d, want 1", a.Stats.ToolObs.ReadFileBatchCalls)
	}
}

func TestRecordToolResultOptimizations_OutlineFooter(t *testing.T) {
	a := &Agent{Stats: NewSessionStats("test")}
	tc := &tools.ToolCall{Tool: "read_file"}

	a.recordToolResultOptimizations(tc, "1: package main\n\n(2200 lines total)\n")

	if a.Stats.Optimizations.OutlineFirstCount != 1 {
		t.Fatalf("OutlineFirstCount = %d, want 1", a.Stats.Optimizations.OutlineFirstCount)
	}
}

func TestRecordToolResultOptimizations_OutlineFooterLegacyFormat(t *testing.T) {
	a := &Agent{Stats: NewSessionStats("test")}
	tc := &tools.ToolCall{Tool: "read_file"}

	a.recordToolResultOptimizations(tc, "1: package main\n\n(2200 lines total. For specific sections: paths=[\"/tmp/file.go:start-end\"])\n")

	if a.Stats.Optimizations.OutlineFirstCount != 1 {
		t.Fatalf("OutlineFirstCount = %d, want 1", a.Stats.Optimizations.OutlineFirstCount)
	}
}

func TestRecordToolObservability_SingleRead(t *testing.T) {
	a := &Agent{Stats: NewSessionStats("test")}
	a.recordToolObservability("read_file", map[string]any{
		"path": "a.go",
	}, nil, "file contents...")

	if a.Stats.ToolObs.ReadFileBatchCalls != 0 {
		t.Errorf("ReadFileBatchCalls = %d, want 0", a.Stats.ToolObs.ReadFileBatchCalls)
	}
	if a.Stats.ToolObs.ReadFileTargetCalls != 0 {
		t.Errorf("ReadFileTargetCalls = %d, want 0", a.Stats.ToolObs.ReadFileTargetCalls)
	}
}

func TestRecordToolObservability_ReadFileTargets(t *testing.T) {
	a := &Agent{Stats: NewSessionStats("test")}
	a.recordToolObservability("read_file", map[string]any{
		"targets": "[L1,L2]",
	}, nil, "file contents...")

	if a.Stats.ToolObs.ReadFileTargetCalls != 1 {
		t.Errorf("ReadFileTargetCalls = %d, want 1", a.Stats.ToolObs.ReadFileTargetCalls)
	}
	if a.Stats.ToolObs.ReadFileBatchCalls != 0 {
		t.Errorf("ReadFileBatchCalls = %d, want 0 for targets-based read", a.Stats.ToolObs.ReadFileBatchCalls)
	}
}

func TestRecordToolObservability_EmptyPathError(t *testing.T) {
	a := &Agent{Stats: NewSessionStats("test")}
	a.recordToolObservability("read_file", map[string]any{
		"paths": []any{},
	}, nil, "Error: paths is empty")

	if a.Stats.ToolObs.ReadFileEmptyPathsErrors != 1 {
		t.Errorf("ReadFileEmptyPathsErrors = %d, want 1", a.Stats.ToolObs.ReadFileEmptyPathsErrors)
	}
}

func TestRecordToolObservability_NotEmptyPathError(t *testing.T) {
	a := &Agent{Stats: NewSessionStats("test")}
	a.recordToolObservability("read_file", map[string]any{
		"path": "a.go",
	}, nil, "Error: file not found")

	if a.Stats.ToolObs.ReadFileEmptyPathsErrors != 0 {
		t.Errorf("ReadFileEmptyPathsErrors = %d, want 0 for non-empty-path error", a.Stats.ToolObs.ReadFileEmptyPathsErrors)
	}
}

func TestRecordToolObservability_MultiPatternSearch(t *testing.T) {
	a := &Agent{Stats: NewSessionStats("test")}
	a.recordToolObservability("search_code", map[string]any{
		"pattern": "func_a,func_b",
	}, nil, "Found 5 matches...")

	if a.Stats.ToolObs.SearchCodeExplicitMultiCalls != 1 {
		t.Errorf("SearchCodeExplicitMultiCalls = %d, want 1", a.Stats.ToolObs.SearchCodeExplicitMultiCalls)
	}
	if a.Stats.ToolObs.SearchCodeMultiPatternCalls != 1 {
		t.Errorf("SearchCodeMultiPatternCalls = %d, want 1", a.Stats.ToolObs.SearchCodeMultiPatternCalls)
	}
	if a.Stats.ToolObs.SearchCodeImpactCalls != 0 {
		t.Errorf("SearchCodeImpactCalls = %d, want 0", a.Stats.ToolObs.SearchCodeImpactCalls)
	}
}

func TestRecordToolObservability_SinglePatternSearch(t *testing.T) {
	a := &Agent{Stats: NewSessionStats("test")}
	a.recordToolObservability("search_code", map[string]any{
		"pattern": "funcName",
	}, nil, "Found 3 matches...")

	if a.Stats.ToolObs.SearchCodeMultiPatternCalls != 0 {
		t.Errorf("SearchCodeMultiPatternCalls = %d, want 0", a.Stats.ToolObs.SearchCodeMultiPatternCalls)
	}
	if a.Stats.ToolObs.SearchCodeExplicitMultiCalls != 0 {
		t.Errorf("SearchCodeExplicitMultiCalls = %d, want 0", a.Stats.ToolObs.SearchCodeExplicitMultiCalls)
	}
	if a.Stats.ToolObs.SearchCodeMissedMultiPattern != 0 {
		t.Errorf("SearchCodeMissedMultiPattern = %d, want 0", a.Stats.ToolObs.SearchCodeMissedMultiPattern)
	}
}

func TestRecordToolObservability_ImpactIntentCountsSeparately(t *testing.T) {
	a := &Agent{Stats: NewSessionStats("test")}
	a.recordToolObservability("search_code", map[string]any{
		"pattern": "NewAgent",
		"intent":  "impact",
	}, nil, "Found 5 matches...")

	if a.Stats.ToolObs.SearchCodeImpactCalls != 1 {
		t.Errorf("SearchCodeImpactCalls = %d, want 1 for intent=impact", a.Stats.ToolObs.SearchCodeImpactCalls)
	}
	if a.Stats.ToolObs.SearchCodeExplicitMultiCalls != 0 {
		t.Errorf("SearchCodeExplicitMultiCalls = %d, want 0 for intent=impact", a.Stats.ToolObs.SearchCodeExplicitMultiCalls)
	}
	if a.Stats.ToolObs.SearchCodeMultiPatternCalls != 0 {
		t.Errorf("SearchCodeMultiPatternCalls = %d, want 0 for intent=impact", a.Stats.ToolObs.SearchCodeMultiPatternCalls)
	}
	if a.Stats.ToolObs.SearchCodeMissedMultiPattern != 0 {
		t.Errorf("SearchCodeMissedMultiPattern = %d, want 0 for intent=impact", a.Stats.ToolObs.SearchCodeMissedMultiPattern)
	}
}

func TestRecordToolObservability_ImpactIntentDoesNotDoubleCountExplicitMulti(t *testing.T) {
	a := &Agent{Stats: NewSessionStats("test")}
	a.recordToolObservability("search_code", map[string]any{
		"pattern": "foo callers,foo references",
		"intent":  "impact",
	}, nil, "Found 5 matches...")

	if a.Stats.ToolObs.SearchCodeImpactCalls != 1 {
		t.Errorf("SearchCodeImpactCalls = %d, want 1", a.Stats.ToolObs.SearchCodeImpactCalls)
	}
	if a.Stats.ToolObs.SearchCodeExplicitMultiCalls != 0 {
		t.Errorf("SearchCodeExplicitMultiCalls = %d, want 0 when intent=impact", a.Stats.ToolObs.SearchCodeExplicitMultiCalls)
	}
	if a.Stats.ToolObs.SearchCodeMultiPatternCalls != 0 {
		t.Errorf("SearchCodeMultiPatternCalls = %d, want 0 when intent=impact", a.Stats.ToolObs.SearchCodeMultiPatternCalls)
	}
}

// ── recordToolObservability (XML rescue path: RawArgs なし, Args のみ) ──

func TestRecordToolObservability_XMLRescue_BatchRead(t *testing.T) {
	a := &Agent{Stats: NewSessionStats("test")}
	// XML rescue: RawArgs=nil, Args に JSON 文字列が入る
	a.recordToolObservability("read_file", nil, map[string]string{
		"paths": `["a.go","b.go","c.go"]`,
	}, "file contents...")

	if a.Stats.ToolObs.ReadFileBatchCalls != 1 {
		t.Errorf("XML rescue: ReadFileBatchCalls = %d, want 1", a.Stats.ToolObs.ReadFileBatchCalls)
	}
}

func TestRecordToolObservability_XMLRescue_BatchReadWithWhitespace(t *testing.T) {
	a := &Agent{Stats: NewSessionStats("test")}
	// XML rescue: <paths>\n["a.go","b.go"]\n</paths> のように前後に改行が入る
	a.recordToolObservability("read_file", nil, map[string]string{
		"paths": "\n[\"a.go\",\"b.go\"]\n",
	}, "file contents...")

	if a.Stats.ToolObs.ReadFileBatchCalls != 1 {
		t.Errorf("XML rescue with whitespace: ReadFileBatchCalls = %d, want 1", a.Stats.ToolObs.ReadFileBatchCalls)
	}
}

func TestRecordToolObservability_XMLRescue_ReadFileTargets(t *testing.T) {
	a := &Agent{Stats: NewSessionStats("test")}
	a.recordToolObservability("read_file", nil, map[string]string{
		"targets": "\n[L1,L2]\n",
	}, "file contents...")

	if a.Stats.ToolObs.ReadFileTargetCalls != 1 {
		t.Errorf("XML rescue: ReadFileTargetCalls = %d, want 1", a.Stats.ToolObs.ReadFileTargetCalls)
	}
	if a.Stats.ToolObs.ReadFileBatchCalls != 0 {
		t.Errorf("XML rescue: ReadFileBatchCalls = %d, want 0 for targets-based read", a.Stats.ToolObs.ReadFileBatchCalls)
	}
}

func TestRecordToolObservability_XMLRescue_MultiPattern(t *testing.T) {
	a := &Agent{Stats: NewSessionStats("test")}
	a.recordToolObservability("search_code", nil, map[string]string{
		"pattern": "func_a,func_b",
	}, "Found 5 matches...")

	if a.Stats.ToolObs.SearchCodeExplicitMultiCalls != 1 {
		t.Errorf("XML rescue: SearchCodeExplicitMultiCalls = %d, want 1", a.Stats.ToolObs.SearchCodeExplicitMultiCalls)
	}
	if a.Stats.ToolObs.SearchCodeMultiPatternCalls != 1 {
		t.Errorf("XML rescue: SearchCodeMultiPatternCalls = %d, want 1", a.Stats.ToolObs.SearchCodeMultiPatternCalls)
	}
	if a.Stats.ToolObs.SearchCodeImpactCalls != 0 {
		t.Errorf("XML rescue: SearchCodeImpactCalls = %d, want 0", a.Stats.ToolObs.SearchCodeImpactCalls)
	}
}

func TestRecordToolObservability_XMLRescue_SinglePattern(t *testing.T) {
	a := &Agent{Stats: NewSessionStats("test")}
	a.recordToolObservability("search_code", nil, map[string]string{
		"pattern": "funcName",
	}, "Found 3 matches...")

	if a.Stats.ToolObs.SearchCodeMultiPatternCalls != 0 {
		t.Errorf("XML rescue: SearchCodeMultiPatternCalls = %d, want 0", a.Stats.ToolObs.SearchCodeMultiPatternCalls)
	}
	if a.Stats.ToolObs.SearchCodeExplicitMultiCalls != 0 {
		t.Errorf("XML rescue: SearchCodeExplicitMultiCalls = %d, want 0", a.Stats.ToolObs.SearchCodeExplicitMultiCalls)
	}
}

func TestRecordToolObservability_XMLRescue_ImpactIntentCountsSeparately(t *testing.T) {
	a := &Agent{Stats: NewSessionStats("test")}
	a.recordToolObservability("search_code", nil, map[string]string{
		"pattern": "NewAgent",
		"intent":  "impact",
	}, "Found 5 matches...")

	if a.Stats.ToolObs.SearchCodeImpactCalls != 1 {
		t.Errorf("XML rescue: SearchCodeImpactCalls = %d, want 1 for intent=impact", a.Stats.ToolObs.SearchCodeImpactCalls)
	}
	if a.Stats.ToolObs.SearchCodeExplicitMultiCalls != 0 {
		t.Errorf("XML rescue: SearchCodeExplicitMultiCalls = %d, want 0 for intent=impact", a.Stats.ToolObs.SearchCodeExplicitMultiCalls)
	}
	if a.Stats.ToolObs.SearchCodeMultiPatternCalls != 0 {
		t.Errorf("XML rescue: SearchCodeMultiPatternCalls = %d, want 0 for intent=impact", a.Stats.ToolObs.SearchCodeMultiPatternCalls)
	}
	if a.Stats.ToolObs.SearchCodeMissedMultiPattern != 0 {
		t.Errorf("XML rescue: SearchCodeMissedMultiPattern = %d, want 0 for intent=impact", a.Stats.ToolObs.SearchCodeMissedMultiPattern)
	}
}

func TestRecordToolObservability_XMLRescue_ImpactIntentDoesNotDoubleCountExplicitMulti(t *testing.T) {
	a := &Agent{Stats: NewSessionStats("test")}
	a.recordToolObservability("search_code", nil, map[string]string{
		"pattern": "foo callers,foo references",
		"intent":  "impact",
	}, "Found 5 matches...")

	if a.Stats.ToolObs.SearchCodeImpactCalls != 1 {
		t.Errorf("XML rescue: SearchCodeImpactCalls = %d, want 1", a.Stats.ToolObs.SearchCodeImpactCalls)
	}
	if a.Stats.ToolObs.SearchCodeExplicitMultiCalls != 0 {
		t.Errorf("XML rescue: SearchCodeExplicitMultiCalls = %d, want 0 when intent=impact", a.Stats.ToolObs.SearchCodeExplicitMultiCalls)
	}
	if a.Stats.ToolObs.SearchCodeMultiPatternCalls != 0 {
		t.Errorf("XML rescue: SearchCodeMultiPatternCalls = %d, want 0 when intent=impact", a.Stats.ToolObs.SearchCodeMultiPatternCalls)
	}
}

func TestRecordToolObservability_SearchCodeMissedMultiPattern(t *testing.T) {
	a := &Agent{Stats: NewSessionStats("test")}
	a.resetSearchCodeTurnObservability()

	a.recordToolObservability("search_code", map[string]any{
		"pattern": "foo definition",
		"path":    ".",
		"mode":    "auto",
	}, nil, "Found 1 match...")
	a.recordToolObservability("search_code", map[string]any{
		"pattern": "foo callers",
		"path":    ".",
		"mode":    "auto",
	}, nil, "Found 2 matches...")

	if a.Stats.ToolObs.SearchCodeMissedMultiPattern != 1 {
		t.Errorf("SearchCodeMissedMultiPattern = %d, want 1", a.Stats.ToolObs.SearchCodeMissedMultiPattern)
	}
}

func TestRecordToolObservability_SearchCodeMissedMultiPattern_DoesNotCountForMultiPattern(t *testing.T) {
	a := &Agent{Stats: NewSessionStats("test")}
	a.resetSearchCodeTurnObservability()

	a.recordToolObservability("search_code", map[string]any{
		"pattern": "foo definition,foo callers",
		"path":    ".",
	}, nil, "Found 3 matches...")

	if a.Stats.ToolObs.SearchCodeMissedMultiPattern != 0 {
		t.Errorf("SearchCodeMissedMultiPattern = %d, want 0", a.Stats.ToolObs.SearchCodeMissedMultiPattern)
	}
}

func TestRecordToolObservability_SearchCodeMissedMultiPattern_CountsOncePerFamilyPerTurn(t *testing.T) {
	a := &Agent{Stats: NewSessionStats("test")}
	a.resetSearchCodeTurnObservability()

	a.recordToolObservability("search_code", map[string]any{
		"pattern": "foo definition",
		"path":    ".",
	}, nil, "Found 1 match...")
	a.recordToolObservability("search_code", map[string]any{
		"pattern": "foo callers",
		"path":    ".",
	}, nil, "Found 2 matches...")
	a.recordToolObservability("search_code", map[string]any{
		"pattern": "foo references",
		"path":    ".",
	}, nil, "Found 3 matches...")

	if a.Stats.ToolObs.SearchCodeMissedMultiPattern != 1 {
		t.Errorf("SearchCodeMissedMultiPattern = %d, want 1", a.Stats.ToolObs.SearchCodeMissedMultiPattern)
	}
}

func TestRecordToolObservability_SearchCodeMissedMultiPattern_ResetsOnNewUserRequest(t *testing.T) {
	a := &Agent{Stats: NewSessionStats("test")}
	a.resetSearchCodeTurnObservability()

	a.recordToolObservability("search_code", map[string]any{
		"pattern": "foo definition",
		"path":    ".",
	}, nil, "Found 1 match...")
	a.recordToolObservability("search_code", map[string]any{
		"pattern": "foo callers",
		"path":    ".",
	}, nil, "Found 2 matches...")
	a.resetSearchCodeTurnObservability()
	a.recordToolObservability("search_code", map[string]any{
		"pattern": "foo definition",
		"path":    ".",
	}, nil, "Found 1 match...")
	a.recordToolObservability("search_code", map[string]any{
		"pattern": "foo callers",
		"path":    ".",
	}, nil, "Found 2 matches...")

	if a.Stats.ToolObs.SearchCodeMissedMultiPattern != 2 {
		t.Errorf("SearchCodeMissedMultiPattern = %d, want 2", a.Stats.ToolObs.SearchCodeMissedMultiPattern)
	}
}

func TestPrintToolObservabilitySection(t *testing.T) {
	stats := NewSessionStats("test")
	stats.ToolObs.ReadFileBatchCalls = 12
	stats.ToolObs.ReadFileTargetCalls = 6
	stats.ToolObs.SearchCodeImpactCalls = 8
	stats.ToolObs.SearchCodeExplicitMultiCalls = 7
	stats.ToolObs.SearchCodeMultiPatternCalls = 7
	stats.ToolObs.SearchCodeMissedMultiPattern = 9
	stats.ToolObs.SearchCodeBatchMerges = 5
	stats.ToolObs.ReadFileBatchMerges = 4
	stats.ToolObs.ReadFileEmptyPathsErrors = 3

	var buf bytes.Buffer
	printToolObservabilitySection(&buf, stats)
	output := buf.String()

	// Tool Selection セクション
	if !strings.Contains(output, "Tool Selection") {
		t.Error("should contain Tool Selection header")
	}
	if !strings.Contains(output, "read_file(batch)") {
		t.Error("should contain read_file(batch) row")
	}
	if !strings.Contains(output, "12") {
		t.Error("should contain batch count 12")
	}
	if !strings.Contains(output, "search_code(multi)") {
		t.Error("should contain search_code(multi) row")
	}
	if !strings.Contains(output, "7") {
		t.Error("should contain multi-pattern count 7")
	}
	if !strings.Contains(output, "search_code(missed multi)") {
		t.Error("should contain search_code(missed multi) row")
	}
	if !strings.Contains(output, "9") {
		t.Error("should contain missed multi count 9")
	}
	if strings.Contains(output, "search_code(batch merge)") {
		t.Error("should not contain search_code(batch merge) row")
	}
	if strings.Contains(output, "read_file(batch merge)") {
		t.Error("should not contain read_file(batch merge) row")
	}
	if strings.Contains(output, "read_file empty-path errors") {
		t.Error("should not contain read_file empty-path errors row")
	}
	if !strings.Contains(output, "Exploration") {
		t.Error("should contain Exploration header")
	}
	if !strings.Contains(output, "search_code(impact)") {
		t.Error("should contain search_code(impact) row")
	}
	if !strings.Contains(output, "8") {
		t.Error("should contain impact count 8")
	}
	if !strings.Contains(output, "search_code(explicit multi)") {
		t.Error("should contain search_code(explicit multi) row")
	}
	if !strings.Contains(output, "read_file(targets)") {
		t.Error("should contain read_file(targets) row")
	}
	if !strings.Contains(output, "6") {
		t.Error("should contain read_file(targets) count 6")
	}
	if !strings.Contains(output, "search_code(batch merges)") {
		t.Error("should contain search_code(batch merges) row")
	}
	if !strings.Contains(output, "read_file(batch merges)") {
		t.Error("should contain read_file(batch merges) row")
	}
}

func TestPrintToolObservabilitySection_AllZero(t *testing.T) {
	stats := NewSessionStats("test")

	var buf bytes.Buffer
	printToolObservabilitySection(&buf, stats)
	output := buf.String()

	// 全て0でもセクションは表示される
	if !strings.Contains(output, "Tool Selection") {
		t.Error("should show Tool Selection even with all zeros")
	}
	if !strings.Contains(output, "read_file(batch)") {
		t.Error("should contain read_file(batch) row")
	}
	if !strings.Contains(output, "search_code(multi)") {
		t.Error("should contain search_code(multi) row")
	}
	if !strings.Contains(output, "search_code(missed multi)") {
		t.Error("should contain search_code(missed multi) row")
	}
	if !strings.Contains(output, "Exploration") {
		t.Error("should contain Exploration header")
	}
	if !strings.Contains(output, "search_code(impact)") {
		t.Error("should contain search_code(impact) row")
	}
	if !strings.Contains(output, "search_code(explicit multi)") {
		t.Error("should contain search_code(explicit multi) row")
	}
	if !strings.Contains(output, "read_file(targets)") {
		t.Error("should contain read_file(targets) row")
	}
	if !strings.Contains(output, "search_code(batch merges)") {
		t.Error("should contain search_code(batch merges) row")
	}
	if !strings.Contains(output, "read_file(batch merges)") {
		t.Error("should contain read_file(batch merges) row")
	}
	if strings.Contains(output, "read_file empty-path errors") {
		t.Error("should not contain read_file empty-path errors row")
	}
}
