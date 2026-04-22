package search

import (
	"encoding/json"
	"testing"
)

func TestParseRipgrepJSON(t *testing.T) {
	input := `{"type":"begin","data":{"path":{"text":"src/main.go"}}}
{"type":"context","data":{"path":{"text":"src/main.go"},"line_number":4,"lines":{"text":"func setup() {\n"}}}
{"type":"match","data":{"path":{"text":"src/main.go"},"line_number":5,"lines":{"text":"\ttarget := \"hello\"\n"},"submatches":[{"match":{"text":"target"},"start":1,"end":7}]}}
{"type":"context","data":{"path":{"text":"src/main.go"},"line_number":6,"lines":{"text":"}\n"}}}
{"type":"end","data":{"path":{"text":"src/main.go"},"stats":{"elapsed":{"secs":0}}}}
{"type":"begin","data":{"path":{"text":"src/util.go"}}}
{"type":"match","data":{"path":{"text":"src/util.go"},"line_number":10,"lines":{"text":"var target = 42\n"},"submatches":[{"match":{"text":"target"},"start":4,"end":10}]}}
{"type":"end","data":{"path":{"text":"src/util.go"},"stats":{"elapsed":{"secs":0}}}}
{"type":"summary","data":{"elapsed_total":{"secs":0},"stats":{"searches":1}}}`

	results := parseRipgrepJSON(input, 200)

	if len(results) != 2 {
		t.Fatalf("Expected 2 files, got %d", len(results))
	}

	r1 := results[0]
	if r1.FilePath != "src/main.go" {
		t.Errorf("Expected file path 'src/main.go', got '%s'", r1.FilePath)
	}
	if r1.MatchCount != 1 {
		t.Errorf("Expected 1 match, got %d", r1.MatchCount)
	}
	if len(r1.Matches) != 3 {
		t.Errorf("Expected 3 entries (context+match+context), got %d", len(r1.Matches))
	}
	found := false
	for _, m := range r1.Matches {
		if m.IsMatch && m.LineNum == 5 {
			found = true
			if m.Line == "" {
				t.Errorf("Match line should not be empty")
			}
		}
	}
	if !found {
		t.Error("Expected match at line 5")
	}

	r2 := results[1]
	if r2.FilePath != "src/util.go" {
		t.Errorf("Expected file path 'src/util.go', got '%s'", r2.FilePath)
	}
	if r2.MatchCount != 1 {
		t.Errorf("Expected 1 match, got %d", r2.MatchCount)
	}
}

func TestParseGrepOutput(t *testing.T) {
	input := `src/main.go-4-func setup() {
src/main.go:5:	target := "hello"
src/main.go-6-}
--
src/util.go:10:var target = 42`

	results := parseGrepOutput(input, 200)

	if len(results) != 2 {
		t.Fatalf("Expected 2 files, got %d", len(results))
	}

	r1 := results[0]
	if r1.FilePath != "src/main.go" {
		t.Errorf("Expected file path 'src/main.go', got '%s'", r1.FilePath)
	}
	if r1.MatchCount != 1 {
		t.Errorf("Expected 1 match, got %d", r1.MatchCount)
	}
	if len(r1.Matches) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(r1.Matches))
	}

	matchFound := false
	for _, m := range r1.Matches {
		if m.LineNum == 5 && m.IsMatch {
			matchFound = true
		}
		if m.LineNum == 4 && m.IsMatch {
			t.Error("Line 4 should be context (IsMatch=false)")
		}
	}
	if !matchFound {
		t.Error("Expected match at line 5 with IsMatch=true")
	}

	r2 := results[1]
	if r2.FilePath != "src/util.go" {
		t.Errorf("Expected file path 'src/util.go', got '%s'", r2.FilePath)
	}
	if r2.MatchCount != 1 {
		t.Errorf("Expected 1 match, got %d", r2.MatchCount)
	}
}

func TestSearchResultCollectorAddLine(t *testing.T) {
	collector := newSearchResultCollector(2)

	if stop := collector.addLine(parsedSearchLine{
		FilePath: "src/main.go",
		LineNum:  4,
		Line:     "func setup() {",
		IsMatch:  false,
		Type:     MatchTypeRef,
	}); stop {
		t.Fatal("context line should not stop collection")
	}

	if stop := collector.addLine(parsedSearchLine{
		FilePath: "src/main.go",
		LineNum:  5,
		Line:     "target := \"hello\"",
		IsMatch:  true,
		Type:     MatchTypeAssignment,
	}); stop {
		t.Fatal("first match should not stop collection when maxTotalMatches=2")
	}

	if stop := collector.addLine(parsedSearchLine{
		FilePath: "src/main.go",
		LineNum:  6,
		Line:     "target = \"world\"",
		IsMatch:  true,
		Type:     MatchTypeAssignment,
	}); !stop {
		t.Fatal("second match should stop collection when maxTotalMatches=2")
	}

	results := collector.results()
	if len(results) != 1 {
		t.Fatalf("Expected 1 file result, got %d", len(results))
	}
	if results[0].MatchCount != 2 {
		t.Fatalf("Expected 2 match count, got %d", results[0].MatchCount)
	}
	if len(results[0].Matches) != 3 {
		t.Fatalf("Expected 3 lines (context+2 matches), got %d", len(results[0].Matches))
	}
}

func TestBuildRipgrepParsedLine(t *testing.T) {
	t.Run("current file fallback and trim newline", func(t *testing.T) {
		record := buildRipgrepParsedLine("src/current.go", rgMatchData{
			LineNumber: 10,
			Lines:      rgText{Text: "func run() {}\n"},
		}, true)

		if record.FilePath != "src/current.go" {
			t.Fatalf("Expected current file path fallback, got %q", record.FilePath)
		}
		if record.Line != "func run() {}" {
			t.Fatalf("Expected trimmed line text, got %q", record.Line)
		}
		if record.Type != MatchTypeDefinition {
			t.Fatalf("Expected definition type, got %v", record.Type)
		}
	})

	t.Run("path in data has priority", func(t *testing.T) {
		record := buildRipgrepParsedLine("src/current.go", rgMatchData{
			Path:       rgPath{Text: "./src/other.go"},
			LineNumber: 3,
			Lines:      rgText{Text: "\ttarget := 1\n"},
		}, false)

		if record.FilePath != "src/other.go" {
			t.Fatalf("Expected normalized data path, got %q", record.FilePath)
		}
		if record.Type != MatchTypeRef {
			t.Fatalf("Expected context line type to be ref, got %v", record.Type)
		}
	})
}

func TestDecodeGrepParsedLine(t *testing.T) {
	record, ok := decodeGrepParsedLine("src/main.go:5:\ttarget := \"hello\"")
	if !ok {
		t.Fatal("Expected match line to be parsed")
	}
	if record.FilePath != "src/main.go" {
		t.Fatalf("Expected file path src/main.go, got %q", record.FilePath)
	}
	if !record.IsMatch {
		t.Fatal("Expected colon format to be match line")
	}

	_, ok = decodeGrepParsedLine("invalid-line")
	if ok {
		t.Fatal("Expected invalid line to be rejected")
	}
}

func TestAppendRipgrepEntry(t *testing.T) {
	t.Run("begin updates current file", func(t *testing.T) {
		collector := newSearchResultCollector(10)
		entry := rgJSONLine{
			Type: "begin",
			Data: json.RawMessage(`{"path":{"text":"src/main.go"}}`),
		}

		currentFile, shouldStop := appendRipgrepEntry(entry, "", collector)
		if shouldStop {
			t.Fatal("begin entry should not stop processing")
		}
		if currentFile != "src/main.go" {
			t.Fatalf("expected current file to be updated, got %q", currentFile)
		}
		results := collector.results()
		if len(results) != 1 || results[0].FilePath != "src/main.go" {
			t.Fatalf("expected begin to ensure file entry, got %+v", results)
		}
	})

	t.Run("match can stop when max reached", func(t *testing.T) {
		collector := newSearchResultCollector(1)
		entry := rgJSONLine{
			Type: "match",
			Data: json.RawMessage(`{"path":{"text":"src/main.go"},"line_number":5,"lines":{"text":"func run() {}\n"}}`),
		}

		_, shouldStop := appendRipgrepEntry(entry, "src/main.go", collector)
		if !shouldStop {
			t.Fatal("match entry should stop when maxTotalMatches reached")
		}
	})
}

func TestAppendGrepParsedLine(t *testing.T) {
	t.Run("separator is ignored", func(t *testing.T) {
		collector := newSearchResultCollector(10)
		if shouldStop := appendGrepParsedLine("--", collector); shouldStop {
			t.Fatal("separator line should not stop processing")
		}
		if got := collector.results(); len(got) != 0 {
			t.Fatalf("separator should not add results, got %+v", got)
		}
	})

	t.Run("match can stop when max reached", func(t *testing.T) {
		collector := newSearchResultCollector(1)
		if shouldStop := appendGrepParsedLine("src/main.go:5:func run() {}", collector); !shouldStop {
			t.Fatal("match line should stop when maxTotalMatches reached")
		}
	})
}

func TestScanSearchOutputLines_StopsWhenConsumerReturnsTrue(t *testing.T) {
	var seen []string
	scanSearchOutputLines("a\nb\nc", func(line string) bool {
		seen = append(seen, line)
		return line == "b"
	})
	if len(seen) != 2 {
		t.Fatalf("expected consumer to stop at second line, got %v", seen)
	}
	if seen[0] != "a" || seen[1] != "b" {
		t.Fatalf("unexpected seen lines: %v", seen)
	}
}
