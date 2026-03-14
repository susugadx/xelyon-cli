package file

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

type testReadFileCache struct {
	getFileFunc  func(path string) (string, bool)
	setFileCalls []testReadFileCacheSetCall
}

type testReadFileCacheSetCall struct {
	path    string
	content string
}

func (c *testReadFileCache) GetFile(path string) (string, bool) {
	if c.getFileFunc != nil {
		return c.getFileFunc(path)
	}
	return "", false
}

func (c *testReadFileCache) SetFile(path, content string) {
	c.setFileCalls = append(c.setFileCalls, testReadFileCacheSetCall{
		path:    path,
		content: content,
	})
}

func (c *testReadFileCache) GetDir(path string) (string, bool) { return "", false }

func (c *testReadFileCache) SetDir(path, result string) {}

func (c *testReadFileCache) InvalidateFile(path string) {}

func (c *testReadFileCache) InvalidateDir(path string) {}

func (c *testReadFileCache) Clear() {}

func (c *testReadFileCache) GetSearch(pattern, path string) (string, bool) { return "", false }

func (c *testReadFileCache) SetSearch(pattern, path, result string, affectedFiles []string) {}

func (c *testReadFileCache) ClearSearchCache() {}

func (c *testReadFileCache) InvalidateSearchCacheForFile(absPath string) {}

func TestExecuteReadFile_Normal(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "line1\nline2\nline3"

	testutil.CreateTempFile(t, tmpDir, "test.txt", testContent)

	output := ExecuteReadFile(testFile, 0, 0)

	if !strings.Contains(output, "line1") {
		t.Errorf("Expected content to contain 'line1', got: %s", output)
	}
}

func TestExecuteReadFile_EmptyPath(t *testing.T) {
	output := ExecuteReadFile("", 0, 0)
	if !strings.Contains(output, "Error: path is empty") {
		t.Errorf("Expected 'path is empty' error, got: %s", output)
	}
}

func TestExecuteReadFile_NonExistent(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()
	nonExistentFile := filepath.Join(tmpDir, "nonexistent.txt")

	output := ExecuteReadFile(nonExistentFile, 0, 0)
	if !strings.Contains(output, "Error reading file") {
		t.Errorf("Expected 'Error reading file', got: %s", output)
	}
}

func TestExecuteReadFile_PathTraversal(t *testing.T) {
	output := ExecuteReadFile("../../../etc/passwd", 0, 0)
	if !strings.Contains(output, "Error:") {
		t.Errorf("Expected security error for path traversal, got: %s", output)
	}
}

func TestExecuteReadFile_LargeFile_Outline(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()

	// 400行のGoファイル（関数定義を含む）
	var sb strings.Builder
	sb.WriteString("package main\n\nimport \"fmt\"\n\n")
	for i := 0; i < 5; i++ {
		fmt.Fprintf(&sb, "func handler%d() {\n", i)
		for j := 0; j < 70; j++ {
			fmt.Fprintf(&sb, "\tfmt.Println(%d)\n", j)
		}
		sb.WriteString("}\n\n")
	}
	testutil.CreateTempFile(t, tmpDir, "large.go", sb.String())

	output := ExecuteReadFile(filepath.Join(tmpDir, "large.go"), 0, 0)

	// outline-first モードの検証
	if !strings.Contains(output, "Signatures") {
		t.Errorf("Expected outline to contain 'Signatures', got:\n%s", output)
	}
	if !strings.Contains(output, "Use start_line/end_line to read function body") {
		t.Errorf("Expected guide message, got:\n%s", output)
	}
	// 先頭行が含まれる
	if !strings.Contains(output, "package main") {
		t.Errorf("Expected head lines to contain 'package main'")
	}
	// シグネチャに関数名が含まれる
	if !strings.Contains(output, "func handler") {
		t.Errorf("Expected signatures to contain function names, got:\n%s", output)
	}
	// 末尾行セクションが含まれる
	if !strings.Contains(output, "Last lines") {
		t.Errorf("Expected 'Last lines' section, got:\n%s", output)
	}
}

func TestExecuteReadFile_LargeFile_PlainText(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()

	// 400行のプレーンテキスト（ブロックなし）
	var lines []string
	for i := 0; i < 400; i++ {
		lines = append(lines, fmt.Sprintf("log entry %d", i))
	}
	testutil.CreateTempFile(t, tmpDir, "large.txt", strings.Join(lines, "\n"))

	output := ExecuteReadFile(filepath.Join(tmpDir, "large.txt"), 0, 0)

	// シグネチャなしでもアウトラインが動作する
	if !strings.Contains(output, "Use start_line/end_line to read function body") {
		t.Errorf("Expected guide message, got:\n%s", output)
	}
	if !strings.Contains(output, "Last lines") {
		t.Errorf("Expected 'Last lines' section, got:\n%s", output)
	}
	if !strings.Contains(output, "log entry 0") {
		t.Errorf("Expected head to contain first entry")
	}
}

func TestExecuteReadFile_LineRange(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "range.txt", "line1\nline2\nline3\nline4\nline5")

	output := ExecuteReadFile(filepath.Join(tmpDir, "range.txt"), 2, 4)

	if !strings.Contains(output, "line2") || !strings.Contains(output, "line4") {
		t.Errorf("Expected output to contain lines 2-4, got: %s", output)
	}
}

func TestExecuteReadFile_StartLineOnly(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()

	// 10行のファイルを作成
	var lines []string
	for i := 1; i <= 10; i++ {
		lines = append(lines, fmt.Sprintf("line%d", i))
	}
	testutil.CreateTempFile(t, tmpDir, "startonly.txt", strings.Join(lines, "\n"))

	// start_line=5, end_line=0 → 5行目から末尾まで
	output := ExecuteReadFile(filepath.Join(tmpDir, "startonly.txt"), 5, 0)

	if !strings.Contains(output, "5: line5") {
		t.Errorf("Expected output to start at line 5, got: %s", output)
	}
	if strings.Contains(output, "4: line4") {
		t.Errorf("Output should NOT contain line 4, got: %s", output)
	}
	if !strings.Contains(output, "10: line10") {
		t.Errorf("Expected output to contain last line, got: %s", output)
	}
}

func TestExecuteReadFile_EndLineOnly(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()

	var lines []string
	for i := 1; i <= 10; i++ {
		lines = append(lines, fmt.Sprintf("line%d", i))
	}
	testutil.CreateTempFile(t, tmpDir, "endonly.txt", strings.Join(lines, "\n"))

	// start_line=0, end_line=3 → 1行目から3行目まで
	output := ExecuteReadFile(filepath.Join(tmpDir, "endonly.txt"), 0, 3)

	if !strings.Contains(output, "1: line1") {
		t.Errorf("Expected output to start at line 1, got: %s", output)
	}
	if !strings.Contains(output, "3: line3") {
		t.Errorf("Expected output to contain line 3, got: %s", output)
	}
	if strings.Contains(output, "4: line4") {
		t.Errorf("Output should NOT contain line 4, got: %s", output)
	}
}

func TestExecuteReadFile_StartLineLargeFile(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()

	// 300行のファイルを作成
	var lines []string
	for i := 1; i <= 300; i++ {
		lines = append(lines, fmt.Sprintf("content_line_%d", i))
	}
	testutil.CreateTempFile(t, tmpDir, "large_start.txt", strings.Join(lines, "\n"))

	// start_line=250 → 250行目から300行（= 250-300 の51行）が返る
	output := ExecuteReadFile(filepath.Join(tmpDir, "large_start.txt"), 250, 0)

	if !strings.Contains(output, "250: content_line_250") {
		t.Errorf("Expected output to start at line 250, got: %s", output)
	}
	if strings.Contains(output, "249: content_line_249") {
		t.Errorf("Output should NOT contain line 249, got: %s", output)
	}
	if !strings.Contains(output, "300: content_line_300") {
		t.Errorf("Expected output to contain last line 300, got: %s", output)
	}
}

func TestExecuteReadFile_QuietModeSuppressesStdout(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "quiet.txt", "line1\nline2")

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	common.SetQuietMode(true)
	defer common.SetQuietMode(false)

	result := ExecuteReadFile(filepath.Join(tmpDir, "quiet.txt"), 0, 0)
	_ = w.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if strings.TrimSpace(string(data)) != "" {
		t.Fatalf("expected no stdout in quiet mode, got %q", string(data))
	}
	if !strings.Contains(result, "1: line1") {
		t.Fatalf("ExecuteReadFile() result = %q", result)
	}
}

func TestExecuteReadFileBySymbol_SingleFunction(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()
	goFile := filepath.Join(tmpDir, "sample.go")
	content := `package main

func helper() {}

// Build creates the server.
func Build() error {
	return nil
}

func after() {}
`
	if err := os.WriteFile(goFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write Go file: %v", err)
	}

	output := ExecuteReadFileBySymbol(goFile, "Build")

	if !strings.Contains(output, "── function Build") {
		t.Fatalf("expected Build header, got: %s", output)
	}
	if !strings.Contains(output, "func Build() error {") {
		t.Fatalf("expected Build body, got: %s", output)
	}
	if !strings.Contains(output, "// Build creates the server.") {
		t.Fatalf("expected surrounding comment, got: %s", output)
	}
	if strings.Contains(output, "func helper() {}") {
		t.Fatalf("did not expect helper body, got: %s", output)
	}
	if strings.Contains(output, "func after() {}") {
		t.Fatalf("did not expect after body, got: %s", output)
	}
}

func TestExecuteReadFileBySymbol_MultipleSymbols(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()
	goFile := filepath.Join(tmpDir, "sample.go")
	content := `package main

type Config struct {
	Name string
}

func Build() error {
	return nil
}
`
	if err := os.WriteFile(goFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write Go file: %v", err)
	}

	output := ExecuteReadFileBySymbol(goFile, "Build,Config")

	if !strings.Contains(output, "── struct Config") {
		t.Fatalf("expected Config header, got: %s", output)
	}
	if !strings.Contains(output, "── function Build") {
		t.Fatalf("expected Build header, got: %s", output)
	}
	if strings.Index(output, "── struct Config") > strings.Index(output, "── function Build") {
		t.Fatalf("expected symbols in file order, got: %s", output)
	}
	if !strings.Contains(output, "(2 symbol(s) from "+goFile+")") {
		t.Fatalf("expected symbol count footer, got: %s", output)
	}
}

func TestExecuteReadFileBySymbol_NotFound(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()
	goFile := filepath.Join(tmpDir, "sample.go")
	content := `package main

func Build() error {
	return nil
}

type Config struct{}
`
	if err := os.WriteFile(goFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write Go file: %v", err)
	}

	output := ExecuteReadFileBySymbol(goFile, "Missing")

	if !strings.Contains(output, "Symbol(s) not found") {
		t.Fatalf("expected not-found message, got: %s", output)
	}
	if !strings.Contains(output, "Available symbols:") {
		t.Fatalf("expected available symbols section, got: %s", output)
	}
	if !strings.Contains(output, "Build") || !strings.Contains(output, "Config") {
		t.Fatalf("expected available symbols to include Build and Config, got: %s", output)
	}
}

func TestExecuteReadFileBySymbol_NonGoFile(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()
	pyFile := filepath.Join(tmpDir, "sample.py")
	content := "def build():\n    return 42\n"
	if err := os.WriteFile(pyFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write Python file: %v", err)
	}

	output := ExecuteReadFileBySymbol(pyFile, "build")

	if !strings.Contains(output, "1: def build():") {
		t.Fatalf("expected fallback full read, got: %s", output)
	}
	if !strings.Contains(output, "2:     return 42") {
		t.Fatalf("expected fallback full read body, got: %s", output)
	}
	if strings.Contains(output, "Available symbols:") {
		t.Fatalf("did not expect symbol listing for non-Go fallback, got: %s", output)
	}
}

func TestExecuteReadFileBySymbol_MethodWithReceiver(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()
	goFile := filepath.Join(tmpDir, "server.go")
	content := `package main

type Server struct{}

func (s *Server) HandleRequest(ctx context.Context) error {
	return nil
}
`
	if err := os.WriteFile(goFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write Go file: %v", err)
	}

	output := ExecuteReadFileBySymbol(goFile, "HandleRequest")

	if !strings.Contains(output, "── method HandleRequest") {
		t.Fatalf("expected method header, got: %s", output)
	}
	if !strings.Contains(output, "func (s *Server) HandleRequest(ctx context.Context) error {") {
		t.Fatalf("expected method body, got: %s", output)
	}
}

func TestExecuteReadFileBySymbolWithRuntime_CachesContentOnMiss(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()
	goFile := filepath.Join(tmpDir, "sample.go")
	content := `package main

func Build() error {
	return nil
}
`
	if err := os.WriteFile(goFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write Go file: %v", err)
	}

	cache := &testReadFileCache{}
	output := ExecuteReadFileBySymbolWithRuntime(common.NewOutput(io.Discard, io.Discard), nil, cache, goFile, "Build")

	if !strings.Contains(output, "func Build() error {") {
		t.Fatalf("expected symbol output, got: %s", output)
	}
	if len(cache.setFileCalls) != 1 {
		t.Fatalf("SetFile call count = %d, want 1", len(cache.setFileCalls))
	}
	absPath, err := filepath.Abs(goFile)
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}
	if cache.setFileCalls[0].path != absPath {
		t.Fatalf("SetFile path = %q, want %q", cache.setFileCalls[0].path, absPath)
	}
	if !strings.Contains(cache.setFileCalls[0].content, "func Build() error {") {
		t.Fatalf("SetFile content = %q", cache.setFileCalls[0].content)
	}
}

func TestReadFileTool_BatchMode(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()

	testutil.CreateTempFile(t, tmpDir, "file1.txt", "hello from file1")
	testutil.CreateTempFile(t, tmpDir, "file2.txt", "hello from file2")

	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	tool := &ReadFileTool{}
	result, fc, err := tool.Run(tools.ExecutionContext{}, map[string]string{
		"paths": `["` + file1 + `", "` + file2 + `"]`,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fc != nil {
		t.Errorf("expected nil FileChange, got: %v", fc)
	}
	if !strings.Contains(result, "hello from file1") {
		t.Errorf("expected result to contain file1 content, got: %s", result)
	}
	if !strings.Contains(result, "hello from file2") {
		t.Errorf("expected result to contain file2 content, got: %s", result)
	}
}

func TestExecuteReadFiles_QuietModeSuppressesStdout(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "file1.txt", "hello from file1")
	testutil.CreateTempFile(t, tmpDir, "file2.txt", "hello from file2")

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	common.SetQuietMode(true)
	defer common.SetQuietMode(false)

	result := ExecuteReadFiles([]string{
		filepath.Join(tmpDir, "file1.txt"),
		filepath.Join(tmpDir, "file2.txt"),
	})
	_ = w.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if strings.TrimSpace(string(data)) != "" {
		t.Fatalf("expected no stdout in quiet mode, got %q", string(data))
	}
	if !strings.Contains(result, "📄 File: ") {
		t.Fatalf("ExecuteReadFiles() result = %q", result)
	}
}

func TestReadFileTool_BatchMode_LineRange(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()

	testutil.CreateTempFile(t, tmpDir, "range.txt", "line1\nline2\nline3\nline4\nline5")

	file := filepath.Join(tmpDir, "range.txt")

	tool := &ReadFileTool{}
	result, _, err := tool.Run(tools.ExecutionContext{}, map[string]string{
		"paths": `["` + file + `:2-3"]`,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "line2") {
		t.Errorf("expected result to contain line2, got: %s", result)
	}
	if strings.Contains(result, "line4") {
		t.Errorf("result should NOT contain line4, got: %s", result)
	}
}

func TestReadFileTool_PathRequired(t *testing.T) {
	tool := &ReadFileTool{}
	result, _, err := tool.Run(tools.ExecutionContext{}, map[string]string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Error") {
		t.Errorf("expected error when neither path nor paths provided, got: %s", result)
	}
}

func TestReadFileTool_SymbolMode(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()
	goFile := filepath.Join(tmpDir, "sample.go")
	content := `package main

func Build() error {
	return nil
}
`
	if err := os.WriteFile(goFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write Go file: %v", err)
	}

	tool := &ReadFileTool{}
	result, fc, err := tool.Run(tools.ExecutionContext{}, map[string]string{
		"path":   goFile,
		"symbol": "Build",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fc != nil {
		t.Fatalf("expected nil FileChange, got: %v", fc)
	}
	if !strings.Contains(result, "func Build() error {") {
		t.Fatalf("expected symbol mode result, got: %s", result)
	}
}

func TestReadFileTool_SymbolMode_UsesCache(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()
	goFile := filepath.Join(tmpDir, "sample.go")
	diskContent := `package main

func DiskOnly() error {
	return nil
}
`
	if err := os.WriteFile(goFile, []byte(diskContent), 0644); err != nil {
		t.Fatalf("failed to write Go file: %v", err)
	}

	cache := &testReadFileCache{
		getFileFunc: func(path string) (string, bool) {
			return `package main

func Build() error {
	return nil
}
`, true
		},
	}

	tool := &ReadFileTool{}
	result, _, err := tool.Run(tools.ExecutionContext{ToolCache: cache}, map[string]string{
		"path":   goFile,
		"symbol": "Build",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "func Build() error {") {
		t.Fatalf("expected cached symbol output, got: %s", result)
	}
	if strings.Contains(result, "DiskOnly") {
		t.Fatalf("expected cache hit to bypass disk content, got: %s", result)
	}
	if len(cache.setFileCalls) != 0 {
		t.Fatalf("SetFile should not be called on cache hit, got %d calls", len(cache.setFileCalls))
	}
}

func TestFormatOutline_Signatures(t *testing.T) {
	// 50行のGoファイルを構築（>30行で先頭ゾーン外にシグネチャがある）
	var sb strings.Builder
	sb.WriteString("package main\n\nimport \"fmt\"\n\n")
	// 先頭を埋める（30行に到達させる）
	for i := 5; i <= 30; i++ {
		fmt.Fprintf(&sb, "// line %d\n", i)
	}
	// 31行目以降に関数定義
	sb.WriteString("func Alpha() {\n")
	for i := 0; i < 5; i++ {
		sb.WriteString("\tfmt.Println()\n")
	}
	sb.WriteString("}\n\n")
	sb.WriteString("func Beta() {\n")
	for i := 0; i < 5; i++ {
		sb.WriteString("\tfmt.Println()\n")
	}
	sb.WriteString("}\n")

	lines := strings.Split(sb.String(), "\n")
	tmpDir := t.TempDir()
	goFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(goFile, []byte(sb.String()), 0644); err != nil {
		t.Fatal(err)
	}

	out := formatOutline(goFile, lines, len(lines))

	// 先頭行
	if !strings.Contains(out, "package main") {
		t.Errorf("expected head to contain package declaration")
	}
	// シグネチャ
	if !strings.Contains(out, "func Alpha") {
		t.Errorf("expected signature for Alpha, got:\n%s", out)
	}
	if !strings.Contains(out, "func Beta") {
		t.Errorf("expected signature for Beta, got:\n%s", out)
	}
	// 行番号付き
	if !strings.Contains(out, "L") {
		t.Errorf("expected line numbers in signatures, got:\n%s", out)
	}
	// ガイドメッセージ
	if !strings.Contains(out, "Use start_line/end_line") {
		t.Errorf("expected guide message")
	}
}

func TestFormatOutline_SmallFile(t *testing.T) {
	// 35行のファイル — headEnd=30, tailStart=25 → tailStart は headEnd に補正
	var sb strings.Builder
	for i := 1; i <= 35; i++ {
		fmt.Fprintf(&sb, "line %d\n", i)
	}
	lines := strings.Split(strings.TrimRight(sb.String(), "\n"), "\n")
	tmpDir := t.TempDir()
	txtFile := filepath.Join(tmpDir, "small.txt")
	if err := os.WriteFile(txtFile, []byte(sb.String()), 0644); err != nil {
		t.Fatal(err)
	}

	out := formatOutline(txtFile, lines, len(lines))

	// 先頭30行 + 末尾5行（35-30=5）
	if !strings.Contains(out, "line 1") {
		t.Errorf("expected first line")
	}
	if !strings.Contains(out, "line 35") {
		t.Errorf("expected last line, got:\n%s", out)
	}
	if !strings.Contains(out, "35 lines total") {
		t.Errorf("expected total line count, got:\n%s", out)
	}
}

func TestExecuteReadFile_BinaryFile(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()
	binFile := filepath.Join(tmpDir, "sample.bin")
	if err := os.WriteFile(binFile, []byte{0x41, 0x00, 0x42, 0x43}, 0644); err != nil {
		t.Fatalf("failed to write binary file: %v", err)
	}

	output := ExecuteReadFile(binFile, 0, 0)
	if !strings.Contains(output, "appears to be a binary file") {
		t.Fatalf("expected binary-file error, got: %s", output)
	}
}

func TestExecuteReadFile_LargeFileStreaming_Outline(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()
	largeFile := filepath.Join(tmpDir, "large.log")

	f, err := os.Create(largeFile)
	if err != nil {
		t.Fatalf("failed to create large file: %v", err)
	}
	for i := 0; i < 25000; i++ {
		if _, err := fmt.Fprintf(f, "line %05d: %s\n", i+1, strings.Repeat("x", 70)); err != nil {
			_ = f.Close()
			t.Fatalf("failed to write large file: %v", err)
		}
	}
	if err := f.Close(); err != nil {
		t.Fatalf("failed to close large file: %v", err)
	}

	output := ExecuteReadFile(largeFile, 0, 0)
	// outline-first モードに切り替わる
	if !strings.Contains(output, "Use start_line/end_line to read function body") {
		t.Fatalf("expected outline guide message for large file, got: %s", output)
	}
	if !strings.Contains(output, "1: line 00001") {
		t.Fatalf("expected first line in output, got: %s", output)
	}
	if !strings.Contains(output, "Last lines") {
		t.Fatalf("expected 'Last lines' section, got: %s", output)
	}
}

func TestExecuteReadFile_LargeFileStreaming_GoOutline(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()
	largeFile := filepath.Join(tmpDir, "large.go")

	// 1MB超のGoファイルを生成（関数定義を含む）
	f, err := os.Create(largeFile)
	if err != nil {
		t.Fatalf("failed to create large file: %v", err)
	}
	fmt.Fprintf(f, "package main\n\nimport \"fmt\"\n\n")
	for i := 0; i < 500; i++ {
		fmt.Fprintf(f, "func handler%d() {\n", i)
		for j := 0; j < 30; j++ {
			fmt.Fprintf(f, "\tfmt.Println(%q)\n", strings.Repeat("x", 100))
		}
		fmt.Fprintf(f, "}\n\n")
	}
	if err := f.Close(); err != nil {
		t.Fatalf("failed to close large file: %v", err)
	}

	output := ExecuteReadFile(largeFile, 0, 0)

	if !strings.Contains(output, "Signatures") {
		t.Fatalf("expected 'Signatures' section for Go file, got: %s", output)
	}
	if !strings.Contains(output, "func handler") {
		t.Fatalf("expected function signatures, got: %s", output)
	}
	if !strings.Contains(output, "Use start_line/end_line to read function body") {
		t.Fatalf("expected guide message, got: %s", output)
	}
}
