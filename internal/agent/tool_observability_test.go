package agent

import (
	"bytes"
	"fmt"
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

func TestRecordToolObservability_SingleRead(t *testing.T) {
	a := &Agent{Stats: NewSessionStats("test")}
	a.recordToolObservability("read_file", map[string]any{
		"path": "a.go",
	}, nil, "file contents...")

	if a.Stats.ToolObs.ReadFileBatchCalls != 0 {
		t.Errorf("ReadFileBatchCalls = %d, want 0", a.Stats.ToolObs.ReadFileBatchCalls)
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

	if a.Stats.ToolObs.SearchCodeMultiPatternCalls != 1 {
		t.Errorf("SearchCodeMultiPatternCalls = %d, want 1", a.Stats.ToolObs.SearchCodeMultiPatternCalls)
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

func TestRecordToolObservability_XMLRescue_MultiPattern(t *testing.T) {
	a := &Agent{Stats: NewSessionStats("test")}
	a.recordToolObservability("search_code", nil, map[string]string{
		"pattern": "func_a,func_b",
	}, "Found 5 matches...")

	if a.Stats.ToolObs.SearchCodeMultiPatternCalls != 1 {
		t.Errorf("XML rescue: SearchCodeMultiPatternCalls = %d, want 1", a.Stats.ToolObs.SearchCodeMultiPatternCalls)
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
}

// ── recordCompaction ──

func TestRecordCompaction_Shortened(t *testing.T) {
	a := &Agent{Stats: NewSessionStats("test")}
	a.recordCompaction("read_file", 1000, 500)

	obs := a.Stats.ToolObs
	if obs.CompactedToolResults != 1 {
		t.Errorf("CompactedToolResults = %d, want 1", obs.CompactedToolResults)
	}
	if obs.CompactedBytesSaved != 500 {
		t.Errorf("CompactedBytesSaved = %d, want 500", obs.CompactedBytesSaved)
	}
	if obs.CompactedByTool["read_file"] != 1 {
		t.Errorf("CompactedByTool[read_file] = %d, want 1", obs.CompactedByTool["read_file"])
	}
}

func TestRecordCompaction_NoChange(t *testing.T) {
	a := &Agent{Stats: NewSessionStats("test")}
	a.recordCompaction("read_file", 500, 500)

	obs := a.Stats.ToolObs
	if obs.CompactedToolResults != 0 {
		t.Errorf("CompactedToolResults = %d, want 0 when no change", obs.CompactedToolResults)
	}
	if obs.CompactedBytesSaved != 0 {
		t.Errorf("CompactedBytesSaved = %d, want 0 when no change", obs.CompactedBytesSaved)
	}
}

func TestRecordCompaction_Longer(t *testing.T) {
	// guidance 追加等で長くなるケースはカウントしない
	a := &Agent{Stats: NewSessionStats("test")}
	a.recordCompaction("read_file", 500, 600)

	obs := a.Stats.ToolObs
	if obs.CompactedToolResults != 0 {
		t.Errorf("CompactedToolResults = %d, want 0 when result got longer", obs.CompactedToolResults)
	}
}

func TestRecordCompaction_MultipleTools(t *testing.T) {
	a := &Agent{Stats: NewSessionStats("test")}
	a.recordCompaction("read_file", 1000, 500)
	a.recordCompaction("search_code", 2000, 800)
	a.recordCompaction("read_file", 800, 400)

	obs := a.Stats.ToolObs
	if obs.CompactedToolResults != 3 {
		t.Errorf("CompactedToolResults = %d, want 3", obs.CompactedToolResults)
	}
	if obs.CompactedBytesSaved != 500+1200+400 {
		t.Errorf("CompactedBytesSaved = %d, want %d", obs.CompactedBytesSaved, 500+1200+400)
	}
	if obs.CompactedByTool["read_file"] != 2 {
		t.Errorf("CompactedByTool[read_file] = %d, want 2", obs.CompactedByTool["read_file"])
	}
	if obs.CompactedByTool["search_code"] != 1 {
		t.Errorf("CompactedByTool[search_code] = %d, want 1", obs.CompactedByTool["search_code"])
	}
}

// ── compactToolResult による統合テスト ──

func TestCompactToolResult_RecordsCompaction(t *testing.T) {
	a := &Agent{Stats: NewSessionStats("test")}

	// 長い read_file 結果（compactReadFileMinLines を超える）を生成
	var lines []string
	for i := 0; i < 200; i++ {
		lines = append(lines, fmt.Sprintf("line %d: some code here", i))
	}
	longResult := strings.Join(lines, "\n")

	tc := &tools.ToolCall{
		Tool: "read_file",
		Args: map[string]string{"path": "/tmp/test.go"},
	}
	compacted := a.compactToolResult(tc, longResult)

	if len(compacted) >= len(longResult) {
		t.Error("compactToolResult should shorten long read_file result")
	}
	if a.Stats.ToolObs.CompactedToolResults != 1 {
		t.Errorf("CompactedToolResults = %d, want 1", a.Stats.ToolObs.CompactedToolResults)
	}
	if a.Stats.ToolObs.CompactedBytesSaved <= 0 {
		t.Error("CompactedBytesSaved should be > 0")
	}
	if a.Stats.ToolObs.CompactedByTool["read_file"] != 1 {
		t.Errorf("CompactedByTool[read_file] = %d, want 1", a.Stats.ToolObs.CompactedByTool["read_file"])
	}
}

func TestCompactToolResult_NoCompactionForShort(t *testing.T) {
	a := &Agent{Stats: NewSessionStats("test")}

	tc := &tools.ToolCall{
		Tool: "read_file",
		Args: map[string]string{"path": "/tmp/test.go"},
	}
	a.compactToolResult(tc, "short content")

	if a.Stats.ToolObs.CompactedToolResults != 0 {
		t.Errorf("CompactedToolResults = %d, want 0 for short content", a.Stats.ToolObs.CompactedToolResults)
	}
}

// ── /status 表示 ──

func TestFormatCompactedByTool(t *testing.T) {
	m := map[string]int{
		"read_file":   7,
		"search_code": 6,
		"list_dir":    2,
	}
	got := formatCompactedByTool(m)
	// 降順ソート
	if !strings.Contains(got, "read_file=7") {
		t.Errorf("should contain read_file=7, got: %s", got)
	}
	if !strings.Contains(got, "search_code=6") {
		t.Errorf("should contain search_code=6, got: %s", got)
	}
	if !strings.Contains(got, "list_dir=2") {
		t.Errorf("should contain list_dir=2, got: %s", got)
	}
}

func TestPrintToolObservabilitySection(t *testing.T) {
	stats := NewSessionStats("test")
	stats.ToolObs.ReadFileBatchCalls = 12
	stats.ToolObs.SearchCodeMultiPatternCalls = 7
	stats.ToolObs.ReadFileEmptyPathsErrors = 3
	stats.ToolObs.CompactedToolResults = 18
	stats.ToolObs.CompactedBytesSaved = 12450
	stats.ToolObs.CompactedByTool["read_file"] = 7
	stats.ToolObs.CompactedByTool["search_code"] = 6

	var buf bytes.Buffer
	printToolObservabilitySection(&buf, stats)
	output := buf.String()

	// Tool Selection セクション
	if !strings.Contains(output, "Tool Selection") {
		t.Error("should contain Tool Selection header")
	}
	if !strings.Contains(output, "12") {
		t.Error("should contain batch count 12")
	}
	if !strings.Contains(output, "7") {
		t.Error("should contain multi-pattern count 7")
	}
	if !strings.Contains(output, "3") {
		t.Error("should contain empty-path error count 3")
	}

	// Compaction セクション
	if !strings.Contains(output, "Compaction") {
		t.Error("should contain Compaction header")
	}
	if !strings.Contains(output, "18") {
		t.Error("should contain compacted results count 18")
	}
	if !strings.Contains(output, "12,450") {
		t.Error("should contain bytes saved 12,450")
	}
	if !strings.Contains(output, "read_file=7") {
		t.Error("should contain by-tool read_file=7")
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
	if !strings.Contains(output, "Compaction") {
		t.Error("should show Compaction even with all zeros")
	}
	// by tool は0件なら表示しない
	if strings.Contains(output, "by tool") {
		t.Error("should not show by-tool when map is empty")
	}
}
