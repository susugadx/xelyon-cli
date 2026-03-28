package search

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanSearchRoute_UnknownLanguageDoesNotGoLane(t *testing.T) {
	dir := t.TempDir()
	trace := planSearchRoute("Close", SearchOptions{Mode: string(SearchModeAuto), Path: dir, FilePattern: ""})
	if trace.Language != "" {
		t.Fatalf("language = %q, want unknown/empty", trace.Language)
	}
	if trace.InitialLane != searchLaneLiteral {
		t.Fatalf("initial lane = %q, want literal for unknown language", trace.InitialLane)
	}
}

func TestPlanSearchRoute_UnknownLanguageRegexLikeDoesNotGoRescue(t *testing.T) {
	dir := t.TempDir()
	trace := planSearchRoute(`\.Close\(\)`, SearchOptions{Mode: string(SearchModeAuto), Path: dir, FilePattern: ""})
	if trace.Language != "" {
		t.Fatalf("language = %q, want unknown/empty", trace.Language)
	}
	if trace.InitialLane != searchLaneRegex {
		t.Fatalf("initial lane = %q, want regex for unknown language", trace.InitialLane)
	}
	if trace.SymbolRescue || trace.SymbolQuery != "" {
		t.Fatalf("unknown language should not trigger Go rescue, got rescue=%v query=%q", trace.SymbolRescue, trace.SymbolQuery)
	}
}

func TestPlanSearchRoute_GoModulePathUsesGoLane(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	trace := planSearchRoute("Close", SearchOptions{Mode: string(SearchModeAuto), Path: dir, FilePattern: ""})
	if trace.Language != "go" {
		t.Fatalf("language = %q, want go", trace.Language)
	}
	if trace.InitialLane != searchLaneSymbol {
		t.Fatalf("initial lane = %q, want symbol", trace.InitialLane)
	}
}

func TestPlanSearchRoute_AutoSymbolLaneForGoIdentifier(t *testing.T) {
	trace := planSearchRoute("Close", SearchOptions{Mode: string(SearchModeAuto), FileType: "go"})
	if trace.InitialLane != searchLaneSymbol {
		t.Fatalf("initial lane = %q, want symbol", trace.InitialLane)
	}
	if trace.SymbolQuery != "Close" {
		t.Fatalf("symbol query = %q, want Close", trace.SymbolQuery)
	}
}

func TestPlanSearchRoute_AutoSymbolLaneForGoDottedSymbol(t *testing.T) {
	trace := planSearchRoute("(*Agent).Close", SearchOptions{Mode: string(SearchModeAuto), FileType: "go"})
	if trace.InitialLane != searchLaneSymbol {
		t.Fatalf("initial lane = %q, want symbol", trace.InitialLane)
	}
	if trace.SymbolQuery != "(*Agent).Close" {
		t.Fatalf("symbol query = %q, want (*Agent).Close", trace.SymbolQuery)
	}
}

func TestPlanSearchRoute_AutoGoRescueForRegexLikeSelector(t *testing.T) {
	trace := planSearchRoute(`\.Close\(\)`, SearchOptions{Mode: string(SearchModeAuto), FileType: "go"})
	if trace.InitialLane != searchLaneSymbol {
		t.Fatalf("initial lane = %q, want symbol", trace.InitialLane)
	}
	if trace.Decision != "go-rescue" {
		t.Fatalf("decision = %q, want go-rescue", trace.Decision)
	}
	if !trace.SymbolRescue {
		t.Fatal("expected symbol rescue to be enabled")
	}
	if trace.SymbolQuery != "Close" {
		t.Fatalf("symbol query = %q, want Close", trace.SymbolQuery)
	}
	if trace.FallbackLane != searchLaneRegex {
		t.Fatalf("fallback lane = %q, want regex", trace.FallbackLane)
	}
}

func TestPlanSearchRoute_AutoGoRescueForRegexLikeFuncDecl(t *testing.T) {
	trace := planSearchRoute(`func \(a \*Agent\) Close\(`, SearchOptions{Mode: string(SearchModeAuto), FileType: "go"})
	if trace.InitialLane != searchLaneSymbol {
		t.Fatalf("initial lane = %q, want symbol", trace.InitialLane)
	}
	if !trace.SymbolRescue {
		t.Fatal("expected symbol rescue to be enabled")
	}
	if trace.SymbolQuery != "(*Agent).Close" && trace.SymbolQuery != "Close" {
		t.Fatalf("symbol query = %q, want (*Agent).Close or Close", trace.SymbolQuery)
	}
}

func TestPlanSearchRoute_ExplicitRegexSkipsRescue(t *testing.T) {
	trace := planSearchRoute(`\.Close\(\)`, SearchOptions{Mode: string(SearchModeRegex), FileType: "go"})
	if trace.InitialLane != searchLaneRegex {
		t.Fatalf("initial lane = %q, want regex", trace.InitialLane)
	}
	if trace.SymbolRescue || trace.SymbolQuery != "" {
		t.Fatalf("explicit regex should not prepare symbol rescue, got rescue=%v query=%q", trace.SymbolRescue, trace.SymbolQuery)
	}
}

func TestPlanSearchRoute_StrongRegexMetaUsesRegexLane(t *testing.T) {
	trace := planSearchRoute(`Set[A-Z]Tools`, SearchOptions{Mode: string(SearchModeAuto), FileType: "go"})
	if trace.InitialLane != searchLaneRegex {
		t.Fatalf("initial lane = %q, want regex", trace.InitialLane)
	}
}

func TestPlanSearchRoute_PlainTextUsesLiteralLane(t *testing.T) {
	trace := planSearchRoute("close connection", SearchOptions{Mode: string(SearchModeAuto), FileType: "go"})
	if trace.InitialLane != searchLaneLiteral {
		t.Fatalf("initial lane = %q, want literal", trace.InitialLane)
	}
	if trace.Decision != "text" {
		t.Fatalf("decision = %q, want text", trace.Decision)
	}
}

func TestPlanSearchRoute_ExplicitSymbolUsesSymbolLaneEvenIfUnknown(t *testing.T) {
	trace := planSearchRoute("Close", SearchOptions{Mode: string(SearchModeSymbol), Path: ".", FilePattern: ""})
	if trace.InitialLane != searchLaneSymbol {
		t.Fatalf("initial lane = %q, want symbol", trace.InitialLane)
	}
	if trace.SymbolQuery != "Close" {
		t.Fatalf("symbol query = %q, want Close", trace.SymbolQuery)
	}
}

func TestExecuteSinglePatternWithTrace_SymbolMissFallsBackToRegex(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file := filepath.Join(dir, "agent.go")
	if err := os.WriteFile(file, []byte("package main\n\nfunc run() {\n\tclient.Close()\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, trace := executeSinglePatternWithTrace(nil, `Close\(`, SearchOptions{
		Pattern:  `Close\(`,
		Mode:     string(SearchModeAuto),
		Path:     dir,
		FileType: "go",
	})

	if !trace.SymbolAttempted {
		t.Fatal("expected symbol attempt before fallback")
	}
	if !trace.FallbackUsed {
		t.Fatal("expected fallback after symbol miss")
	}
	if trace.FinalLane != searchLaneRegex {
		t.Fatalf("final lane = %q, want regex", trace.FinalLane)
	}
	if !strings.Contains(result, lineRangeHint) {
		t.Fatalf("expected regex fallback to return text-search style output, got:\n%s", result)
	}
}

func TestExecuteSinglePatternWithTrace_SymbolMissFallsBackToLiteral(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file := filepath.Join(dir, "agent.go")
	if err := os.WriteFile(file, []byte("package main\n\n// Close path is tracked here.\nvar closePath = \"Close\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, trace := executeSinglePatternWithTrace(nil, "Close", SearchOptions{
		Pattern:  "Close",
		Mode:     string(SearchModeAuto),
		Path:     dir,
		FileType: "go",
	})

	if !trace.SymbolAttempted {
		t.Fatal("expected symbol attempt before fallback")
	}
	if !trace.FallbackUsed {
		t.Fatal("expected literal fallback after symbol miss")
	}
	if trace.FinalLane != searchLaneLiteral {
		t.Fatalf("final lane = %q, want literal", trace.FinalLane)
	}
	if !strings.Contains(result, lineRangeHint) {
		t.Fatalf("expected text-search style fallback result, got:\n%s", result)
	}
}

func TestBuildSearchCacheKeyWithRoute_DiffersByModeAndRoute(t *testing.T) {
	optsAuto, ok := normalizeSearchOptions(SearchOptions{Mode: string(SearchModeAuto), Path: ".", FileType: "go"})
	if !ok {
		t.Fatal("expected auto options to normalize")
	}
	optsRegex, ok := normalizeSearchOptions(SearchOptions{Mode: string(SearchModeRegex), Path: ".", FileType: "go"})
	if !ok {
		t.Fatal("expected regex options to normalize")
	}

	autoKey := buildSearchCacheKeyWithRoute(optsAuto, planSearchRoute(`\.Close\(\)`, optsAuto).cacheSignature())
	regexKey := buildSearchCacheKeyWithRoute(optsRegex, planSearchRoute(`\.Close\(\)`, optsRegex).cacheSignature())

	if autoKey == regexKey {
		t.Fatalf("expected cache keys to differ by mode/route, got %q", autoKey)
	}
}

func TestBuildSearchCacheKeyWithRoute_LegacyRegexMatchesModeRegex(t *testing.T) {
	legacyRegex, ok := normalizeSearchOptions(SearchOptions{Path: ".", IsRegex: true, LegacyIsRegexSet: true})
	if !ok {
		t.Fatal("expected legacy regex options to normalize")
	}
	modeRegex, ok := normalizeSearchOptions(SearchOptions{Mode: string(SearchModeRegex), Path: "."})
	if !ok {
		t.Fatal("expected mode regex options to normalize")
	}

	legacyKey := buildSearchCacheKeyWithRoute(legacyRegex, planSearchRoute(`\.Close\(\)`, legacyRegex).cacheSignature())
	modeKey := buildSearchCacheKeyWithRoute(modeRegex, planSearchRoute(`\.Close\(\)`, modeRegex).cacheSignature())
	if legacyKey != modeKey {
		t.Fatalf("expected legacy regex and mode=regex to share cache key, got %q vs %q", legacyKey, modeKey)
	}
}
