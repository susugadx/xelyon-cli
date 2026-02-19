package lsp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func TestLSPFindTool_Name(t *testing.T) {
	tool := &LSPFindTool{}
	if got := tool.Name(); got != "lsp_find" {
		t.Errorf("Name() = %q, want %q", got, "lsp_find")
	}
}

func TestLSPFindTool_Description(t *testing.T) {
	tool := &LSPFindTool{}
	desc := tool.Description()
	if desc == "" {
		t.Error("Description() should not be empty")
	}
	if !strings.Contains(desc, "symbol") {
		t.Error("Description() should mention 'symbol'")
	}
}

func TestLSPFindTool_Parameters(t *testing.T) {
	tool := &LSPFindTool{}
	params := tool.Parameters()

	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Parameters() should have 'properties'")
	}

	// symbol パラメータ
	if _, ok := props["symbol"]; !ok {
		t.Error("Parameters() should have 'symbol' property")
	}

	// action パラメータ
	if _, ok := props["action"]; !ok {
		t.Error("Parameters() should have 'action' property")
	}

	// scope パラメータ
	if _, ok := props["scope"]; !ok {
		t.Error("Parameters() should have 'scope' property")
	}

	// required は symbol のみ
	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("Parameters() should have 'required'")
	}
	if len(required) != 1 || required[0] != "symbol" {
		t.Errorf("required = %v, want [symbol]", required)
	}
}

func TestLSPFindTool_MissingSymbol(t *testing.T) {
	tool := &LSPFindTool{}
	output, change, err := tool.Run(map[string]string{})

	if err == nil {
		t.Error("expected error for missing symbol")
	}
	if change != nil {
		t.Errorf("expected nil change, got %v", change)
	}
	if !strings.Contains(output, "symbol is required") {
		t.Errorf("output should mention symbol is required, got: %s", output)
	}
}

func TestLSPFindTool_InvalidAction(t *testing.T) {
	tool := &LSPFindTool{}
	output, _, err := tool.Run(map[string]string{
		"symbol": "Foo",
		"action": "invalid",
	})

	if err == nil {
		t.Error("expected error for invalid action")
	}
	if !strings.Contains(output, "action must be") {
		t.Errorf("output should mention invalid action, got: %s", output)
	}
}

func TestLSPFindTool_NoLSP_GrepFallback(t *testing.T) {
	// LSP なしでも grep フォールバックで結果を返すことを確認
	originalClient := LSPClient
	LSPClient = nil
	defer func() { LSPClient = originalClient }()

	// テスト用の一時ディレクトリを作成
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "main.go")
	content := `package main

func HelloWorld() {
	fmt.Println("hello")
}

func main() {
	HelloWorld()
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := &LSPFindTool{}
	output, _, err := tool.Run(map[string]string{
		"symbol": "HelloWorld",
		"scope":  tmpDir,
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "LSP not available") {
		t.Errorf("output should mention LSP not available, got: %s", output)
	}
	if !strings.Contains(output, "HelloWorld") {
		t.Errorf("output should contain symbol name, got: %s", output)
	}
	if !strings.Contains(output, "main.go") {
		t.Errorf("output should contain file name, got: %s", output)
	}
}

func TestLSPFindTool_SymbolNotFound(t *testing.T) {
	originalClient := LSPClient
	LSPClient = nil
	defer func() { LSPClient = originalClient }()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.go")
	if err := os.WriteFile(testFile, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := &LSPFindTool{}
	output, _, err := tool.Run(map[string]string{
		"symbol": "NonExistentSymbol",
		"scope":  tmpDir,
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "not found") {
		t.Errorf("output should mention symbol not found, got: %s", output)
	}
}

func TestLSPFindTool_DefaultAction(t *testing.T) {
	// action 未指定時は "definition" がデフォルト
	originalClient := LSPClient
	LSPClient = nil
	defer func() { LSPClient = originalClient }()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	content := `package main

func MyFunc() {}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := &LSPFindTool{}
	output, _, _ := tool.Run(map[string]string{
		"symbol": "MyFunc",
		"scope":  tmpDir,
	})

	// definition アクションとして処理されたことを確認
	if !strings.Contains(output, "definition") {
		t.Errorf("default action should be 'definition', got: %s", output)
	}
}

// ===== 内部関数のテスト =====

func TestClassifyMatch(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		symbol   string
		expected matchType
	}{
		{"Go func def", "func ExecuteWebSearch(query string) string {", "ExecuteWebSearch", matchDefinition},
		{"Go method def", "func (t *LSPFindTool) Name() string {", "Name", matchDefinition},
		{"Go type def", "type LSPFindTool struct{}", "LSPFindTool", matchDefinition},
		{"Go var def", "var LSPClient *lsplib.Client", "LSPClient", matchDefinition},
		{"Go const def", "const LSPToolTimeout = 30", "LSPToolTimeout", matchDefinition},
		{"Go short assign", "result := doSomething()", "result", matchAssignment},
		{"Go assign", "result = doSomething()", "result", matchAssignment},
		{"Usage", "fmt.Println(result)", "result", matchUsage},
		{"Python func", "def my_function():", "my_function", matchDefinition},
		{"Python class", "class MyClass:", "MyClass", matchDefinition},
		{"JS function", "function myFunction() {", "myFunction", matchDefinition},
		{"JS class", "class MyClass {", "MyClass", matchDefinition},
		{"JS const", "const myVar = 42", "myVar", matchDefinition},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyMatch(tt.content, tt.symbol)
			if got != tt.expected {
				t.Errorf("classifyMatch(%q, %q) = %d, want %d", tt.content, tt.symbol, got, tt.expected)
			}
		})
	}
}

func TestFindSymbolColumn(t *testing.T) {
	tests := []struct {
		content  string
		symbol   string
		expected int
	}{
		{"func HelloWorld() {", "HelloWorld", 6},
		{"	result := doSomething()", "result", 2},
		{"fmt.Println(result)", "result", 13},
		{"type Foo struct{}", "Foo", 6},
	}

	for _, tt := range tests {
		t.Run(tt.symbol, func(t *testing.T) {
			got := findSymbolColumn(tt.content, tt.symbol)
			if got != tt.expected {
				t.Errorf("findSymbolColumn(%q, %q) = %d, want %d", tt.content, tt.symbol, got, tt.expected)
			}
		})
	}
}

func TestIsTestFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"internal/tools/lsp/tools_test.go", true},
		{"internal/tools/lsp/tools.go", false},
		{"src/component.test.js", true},
		{"src/component.spec.ts", true},
		{"src/component.js", false},
		{"test_helper.py", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isTestFile(tt.path)
			if got != tt.expected {
				t.Errorf("isTestFile(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}

func TestSortMatchesByPriority(t *testing.T) {
	matches := []symbolMatch{
		{FilePath: "a.go", MatchType: matchUsage},
		{FilePath: "b.go", MatchType: matchDefinition},
		{FilePath: "c_test.go", MatchType: matchDefinition},
		{FilePath: "d.go", MatchType: matchAssignment},
	}

	sortMatchesByPriority(matches)

	// 1. 定義（非テスト） → 2. 定義（テスト） → 3. 代入 → 4. 使用
	if matches[0].FilePath != "b.go" {
		t.Errorf("first match should be non-test definition, got %s", matches[0].FilePath)
	}
	if matches[1].FilePath != "c_test.go" {
		t.Errorf("second match should be test definition, got %s", matches[1].FilePath)
	}
	if matches[2].FilePath != "d.go" {
		t.Errorf("third match should be assignment, got %s", matches[2].FilePath)
	}
	if matches[3].FilePath != "a.go" {
		t.Errorf("fourth match should be usage, got %s", matches[3].FilePath)
	}
}

func TestParseGrepLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		symbol   string
		wantNil  bool
		wantLine int
		wantType matchType
	}{
		{
			name:     "Go func definition",
			line:     "internal/tools/lsp/tools.go:35:func (t *LSPReferencesTool) Name() string { return \"lsp_references\" }",
			symbol:   "Name",
			wantNil:  false,
			wantLine: 35,
			wantType: matchDefinition,
		},
		{
			name:     "Usage line",
			line:     "main.go:10:	result := Name()",
			symbol:   "Name",
			wantNil:  false,
			wantLine: 10,
			wantType: matchUsage,
		},
		{
			name:    "Invalid line",
			line:    "no-colon-here",
			symbol:  "test",
			wantNil: true,
		},
		{
			name:    "Invalid line number",
			line:    "file.go:abc:content",
			symbol:  "test",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseGrepLine(tt.line, tt.symbol)
			if tt.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil result")
			}
			if got.Line != tt.wantLine {
				t.Errorf("Line = %d, want %d", got.Line, tt.wantLine)
			}
			if got.MatchType != tt.wantType {
				t.Errorf("MatchType = %d, want %d", got.MatchType, tt.wantType)
			}
		})
	}
}

func TestFormatGrepFallback(t *testing.T) {
	matches := []symbolMatch{
		{FilePath: "main.go", Line: 3, Column: 6, Content: "func HelloWorld() {", MatchType: matchDefinition},
		{FilePath: "main.go", Line: 8, Column: 2, Content: "HelloWorld()", MatchType: matchUsage},
	}

	// LSP なし
	originalClient := LSPClient
	LSPClient = nil
	defer func() { LSPClient = originalClient }()

	output := formatGrepFallback("HelloWorld", "definition", matches)

	if !strings.Contains(output, "LSP not available") {
		t.Errorf("should mention LSP not available, got: %s", output)
	}
	if !strings.Contains(output, "[DEF]") {
		t.Errorf("should show [DEF] for definition matches, got: %s", output)
	}
	if !strings.Contains(output, "main.go:3:6") {
		t.Errorf("should show definition location, got: %s", output)
	}
}

func TestFindSymbolLocations(t *testing.T) {
	tmpDir := t.TempDir()

	// テストファイルを作成
	goFile := filepath.Join(tmpDir, "main.go")
	content := `package main

import "fmt"

func HelloWorld() {
	fmt.Println("hello")
}

func main() {
	HelloWorld()
}
`
	if err := os.WriteFile(goFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	matches := findSymbolLocations("HelloWorld", tmpDir)

	if len(matches) == 0 {
		t.Fatal("expected at least one match")
	}

	// 定義箇所が見つかるべき
	foundDef := false
	for _, m := range matches {
		if m.MatchType == matchDefinition {
			foundDef = true
			break
		}
	}
	if !foundDef {
		t.Error("expected to find a definition match")
	}
}

func TestReadFileLine(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	content := "line1\nline2\nline3\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if got := readFileLine(testFile, 1); got != "line1" {
		t.Errorf("readFileLine(1) = %q, want %q", got, "line1")
	}
	if got := readFileLine(testFile, 2); got != "line2" {
		t.Errorf("readFileLine(2) = %q, want %q", got, "line2")
	}
	if got := readFileLine(testFile, 3); got != "line3" {
		t.Errorf("readFileLine(3) = %q, want %q", got, "line3")
	}
	if got := readFileLine(testFile, 99); got != "" {
		t.Errorf("readFileLine(99) = %q, want empty", got)
	}
	if got := readFileLine("/nonexistent/file", 1); got != "" {
		t.Errorf("readFileLine(nonexistent) = %q, want empty", got)
	}
}

func TestRegisterTools_IncludesLSPFind(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterTools(registry)

	// lsp_find が登録されていることを確認
	if tool := registry.GetTool("lsp_find"); tool == nil {
		t.Error("lsp_find tool not registered")
	}
}

func TestLSPFindTool_SafetyLevel(t *testing.T) {
	safety := common.GetToolSafety("lsp_find")
	if safety != common.SafetyHigh {
		t.Errorf("lsp_find safety = %v, want SafetyHigh", safety)
	}
}
