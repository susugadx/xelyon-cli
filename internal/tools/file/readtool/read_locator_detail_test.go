package readtool

import (
	"strconv"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/testutil"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestReadFileTool_TargetsCompactSingleLineUsesEnclosingBlock(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	filePath := tmpDir + "/compact.go"
	testutil.CreateTempFile(t, tmpDir, "compact.go", strings.Join([]string{
		"package main",
		"",
		"func alpha() {",
		"\tprintln(\"alpha\")",
		"}",
		"",
		"func beta() {",
		"\tprintln(\"beta\")",
		"\tprintln(\"gamma\")",
		"}",
	}, "\n"))

	reg := locator.NewRegistry()
	reg.Register(locator.Location{FilePath: filePath, Line: 8})

	tool := &ReadFileTool{}
	result, _, err := tool.Run(tools.ExecutionContext{LocatorRegistry: reg}, map[string]string{
		"targets": "[L1]",
		"detail":  "compact",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "📄 File: "+filePath+":7-10 [L2]") {
		t.Fatalf("expected expanded header with actual range, got:\n%s", result)
	}
	if !strings.Contains(result, "7: func beta() {") || !strings.Contains(result, "10: }") {
		t.Fatalf("expected enclosing block content, got:\n%s", result)
	}
	if strings.Contains(result, "3: func alpha() {") {
		t.Fatalf("compact locator read should not include sibling blocks, got:\n%s", result)
	}
}

func TestReadFileTool_TargetsCompactRespectsExistingLocatorRange(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	filePath := tmpDir + "/range.go"
	testutil.CreateTempFile(t, tmpDir, "range.go", strings.Join([]string{
		"package main",
		"",
		"func target() {",
		"\tprintln(\"one\")",
		"\tprintln(\"two\")",
		"}",
	}, "\n"))

	reg := locator.NewRegistry()
	reg.Register(locator.Location{FilePath: filePath, Line: 3, EndLine: 6})

	tool := &ReadFileTool{}
	result, _, err := tool.Run(tools.ExecutionContext{LocatorRegistry: reg}, map[string]string{
		"targets": "[L1]",
		"detail":  "compact",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "📄 File: "+filePath+":3-6 [L1]") {
		t.Fatalf("expected existing locator range to be preserved, got:\n%s", result)
	}
	if !strings.Contains(result, "3: func target() {") || !strings.Contains(result, "6: }") {
		t.Fatalf("expected preserved range content, got:\n%s", result)
	}
}

func TestReadFileTool_TargetsCompactFallsBackToWindow(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	filePath := tmpDir + "/notes.txt"
	lines := make([]string, 120)
	for i := range lines {
		lines[i] = "line" + strconv.Itoa(i+1)
	}
	testutil.CreateTempFile(t, tmpDir, "notes.txt", strings.Join(lines, "\n"))

	reg := locator.NewRegistry()
	reg.Register(locator.Location{FilePath: filePath, Line: 50})

	tool := &ReadFileTool{}
	result, _, err := tool.Run(tools.ExecutionContext{LocatorRegistry: reg}, map[string]string{
		"targets": "[L1]",
		"detail":  "compact",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "📄 File: "+filePath+":45-100 [L2]") {
		t.Fatalf("expected fallback window header, got:\n%s", result)
	}
	if !strings.Contains(result, "45: line45") || !strings.Contains(result, "100: line100") {
		t.Fatalf("expected fallback window content, got:\n%s", result)
	}
	if strings.Contains(result, "44: line44") {
		t.Fatalf("fallback window should start at line 45, got:\n%s", result)
	}
}

func TestReadFileTool_TargetsSingleLineFullReadsWholeFile(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	filePath := tmpDir + "/full.txt"
	lines := make([]string, 120)
	for i := range lines {
		lines[i] = "line" + strconv.Itoa(i+1)
	}
	testutil.CreateTempFile(t, tmpDir, "full.txt", strings.Join(lines, "\n"))

	reg := locator.NewRegistry()
	reg.Register(locator.Location{FilePath: filePath, Line: 50})

	tool := &ReadFileTool{}
	result, _, err := tool.Run(tools.ExecutionContext{LocatorRegistry: reg}, map[string]string{
		"targets": "[L1]",
		"detail":  "full",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "📄 File: "+filePath) {
		t.Fatalf("expected whole-file header, got:\n%s", result)
	}
	if !strings.Contains(result, "1: line1") || !strings.Contains(result, "120: line120") {
		t.Fatalf("detail=full should read the whole file for single-line locator, got:\n%s", result)
	}
	if strings.Contains(result, filePath+":45-100") {
		t.Fatalf("detail=full should not fall back to a locator window, got:\n%s", result)
	}
}

func TestReadFileTool_TargetsSingleLineOutlineReadsWholeFileOutline(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	filePath := tmpDir + "/outline.txt"
	lines := make([]string, 120)
	for i := range lines {
		lines[i] = "line" + strconv.Itoa(i+1)
	}
	testutil.CreateTempFile(t, tmpDir, "outline.txt", strings.Join(lines, "\n"))

	reg := locator.NewRegistry()
	reg.Register(locator.Location{FilePath: filePath, Line: 50})

	tool := &ReadFileTool{}
	result, _, err := tool.Run(tools.ExecutionContext{LocatorRegistry: reg}, map[string]string{
		"targets": "[L1]",
		"detail":  "outline",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "lines total") {
		t.Fatalf("detail=outline should force whole-file outline for single-line locator, got:\n%s", result)
	}
	if strings.Contains(result, "60: line60") {
		t.Fatalf("outline output should omit middle lines, got:\n%s", result)
	}
	if strings.Contains(result, filePath+":45-100") {
		t.Fatalf("detail=outline should not fall back to a locator window, got:\n%s", result)
	}
}

func TestReadFileTool_TargetsRangeDetailFullReadsWholeFile(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	filePath := tmpDir + "/range-full.txt"
	lines := make([]string, 120)
	for i := range lines {
		lines[i] = "line" + strconv.Itoa(i+1)
	}
	testutil.CreateTempFile(t, tmpDir, "range-full.txt", strings.Join(lines, "\n"))

	reg := locator.NewRegistry()
	reg.Register(locator.Location{FilePath: filePath, Line: 10, EndLine: 20})
	reg.Register(locator.Location{FilePath: filePath, Line: 60, EndLine: 70})

	tool := &ReadFileTool{}
	result, _, err := tool.Run(tools.ExecutionContext{LocatorRegistry: reg}, map[string]string{
		"targets": "[L1,L2]",
		"detail":  "full",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count := strings.Count(result, "📄 File: "); count != 1 {
		t.Fatalf("expected whole-file detail dedupe for range locators, got %d headers:\n%s", count, result)
	}
	if strings.Contains(result, filePath+":10-20") || strings.Contains(result, filePath+":60-70") {
		t.Fatalf("detail=full should override locator ranges to whole-file read, got:\n%s", result)
	}
	if !strings.Contains(result, "1: line1") || !strings.Contains(result, "120: line120") {
		t.Fatalf("expected whole-file content, got:\n%s", result)
	}
}

func TestReadFileTool_TargetsSingleLineRangeAutoPreservesExactRead(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	filePath := tmpDir + "/single-line.txt"
	lines := make([]string, 120)
	for i := range lines {
		lines[i] = "line" + strconv.Itoa(i+1)
	}
	testutil.CreateTempFile(t, tmpDir, "single-line.txt", strings.Join(lines, "\n"))

	reg := locator.NewRegistry()
	reg.Register(locator.Location{FilePath: filePath, Line: 50, EndLine: 50})

	tool := &ReadFileTool{}
	result, _, err := tool.Run(tools.ExecutionContext{LocatorRegistry: reg}, map[string]string{
		"targets": "[L1]",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "📄 File: "+filePath+":50-50 [L1]") {
		t.Fatalf("expected exact single-line locator range, got:\n%s", result)
	}
	if !strings.Contains(result, "50: line50") {
		t.Fatalf("expected exact target line, got:\n%s", result)
	}
	if strings.Contains(result, "49: line49") || strings.Contains(result, "51: line51") {
		t.Fatalf("detail=auto should preserve exact single-line locator span, got:\n%s", result)
	}
}

func TestReadFileTool_TargetsCompactFileLevelLocatorFallsBackToWholeFile(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	filePath := tmpDir + "/file-level.txt"
	lines := make([]string, 80)
	for i := range lines {
		lines[i] = "line" + strconv.Itoa(i+1)
	}
	testutil.CreateTempFile(t, tmpDir, "file-level.txt", strings.Join(lines, "\n"))

	reg := locator.NewRegistry()
	reg.Register(locator.Location{FilePath: filePath})

	tool := &ReadFileTool{}
	result, _, err := tool.Run(tools.ExecutionContext{LocatorRegistry: reg}, map[string]string{
		"targets": "[L1]",
		"detail":  "compact",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, `Error: detail="compact"`) {
		t.Fatalf("file-level locator compact should fall back instead of erroring, got:\n%s", result)
	}
	if !strings.Contains(result, "📄 File: "+filePath+" [L1]") {
		t.Fatalf("expected whole-file locator header, got:\n%s", result)
	}
	if !strings.Contains(result, "1: line1") || !strings.Contains(result, "80: line80") {
		t.Fatalf("expected whole-file fallback content, got:\n%s", result)
	}
}

func TestReadFileTool_TargetsCompactFileLevelLocatorLargeSingleLineFallsBackToOutline(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	filePath := tmpDir + "/manifest.min.js"
	testutil.CreateTempFile(t, tmpDir, "manifest.min.js", strings.Repeat("x", LargeFileThreshold+1024))

	reg := locator.NewRegistry()
	reg.Register(locator.Location{FilePath: filePath})

	tool := &ReadFileTool{}
	result, _, err := tool.Run(tools.ExecutionContext{LocatorRegistry: reg}, map[string]string{
		"targets": "[L1]",
		"detail":  "compact",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, "token too long") || strings.Contains(result, "Error reading file:") {
		t.Fatalf("file-level locator compact should not fail on large single-line files, got:\n%s", result)
	}
	if !strings.Contains(result, "📄 File: "+filePath+" [L1]") {
		t.Fatalf("expected whole-file locator header, got:\n%s", result)
	}
	if !strings.Contains(result, "lines total") || !strings.Contains(result, "...") {
		t.Fatalf("expected safe outline fallback, got:\n%s", result)
	}
	if len(result) > 10000 {
		t.Fatalf("file-level locator compact fallback should stay bounded, got %d bytes", len(result))
	}
}

func TestReadFileTool_TargetsWholeFileDetailDedupesSameFile(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	filePath := tmpDir + "/dedupe-full.txt"
	lines := make([]string, 120)
	for i := range lines {
		lines[i] = "line" + strconv.Itoa(i+1)
	}
	testutil.CreateTempFile(t, tmpDir, "dedupe-full.txt", strings.Join(lines, "\n"))

	reg := locator.NewRegistry()
	reg.Register(locator.Location{FilePath: filePath, Line: 10})
	reg.Register(locator.Location{FilePath: filePath, Line: 90})

	tool := &ReadFileTool{}
	result, _, err := tool.Run(tools.ExecutionContext{LocatorRegistry: reg}, map[string]string{
		"targets": "[L1,L2]",
		"detail":  "full",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count := strings.Count(result, "📄 File: "); count != 1 {
		t.Fatalf("expected deduped whole-file detail read, got %d headers:\n%s", count, result)
	}
	if !strings.Contains(result, "1: line1") || !strings.Contains(result, "120: line120") {
		t.Fatalf("expected whole-file content, got:\n%s", result)
	}
}

func TestReadFileTool_TargetsWholeFileDetailDedupesBeforeMaxReadValidation(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	filePath := tmpDir + "/dedupe-limit.txt"
	lines := make([]string, 120)
	for i := range lines {
		lines[i] = "line" + strconv.Itoa(i+1)
	}
	testutil.CreateTempFile(t, tmpDir, "dedupe-limit.txt", strings.Join(lines, "\n"))

	reg := locator.NewRegistry()
	targetIDs := make([]string, 0, MaxReadFilesPaths+1)
	for i := 0; i < MaxReadFilesPaths+1; i++ {
		id := reg.Register(locator.Location{FilePath: filePath, Line: i + 1})
		targetIDs = append(targetIDs, id)
	}

	tool := &ReadFileTool{}
	result, _, err := tool.Run(tools.ExecutionContext{LocatorRegistry: reg}, map[string]string{
		"targets": "[" + strings.Join(targetIDs, ",") + "]",
		"detail":  "full",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, "Error: too many paths") {
		t.Fatalf("deduped locator full read should not fail max-read validation, got:\n%s", result)
	}
	if count := strings.Count(result, "📄 File: "); count != 1 {
		t.Fatalf("expected one deduped whole-file result, got %d headers:\n%s", count, result)
	}
}
