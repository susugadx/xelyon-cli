package tools

import (
	"os/exec"
	"strings"
	"testing"
)

func TestAstGrepEmptyPattern(t *testing.T) {
	result := executeAstGrep("", "", "")
	if !strings.HasPrefix(result, "Error: pattern is required") {
		t.Errorf("Expected error for empty pattern, got: %s", result)
	}
}

func TestAstGrepNotInstalled(t *testing.T) {
	// このテストはast-grepがインストールされていない環境でのみ動作
	if _, err := exec.LookPath("ast-grep"); err == nil {
		t.Skip("ast-grep is installed, skipping not-installed test")
	}

	result := executeAstGrep("func $NAME", "", "")
	if !strings.Contains(result, "ast-grep is not installed") {
		t.Errorf("Expected not installed error, got: %s", result)
	}
}

func TestAstGrepBasicGoPattern(t *testing.T) {
	if _, err := exec.LookPath("ast-grep"); err != nil {
		t.Skip("ast-grep not installed, skipping")
	}

	// 現在のプロジェクトでGoの関数を検索
	result := executeAstGrep("func $NAME($ARGS)", "go", ".")
	// マッチがあるか、No matchesのいずれか
	if result == "" {
		t.Error("Expected non-empty result")
	}
}

func TestAstGrepWithLang(t *testing.T) {
	if _, err := exec.LookPath("ast-grep"); err != nil {
		t.Skip("ast-grep not installed, skipping")
	}

	// 言語指定ありでの検索
	result := executeAstGrep("if err != nil { $BODY }", "go", ".")
	// 結果があるか確認（エラーでないこと）
	if strings.HasPrefix(result, "Error:") {
		t.Errorf("Expected valid result, got error: %s", result)
	}
}

func TestAstGrepWithPath(t *testing.T) {
	if _, err := exec.LookPath("ast-grep"); err != nil {
		t.Skip("ast-grep not installed, skipping")
	}

	// 特定ディレクトリでの検索
	result := executeAstGrep("type $NAME struct", "go", "./internal/tools")
	if strings.HasPrefix(result, "Error:") && !strings.Contains(result, "No matches") {
		t.Errorf("Expected valid result or no matches, got: %s", result)
	}
}

func TestAstGrepHelpContent(t *testing.T) {
	help := getAstGrepHelp()
	if !strings.Contains(help, "Metavariables") {
		t.Error("Help should contain Metavariables section")
	}
	if !strings.Contains(help, "$NAME") {
		t.Error("Help should contain $NAME example")
	}
	if !strings.Contains(help, "$$$") {
		t.Error("Help should contain $$$ example")
	}
}

func TestAstGrepToolRegistration(t *testing.T) {
	// ツールが正しく登録されているか確認
	tool := &AstGrepTool{}
	if tool.Name() != "ast_grep" {
		t.Errorf("Expected tool name 'ast_grep', got '%s'", tool.Name())
	}
}

func TestAstGrepToolRun(t *testing.T) {
	tool := &AstGrepTool{}

	// 空パターンでのエラー確認
	result, change, err := tool.Run(map[string]string{"pattern": ""})
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if change != nil {
		t.Error("Expected no file change for ast_grep")
	}
	if !strings.HasPrefix(result, "Error:") {
		t.Errorf("Expected error result, got: %s", result)
	}
}

// Phase 2: Pattern Suggestion Tests

func TestGetPatternSuggestionGo(t *testing.T) {
	tests := []struct {
		query           string
		expectedPattern string
		expectedLang    string
	}{
		{"Find Go functions", "func $NAME($ARGS)", "go"},
		{"golang error handling", "if err != nil { $$$ }", "go"},
		{"go functions returning error", "func $NAME($ARGS) error", "go"},
		{"go struct definitions", "type $NAME struct { $$$ }", "go"},
		{"go interface", "type $NAME interface { $$$ }", "go"},
		{"go method", "func ($R $TYPE) $NAME($ARGS)", "go"},
	}

	for _, tt := range tests {
		suggestion := GetPatternSuggestion(tt.query)
		if suggestion.Pattern != tt.expectedPattern {
			t.Errorf("Query '%s': expected pattern '%s', got '%s'", tt.query, tt.expectedPattern, suggestion.Pattern)
		}
		if suggestion.Lang != tt.expectedLang {
			t.Errorf("Query '%s': expected lang '%s', got '%s'", tt.query, tt.expectedLang, suggestion.Lang)
		}
	}
}

func TestGetPatternSuggestionJS(t *testing.T) {
	tests := []struct {
		query           string
		expectedPattern string
		expectedLang    string
	}{
		{"javascript async functions", "async function $NAME($ARGS)", "js"},
		{"js console.log", "console.log($ARG)", "js"},
		{"react useState hook", "useState($INIT)", "tsx"},
		{"typescript try catch", "try { $$$ } catch ($E) { $$$ }", "js"},
		{"js class definition", "class $NAME { $$$ }", "js"},
		{"javascript function", "function $NAME($ARGS)", "js"},
	}

	for _, tt := range tests {
		suggestion := GetPatternSuggestion(tt.query)
		if suggestion.Pattern != tt.expectedPattern {
			t.Errorf("Query '%s': expected pattern '%s', got '%s'", tt.query, tt.expectedPattern, suggestion.Pattern)
		}
		if suggestion.Lang != tt.expectedLang {
			t.Errorf("Query '%s': expected lang '%s', got '%s'", tt.query, tt.expectedLang, suggestion.Lang)
		}
	}
}

func TestGetPatternSuggestionPython(t *testing.T) {
	tests := []struct {
		query           string
		expectedPattern string
		expectedLang    string
	}{
		{"python function", "def $NAME($ARGS):", "py"},
		{"python async function", "async def $NAME($ARGS):", "py"},
		{"py class", "class $NAME:", "py"},
	}

	for _, tt := range tests {
		suggestion := GetPatternSuggestion(tt.query)
		if suggestion.Pattern != tt.expectedPattern {
			t.Errorf("Query '%s': expected pattern '%s', got '%s'", tt.query, tt.expectedPattern, suggestion.Pattern)
		}
		if suggestion.Lang != tt.expectedLang {
			t.Errorf("Query '%s': expected lang '%s', got '%s'", tt.query, tt.expectedLang, suggestion.Lang)
		}
	}
}

func TestGetPatternSuggestionRust(t *testing.T) {
	tests := []struct {
		query           string
		expectedPattern string
		expectedLang    string
	}{
		{"rust function", "fn $NAME($ARGS)", "rs"},
		{"rust result", "fn $NAME($ARGS) -> Result<$T, $E>", "rs"},
		{"rust struct", "struct $NAME { $$$ }", "rs"},
		{"rust impl", "impl $TYPE { $$$ }", "rs"},
	}

	for _, tt := range tests {
		suggestion := GetPatternSuggestion(tt.query)
		if suggestion.Pattern != tt.expectedPattern {
			t.Errorf("Query '%s': expected pattern '%s', got '%s'", tt.query, tt.expectedPattern, suggestion.Pattern)
		}
		if suggestion.Lang != tt.expectedLang {
			t.Errorf("Query '%s': expected lang '%s', got '%s'", tt.query, tt.expectedLang, suggestion.Lang)
		}
	}
}

func TestGetPatternSuggestionUnknown(t *testing.T) {
	suggestion := GetPatternSuggestion("something random")
	if suggestion.Pattern != "" {
		t.Errorf("Expected empty pattern for unknown query, got '%s'", suggestion.Pattern)
	}
	if suggestion.Hint == "" {
		t.Error("Expected hint for unknown query")
	}
}
