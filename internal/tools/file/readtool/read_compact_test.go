package readtool

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/testutil"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestReadFileTool_DetailCompactKeepsExplicitRange(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	lines := make([]string, 40)
	for i := range lines {
		lines[i] = fmt.Sprintf("line%d", i+1)
	}
	filePath := filepath.Join(tmpDir, "explicit.txt")
	testutil.CreateTempFile(t, tmpDir, "explicit.txt", strings.Join(lines, "\n"))

	pathsJSON, err := json.Marshal([]string{filePath + ":10-20"})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	tool := &ReadFileTool{}
	result, _, err := tool.Run(newTestToolExecContext(), map[string]string{
		"paths":  string(pathsJSON),
		"detail": "compact",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "📄 File: "+filePath+":10-20") {
		t.Fatalf("expected explicit range header, got:\n%s", result)
	}
	if !strings.Contains(result, "10: line10") || !strings.Contains(result, "20: line20") {
		t.Fatalf("expected requested range content, got:\n%s", result)
	}
	if strings.Contains(result, "9: line9") || strings.Contains(result, "21: line21") {
		t.Fatalf("compact explicit range should not widen, got:\n%s", result)
	}
}

func TestReadFileTool_DetailCompactTreatsSingleLinePathAsExactRange(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	lines := make([]string, 40)
	for i := range lines {
		lines[i] = fmt.Sprintf("line%d", i+1)
	}
	filePath := filepath.Join(tmpDir, "single.txt")
	testutil.CreateTempFile(t, tmpDir, "single.txt", strings.Join(lines, "\n"))

	pathsJSON, err := json.Marshal([]string{filePath + ":10"})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	tool := &ReadFileTool{}
	result, _, err := tool.Run(newTestToolExecContext(), map[string]string{
		"paths":  string(pathsJSON),
		"detail": "compact",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "📄 File: "+filePath+":10") {
		t.Fatalf("expected single-line path header, got:\n%s", result)
	}
	if !strings.Contains(result, "10: line10") {
		t.Fatalf("expected exact requested line, got:\n%s", result)
	}
	if strings.Contains(result, "9: line9") || strings.Contains(result, "11: line11") {
		t.Fatalf("compact single-line path should not widen, got:\n%s", result)
	}
}

func TestReadFileTool_DetailCompactDedupesExpandedLocatorRanges(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "dedupe.go")
	testutil.CreateTempFile(t, tmpDir, "dedupe.go", strings.Join([]string{
		"package main",
		"",
		"func shared() {",
		"\tprintln(\"one\")",
		"\tprintln(\"two\")",
		"\tprintln(\"three\")",
		"}",
	}, "\n"))

	reg := locator.NewRegistry()
	reg.Register(locator.Location{FilePath: filePath, Line: 4})
	reg.Register(locator.Location{FilePath: filePath, Line: 5})

	tool := &ReadFileTool{}
	result, _, err := tool.Run(tools.ExecutionContext{LocatorRegistry: reg}, map[string]string{
		"targets": "[L1,L2]",
		"detail":  "compact",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count := strings.Count(result, "📄 File: "); count != 1 {
		t.Fatalf("expected deduped compact read, got %d headers:\n%s", count, result)
	}
	if !strings.Contains(result, "📄 File: "+filePath+":3-7 [L3]") {
		t.Fatalf("expected actual expanded range header, got:\n%s", result)
	}
}

func TestReadFileTool_DetailCompactLargeFileFallsBackToWindow(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "large.txt")
	lines := make([]string, 1300)
	for i := range lines {
		lines[i] = fmt.Sprintf("line%d %s", i+1, strings.Repeat("x", 900))
	}
	testutil.CreateTempFile(t, tmpDir, "large.txt", strings.Join(lines, "\n"))

	reg := locator.NewRegistry()
	reg.Register(locator.Location{FilePath: filePath, Line: 600})

	cache := &countingReadCache{}
	tool := &ReadFileTool{}
	result, _, err := tool.Run(tools.ExecutionContext{
		LocatorRegistry: reg,
		ToolCache:       cache,
	}, map[string]string{
		"targets": "[L1]",
		"detail":  "compact",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "📄 File: "+filePath+":595-650 [L2]") {
		t.Fatalf("expected large-file compact fallback window, got:\n%s", result)
	}
	if cache.setFileCalls != 0 {
		t.Fatalf("large-file compact fallback should avoid whole-file reads, got %d SetFile calls", cache.setFileCalls)
	}
}

func TestReadFileTool_DetailCompactReusesWholeFileCacheForRangeRead(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "cached.go")
	testutil.CreateTempFile(t, tmpDir, "cached.go", strings.Join([]string{
		"package main",
		"",
		"func target() {",
		"\tprintln(\"one\")",
		"\tprintln(\"two\")",
		"\tprintln(\"three\")",
		"}",
	}, "\n"))

	reg := locator.NewRegistry()
	reg.Register(locator.Location{FilePath: filePath, Line: 4})

	cache := &countingReadCache{}
	tool := &ReadFileTool{}
	result, _, err := tool.Run(tools.ExecutionContext{
		LocatorRegistry: reg,
		ToolCache:       cache,
	}, map[string]string{
		"targets": "[L1]",
		"detail":  "compact",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "📄 File: "+filePath+":3-7 [L2]") {
		t.Fatalf("expected compact block result, got:\n%s", result)
	}
	if cache.setFileCalls != 1 {
		t.Fatalf("expected one whole-file cache population from block scan, got %d", cache.setFileCalls)
	}
	if cache.getFileCalls < 2 {
		t.Fatalf("expected range read to reuse cached whole-file content, got %d GetFile calls", cache.getFileCalls)
	}
}

func TestReadFileTool_DetailCompactLargeFileReverseRangeErrors(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "reverse.txt")
	lines := make([]string, 1300)
	for i := range lines {
		lines[i] = fmt.Sprintf("line%d %s", i+1, strings.Repeat("x", 900))
	}
	testutil.CreateTempFile(t, tmpDir, "reverse.txt", strings.Join(lines, "\n"))

	pathsJSON, err := json.Marshal([]string{filePath + ":20-10"})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	tool := &ReadFileTool{}
	result, _, err := tool.Run(newTestToolExecContext(), map[string]string{
		"paths":  string(pathsJSON),
		"detail": "compact",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Error: start_line 20 is greater than end_line 10") {
		t.Fatalf("expected reverse range error, got:\n%s", result)
	}
}

func TestReadFileTool_DetailCompactLargeCachedLocatorFallsBackToWindow(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "large-cached.go")
	lines := make([]string, 0, 1304)
	lines = append(lines, "package main", "", "func target() {")
	for i := 0; i < 1300; i++ {
		lines = append(lines, fmt.Sprintf("\tprintln(%q)", strings.Repeat("x", 900)))
	}
	lines = append(lines, "}")
	content := strings.Join(lines, "\n")
	testutil.CreateTempFile(t, tmpDir, "large-cached.go", content)

	reg := locator.NewRegistry()
	reg.Register(locator.Location{FilePath: filePath, Line: 600})

	cache := &countingReadCache{}
	cache.SetFile(filePath, content)

	tool := &ReadFileTool{}
	result, _, err := tool.Run(tools.ExecutionContext{
		LocatorRegistry: reg,
		ToolCache:       cache,
	}, map[string]string{
		"targets": "[L1]",
		"detail":  "compact",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "📄 File: "+filePath+":595-650 [L2]") {
		t.Fatalf("expected cached large-file compact read to use fallback window, got:\n%s", result)
	}
	if strings.Contains(result, "3: func target() {") || strings.Contains(result, "1304: }") {
		t.Fatalf("cached large-file compact read should not expand to the huge enclosing block, got:\n%s", result)
	}
}

func TestReadFileTool_DetailCompactLargeSingleLinePathTruncatesOutput(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "minified.js")
	testutil.CreateTempFile(t, tmpDir, "minified.js", strings.Repeat("x", LargeFileThreshold+1024))

	pathsJSON, err := json.Marshal([]string{filePath + ":1"})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	tool := &ReadFileTool{}
	result, _, err := tool.Run(newTestToolExecContext(), map[string]string{
		"paths":  string(pathsJSON),
		"detail": "compact",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, "Error reading file:") {
		t.Fatalf("compact single-line path should not fail on large single-line files, got:\n%s", result)
	}
	if !strings.Contains(result, "1: ") || !strings.Contains(result, "...") {
		t.Fatalf("expected compact range output to be truncated, got:\n%s", result)
	}
	if len(result) > 6000 {
		t.Fatalf("compact single-line output should stay bounded, got %d bytes", len(result))
	}
}

func TestReadFileTool_DetailCompactSmallSingleLinePathTruncatesOutput(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "small-minified.js")
	testutil.CreateTempFile(t, tmpDir, "small-minified.js", strings.Repeat("x", 900*1024))

	pathsJSON, err := json.Marshal([]string{filePath + ":1"})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	tool := &ReadFileTool{}
	result, _, err := tool.Run(newTestToolExecContext(), map[string]string{
		"paths":  string(pathsJSON),
		"detail": "compact",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "1: ") || !strings.Contains(result, "...") {
		t.Fatalf("expected bounded compact output for sub-threshold single-line file, got:\n%s", result)
	}
	if len(result) > 6000 {
		t.Fatalf("compact single-line output should stay bounded, got %d bytes", len(result))
	}
}

func TestReadFileTool_DetailCompactCachedLargeSingleLinePathTruncatesOutput(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "cached-minified.js")
	content := strings.Repeat("x", LargeFileThreshold+1024)
	testutil.CreateTempFile(t, tmpDir, "cached-minified.js", content)

	pathsJSON, err := json.Marshal([]string{filePath + ":1"})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	cache := &countingReadCache{}
	cache.SetFile(filePath, content)

	tool := &ReadFileTool{}
	result, _, err := tool.Run(tools.ExecutionContext{ToolCache: cache}, map[string]string{
		"paths":  string(pathsJSON),
		"detail": "compact",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "1: ") || !strings.Contains(result, "...") {
		t.Fatalf("expected bounded compact output from cached whole-file content, got:\n%s", result)
	}
	if len(result) > 6000 {
		t.Fatalf("compact cached single-line output should stay bounded, got %d bytes", len(result))
	}
}

func TestReadFileTool_DetailCompactCachedRangeNormalizesCRLF(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "crlf.txt")
	content := "line1\r\nline2\r\nline3\r\n"
	testutil.CreateTempFile(t, tmpDir, "crlf.txt", content)

	pathsJSON, err := json.Marshal([]string{filePath + ":2-3"})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	cache := &countingReadCache{}
	cache.SetFile(filePath, content)

	tool := &ReadFileTool{}
	result, _, err := tool.Run(tools.ExecutionContext{ToolCache: cache}, map[string]string{
		"paths":  string(pathsJSON),
		"detail": "compact",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, "\r") {
		t.Fatalf("compact cached range should normalize CRLF, got raw carriage return in:\n%q", result)
	}
	if !strings.Contains(result, "2: line2") || !strings.Contains(result, "3: line3") {
		t.Fatalf("expected normalized cached range output, got:\n%s", result)
	}
}

func TestReadFileTool_DetailCompactCachedRangeDoesNotCountTrailingNewlineAsExtraLine(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "trailing-newline.txt")
	content := "line1\nline2\nline3\n"
	testutil.CreateTempFile(t, tmpDir, "trailing-newline.txt", content)

	pathsJSON, err := json.Marshal([]string{filePath + ":4"})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	cache := &countingReadCache{}
	cache.SetFile(filePath, content)

	tool := &ReadFileTool{}
	result, _, err := tool.Run(tools.ExecutionContext{ToolCache: cache}, map[string]string{
		"paths":  string(pathsJSON),
		"detail": "compact",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Error: start_line 4 exceeds total lines 3") {
		t.Fatalf("expected EOF validation against 3 real lines, got:\n%s", result)
	}
	if strings.Contains(result, "4: ") || strings.Contains(result, "of 4") {
		t.Fatalf("trailing newline must not create a phantom fourth line, got:\n%s", result)
	}
}
