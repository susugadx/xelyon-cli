package search

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchCode_CacheDifferentParams(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "params.go")
	content := "line1\nline2\ntarget_param_check\nline4\nline5\n"
	if err := os.WriteFile(file1, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cache := &testSearchCache{data: make(map[string]string)}

	resultDefault := ExecuteSearchCodeWithCache(cache, SearchOptions{Pattern: "target_param_check", Path: dir, FilePattern: "", FileType: "", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})
	if strings.Contains(resultDefault, "No matches found") {
		t.Fatal("Expected matches with default output mode")
	}

	resultManifest := ExecuteSearchCodeWithCache(cache, SearchOptions{Pattern: "target_param_check", Path: dir, FilePattern: "", FileType: "", CtxLines: 3, TokenBudget: 3000, IsRegex: true, Multiline: false, OutputMode: "manifest"})
	if strings.Contains(resultManifest, "No matches found") {
		t.Fatal("Expected matches with manifest output mode")
	}

	if resultDefault == resultManifest {
		t.Error("Results with different output_mode should differ (cache key should include output_mode)")
	}
}

func TestExecuteSearchCode_MultiplePatterns_PartialMatch(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "partial.go")
	content := "func existing_func() {}\nvar x = 1\n"
	if err := os.WriteFile(file1, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode(SearchOptions{Pattern: "existing_func,nonexistent_xyz_pattern", Path: dir, FilePattern: "*.go", FileType: "", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})

	if !strings.Contains(result, "existing_func") {
		t.Error("Expected existing_func match in result")
	}
	if !strings.Contains(result, "No matches found") {
		t.Error("Expected 'No matches found' for unmatched pattern")
	}
}

func TestSearchCode_ResultRanking(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "rank.go")
	content := "fmt.Println(target)\nfunc target() {}\n"
	if err := os.WriteFile(file1, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode(SearchOptions{Pattern: "target", Path: dir, FilePattern: "*.go", FileType: "", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})

	if strings.Contains(result, "No matches found") {
		t.Error("Expected matches")
	}

	defIdx := strings.Index(result, "[def]")
	refIdx := strings.Index(result, "[ref]")
	if defIdx < 0 {
		t.Error("Expected [def] tag in result")
	}
	if refIdx < 0 {
		t.Error("Expected [ref] tag in result")
	}
	if defIdx >= 0 && refIdx >= 0 && defIdx >= refIdx {
		t.Errorf("Expected [def] before [ref], but [def] at %d, [ref] at %d", defIdx, refIdx)
	}
}

func TestSearchCode_BlockDetection(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "blocks.go")
	content := `package main

func setup() {
	target_var := 1
	fmt.Println(target_var)
}

func process() {
	x := 2
}

func cleanup() {
	target_var := 3
}
`
	if err := os.WriteFile(file1, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode(SearchOptions{Pattern: "target_var", Path: dir, FilePattern: "*.go", FileType: "", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})

	if strings.Contains(result, "No matches found") {
		t.Error("Expected matches")
	}
	if !strings.Contains(result, "── in") {
		t.Errorf("Expected block annotation '── in' in result, got:\n%s", result)
	}

	hasSetup := strings.Contains(result, "func setup")
	hasCleanup := strings.Contains(result, "func cleanup")
	if !hasSetup && !hasCleanup {
		t.Errorf("Expected block annotation for func setup or func cleanup, got:\n%s", result)
	}
}

func TestSearchCode_ManifestMode(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "hello.go")
	if err := os.WriteFile(file1, []byte("package main\n\nfunc hello() {\n\tfmt.Println(\"hello world\")\n}\n\nfunc goodbye() {\n\tfmt.Println(\"hello again\")\n}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode(SearchOptions{
		Pattern:     "hello",
		Path:        dir,
		FilePattern: "*.go",
		CtxLines:    0,
		TokenBudget: 3000,
		IsRegex:     true,
		OutputMode:  "manifest",
	})

	if !strings.Contains(result, "matches") {
		t.Errorf("expected manifest output with 'matches', got:\n%s", result)
	}
	if strings.Contains(result, "Println") {
		t.Errorf("manifest mode should not include code snippets, got:\n%s", result)
	}
	if strings.Contains(result, "│") {
		t.Errorf("manifest mode should not contain │ separator, got:\n%s", result)
	}
	if strings.Contains(result, lineRangeHint) {
		t.Errorf("manifest mode should not include line-range hint, got:\n%s", result)
	}
}
