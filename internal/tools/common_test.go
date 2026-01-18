package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/testutil"
)

func TestParseToolCall_Valid(t *testing.T) {
	tests := []struct {
		name     string
		response string
		wantTool string
	}{
		{
			name:     "simple tool call",
			response: `{"tool": "read_file", "args": {"path": "test.txt"}}`,
			wantTool: "read_file",
		},
		{
			name:     "tool call with space after brace",
			response: `{ "tool": "write_file", "args": {"path": "out.txt", "content": "hello"}}`,
			wantTool: "write_file",
		},
		{
			name:     "tool call in text",
			response: `Here is the tool call: {"tool": "delete_file", "args": {"path": "old.txt"}} Done.`,
			wantTool: "delete_file",
		},
		{
			name:     "nested json",
			response: `{"tool": "bash", "args": {"command": "echo {\"key\": \"value\"}"}}`,
			wantTool: "bash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseToolCall(tt.response)
			if got == nil {
				t.Fatal("ParseToolCall() returned nil")
			}
			if got.Tool != tt.wantTool {
				t.Errorf("ParseToolCall() tool = %v, want %v", got.Tool, tt.wantTool)
			}
		})
	}
}

func TestParseToolCall_Invalid(t *testing.T) {
	tests := []struct {
		name     string
		response string
	}{
		{
			name:     "no tool call",
			response: "This is just plain text without any tool call.",
		},
		{
			name:     "incomplete json",
			response: `{"tool": "read_file", "args": {`,
		},
		{
			name:     "invalid json",
			response: `{"tool": "read_file" invalid}`,
		},
		{
			name:     "empty string",
			response: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseToolCall(tt.response)
			if got != nil {
				t.Errorf("ParseToolCall() should return nil for invalid input, got %+v", got)
			}
		})
	}
}

func TestParseToolCalls_Multiple(t *testing.T) {
	tests := []struct {
		name      string
		response  string
		wantCount int
		wantTools []string
	}{
		{
			name:      "single tool",
			response:  `{"tool": "read_file", "args": {"path": "test.txt"}}`,
			wantCount: 1,
			wantTools: []string{"read_file"},
		},
		{
			name: "two tools",
			response: `First I'll read the file {"tool": "read_file", "args": {"path": "a.txt"}}
then write {"tool": "write_file", "args": {"path": "b.txt", "content": "hello"}}`,
			wantCount: 2,
			wantTools: []string{"read_file", "write_file"},
		},
		{
			name: "three tools",
			response: `{"tool": "bash", "args": {"command": "ls"}}
{"tool": "read_file", "args": {"path": "a.txt"}}
{"tool": "delete_file", "args": {"path": "old.txt"}}`,
			wantCount: 3,
			wantTools: []string{"bash", "read_file", "delete_file"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseToolCalls(tt.response)
			if len(got) != tt.wantCount {
				t.Fatalf("ParseToolCalls() returned %d tools, want %d", len(got), tt.wantCount)
			}
			for i, tc := range got {
				if tc.Tool != tt.wantTools[i] {
					t.Errorf("ParseToolCalls()[%d].Tool = %v, want %v", i, tc.Tool, tt.wantTools[i])
				}
			}
		})
	}
}

func TestParseToolCalls_CodeBlockExclusion(t *testing.T) {
	tests := []struct {
		name      string
		response  string
		wantCount int
		wantTools []string
	}{
		{
			name: "tool in code block should be excluded",
			response: "Here's an example:\n```json\n{\"tool\": \"read_file\", \"args\": {}}\n```\n" +
				"But this is real: {\"tool\": \"write_file\", \"args\": {\"path\": \"out.txt\"}}",
			wantCount: 1,
			wantTools: []string{"write_file"},
		},
		{
			name: "multiple code blocks",
			response: "```\n{\"tool\": \"a\"}\n```\n```\n{\"tool\": \"b\"}\n```\n" +
				"{\"tool\": \"real\", \"args\": {}}",
			wantCount: 1,
			wantTools: []string{"real"},
		},
		{
			name:      "all in code block",
			response:  "```\n{\"tool\": \"read_file\", \"args\": {}}\n```",
			wantCount: 0,
			wantTools: []string{},
		},
		{
			name: "code block with language specifier",
			response: "```javascript\n{\"tool\": \"fake\"}\n```\n" +
				"{\"tool\": \"actual\", \"args\": {}}",
			wantCount: 1,
			wantTools: []string{"actual"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseToolCalls(tt.response)
			if len(got) != tt.wantCount {
				t.Fatalf("ParseToolCalls() returned %d tools, want %d", len(got), tt.wantCount)
			}
			for i, tc := range got {
				if tc.Tool != tt.wantTools[i] {
					t.Errorf("ParseToolCalls()[%d].Tool = %v, want %v", i, tc.Tool, tt.wantTools[i])
				}
			}
		})
	}
}

func TestFindCodeBlockRanges(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		wantRanges int
	}{
		{
			name:       "no code blocks",
			text:       "plain text without code blocks",
			wantRanges: 0,
		},
		{
			name:       "single code block",
			text:       "text ```code``` more",
			wantRanges: 1,
		},
		{
			name:       "two code blocks",
			text:       "```a``` middle ```b```",
			wantRanges: 2,
		},
		{
			name:       "unclosed code block",
			text:       "```start of code block",
			wantRanges: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findCodeBlockRanges(tt.text)
			if len(got) != tt.wantRanges {
				t.Errorf("findCodeBlockRanges() returned %d ranges, want %d", len(got), tt.wantRanges)
			}
		})
	}
}

// TestParseToolCalls_UnclosedCodeBlock はコードブロックが閉じていない場合の挙動をテスト
// この問題が起こりうる: AIがコードブロックを開始して閉じ忘れた場合、
// その後のすべてのツールJSONがコードブロック内として扱われてスキップされる
func TestParseToolCalls_UnclosedCodeBlock(t *testing.T) {
	tests := []struct {
		name      string
		response  string
		wantCount int
		wantTools []string
	}{
		{
			name: "unclosed code block before tool (CURRENT BEHAVIOR - may cause issues)",
			// 現在の実装では、閉じていないコードブロック以降がすべてスキップされる
			response:  "```json\nsome example\n{\"tool\": \"read_file\", \"args\": {}}",
			wantCount: 0, // 現在の挙動: コードブロック内なのでスキップ
			wantTools: []string{},
		},
		{
			name:      "tool after properly closed code block",
			response:  "```json\nexample\n```\n{\"tool\": \"read_file\", \"args\": {}}",
			wantCount: 1,
			wantTools: []string{"read_file"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseToolCalls(tt.response)
			if len(got) != tt.wantCount {
				t.Fatalf("ParseToolCalls() returned %d tools, want %d", len(got), tt.wantCount)
			}
			for i, tc := range got {
				if tc.Tool != tt.wantTools[i] {
					t.Errorf("ParseToolCalls()[%d].Tool = %v, want %v", i, tc.Tool, tt.wantTools[i])
				}
			}
		})
	}
}

func TestCreateBackup_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "new.txt")

	// ファイルが存在しない場合、バックアップは作成されない
	backupPath, err := createBackup(testFile)
	if err != nil {
		t.Fatalf("createBackup() error = %v", err)
	}

	if backupPath != "" {
		t.Errorf("createBackup() should return empty string for non-existent file, got %v", backupPath)
	}
}

func TestCreateBackup_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "existing.txt")
	testContent := "original content"

	// ファイル作成
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// バックアップ作成
	backupPath, err := createBackup(testFile)
	if err != nil {
		t.Fatalf("createBackup() error = %v", err)
	}

	if backupPath == "" {
		t.Fatal("createBackup() should return backup path for existing file")
	}

	// バックアップファイルが存在することを確認
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Errorf("Backup file not created: %v", backupPath)
	}

	// バックアップの内容を確認
	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("Failed to read backup file: %v", err)
	}

	if string(backupContent) != testContent {
		t.Errorf("Backup content = %v, want %v", string(backupContent), testContent)
	}

	// バックアップファイル名のフォーマット確認
	if !strings.Contains(backupPath, ".bak.") {
		t.Error("Backup path should contain '.bak.' timestamp")
	}
}

func TestCleanupOldBackups(t *testing.T) {
	tmpDir := testutil.SetupTempHome(t)
	testFile := filepath.Join(tmpDir, "test.txt")

	// デフォルト設定作成（MaxGenerations=5）
	cfg := config.DefaultConfig()
	cfg.Backup.MaxGenerations = 3 // テスト用に3世代に設定
	config.SetGlobalConfig(cfg)
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// 5つのバックアップファイルを作成
	for i := 0; i < 5; i++ {
		backupPath := filepath.Join(tmpDir, "test.txt.bak.2024010"+string(rune('1'+i))+"_120000")
		if err := os.WriteFile(backupPath, []byte("backup"), 0644); err != nil {
			t.Fatalf("Failed to create backup file: %v", err)
		}
	}

	// 元ファイル作成
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// クリーンアップ実行
	err := cleanupOldBackups(testFile)
	if err != nil {
		t.Fatalf("cleanupOldBackups() error = %v", err)
	}

	// バックアップファイル数を確認（3つ残る）
	matches, err := filepath.Glob(filepath.Join(tmpDir, "test.txt.bak.*"))
	if err != nil {
		t.Fatalf("Failed to glob backup files: %v", err)
	}

	if len(matches) != 3 {
		t.Errorf("cleanupOldBackups() left %d backups, want 3", len(matches))
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "short string",
			input:  "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "exact length",
			input:  "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "truncate needed",
			input:  "hello world",
			maxLen: 8,
			want:   "hello wo...",
		},
		{
			name:   "empty string",
			input:  "",
			maxLen: 5,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeLeadingWhitespace(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "tabs to spaces",
			input: "\tfunc main() {\n\t\treturn\n\t}",
			want:  "func main() {\nreturn\n}",
		},
		{
			name:  "leading spaces removed",
			input: "    line1\n        line2",
			want:  "line1\nline2",
		},
		{
			name:  "preserve internal spaces",
			input: "hello  world",
			want:  "hello  world",
		},
		{
			name:  "mixed tabs and spaces",
			input: "\t    hello\n\tworld",
			want:  "hello\nworld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeLeadingWhitespace(tt.input)
			if got != tt.want {
				t.Errorf("normalizeLeadingWhitespace() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDetectTestFramework(t *testing.T) {
	tests := []struct {
		name          string
		createFiles   []string
		wantFramework string
		wantCommand   string
	}{
		{
			name:          "Go project",
			createFiles:   []string{"go.mod"},
			wantFramework: "Go",
			wantCommand:   "go test ./...",
		},
		{
			name:          "JavaScript (npm)",
			createFiles:   []string{"package.json"},
			wantFramework: "JavaScript (npm)",
			wantCommand:   "npm test",
		},
		{
			name:          "JavaScript (yarn)",
			createFiles:   []string{"package.json", "yarn.lock"},
			wantFramework: "JavaScript (yarn)",
			wantCommand:   "yarn test",
		},
		{
			name:          "Python (pytest.ini)",
			createFiles:   []string{"pytest.ini"},
			wantFramework: "Python (pytest)",
			wantCommand:   "pytest",
		},
		{
			name:          "Python (setup.py)",
			createFiles:   []string{"setup.py"},
			wantFramework: "Python (pytest)",
			wantCommand:   "pytest",
		},
		{
			name:          "Rust",
			createFiles:   []string{"Cargo.toml"},
			wantFramework: "Rust",
			wantCommand:   "cargo test",
		},
		{
			name:          "no framework",
			createFiles:   []string{},
			wantFramework: "",
			wantCommand:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// テストファイル作成
			for _, file := range tt.createFiles {
				filePath := filepath.Join(tmpDir, file)
				if err := os.WriteFile(filePath, []byte(""), 0644); err != nil {
					t.Fatalf("Failed to create test file %s: %v", file, err)
				}
			}

			framework, command := detectTestFramework(tmpDir)
			if framework != tt.wantFramework {
				t.Errorf("detectTestFramework() framework = %v, want %v", framework, tt.wantFramework)
			}
			if command != tt.wantCommand {
				t.Errorf("detectTestFramework() command = %v, want %v", command, tt.wantCommand)
			}
		})
	}
}

func TestDetectTestFramework_ParentDirectory(t *testing.T) {
	// Test that framework detection works when config file is in parent directory
	tmpDir := t.TempDir()

	// Create go.mod in root
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test"), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create subdirectory
	subDir := filepath.Join(tmpDir, "internal", "cache")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Test from subdirectory - should find go.mod in parent
	framework, command := detectTestFramework(subDir)
	if framework != "Go" {
		t.Errorf("detectTestFramework() from subdir: framework = %v, want Go", framework)
	}
	if command != "go test ./..." {
		t.Errorf("detectTestFramework() from subdir: command = %v, want 'go test ./...'", command)
	}
}

func TestFindProjectRoot(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod in root
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test"), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create nested subdirectory
	deepDir := filepath.Join(tmpDir, "internal", "tools", "helpers")
	if err := os.MkdirAll(deepDir, 0755); err != nil {
		t.Fatalf("Failed to create deep subdirectory: %v", err)
	}

	tests := []struct {
		name       string
		startPath  string
		configFile string
		wantRoot   string
	}{
		{
			name:       "find from root",
			startPath:  tmpDir,
			configFile: "go.mod",
			wantRoot:   tmpDir,
		},
		{
			name:       "find from subdirectory",
			startPath:  filepath.Join(tmpDir, "internal"),
			configFile: "go.mod",
			wantRoot:   tmpDir,
		},
		{
			name:       "find from deep subdirectory",
			startPath:  deepDir,
			configFile: "go.mod",
			wantRoot:   tmpDir,
		},
		{
			name:       "not found",
			startPath:  deepDir,
			configFile: "nonexistent.file",
			wantRoot:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findProjectRoot(tt.startPath, tt.configFile)
			if got != tt.wantRoot {
				t.Errorf("findProjectRoot() = %v, want %v", got, tt.wantRoot)
			}
		})
	}
}

func TestDetectFormatter(t *testing.T) {
	tests := []struct {
		name          string
		createFiles   []string
		wantFormatter string
		wantCommand   string
	}{
		{
			name:          "Go project",
			createFiles:   []string{"go.mod"},
			wantFormatter: "gofmt",
			wantCommand:   "go fmt ./...",
		},
		{
			name:          "Prettier (.prettierrc)",
			createFiles:   []string{".prettierrc"},
			wantFormatter: "prettier",
			wantCommand:   "prettier --write .",
		},
		{
			name:          "Prettier (.prettierrc.json)",
			createFiles:   []string{".prettierrc.json"},
			wantFormatter: "prettier",
			wantCommand:   "prettier --write .",
		},
		{
			name:          "Rust",
			createFiles:   []string{"Cargo.toml"},
			wantFormatter: "rustfmt",
			wantCommand:   "cargo fmt",
		},
		{
			name:          "no formatter",
			createFiles:   []string{},
			wantFormatter: "",
			wantCommand:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// テストファイル作成
			for _, file := range tt.createFiles {
				filePath := filepath.Join(tmpDir, file)
				if err := os.WriteFile(filePath, []byte(""), 0644); err != nil {
					t.Fatalf("Failed to create test file %s: %v", file, err)
				}
			}

			formatter, command := detectFormatter(tmpDir)
			if formatter != tt.wantFormatter {
				t.Errorf("detectFormatter() formatter = %v, want %v", formatter, tt.wantFormatter)
			}
			if command != tt.wantCommand {
				t.Errorf("detectFormatter() command = %v, want %v", command, tt.wantCommand)
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "exists.txt")

	// ファイル作成
	if err := os.WriteFile(existingFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if !fileExists(existingFile) {
		t.Error("fileExists() should return true for existing file")
	}

	if fileExists(filepath.Join(tmpDir, "notexist.txt")) {
		t.Error("fileExists() should return false for non-existent file")
	}
}

func TestCommandExists(t *testing.T) {
	// 一般的に存在するコマンド
	if !commandExists("ls") && !commandExists("dir") {
		t.Error("commandExists() should return true for ls/dir")
	}

	// 存在しないコマンド
	if commandExists("this_command_definitely_does_not_exist_12345") {
		t.Error("commandExists() should return false for non-existent command")
	}
}

func TestHasGlobMatches(t *testing.T) {
	tmpDir := t.TempDir()

	// *.txtファイルを作成
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	pattern := filepath.Join(tmpDir, "*.txt")
	if !hasGlobMatches(pattern) {
		t.Error("hasGlobMatches() should return true for matching pattern")
	}

	noMatchPattern := filepath.Join(tmpDir, "*.xyz")
	if hasGlobMatches(noMatchPattern) {
		t.Error("hasGlobMatches() should return false for non-matching pattern")
	}
}

func TestDetectLinter(t *testing.T) {
	tests := []struct {
		name        string
		createFiles []string
		wantName    string // 空文字を許容（環境依存）
	}{
		{
			name:        "Go project",
			createFiles: []string{"go.mod"},
			wantName:    "", // golangci-lint/go vet は環境依存
		},
		{
			name:        "JavaScript project",
			createFiles: []string{"package.json", ".eslintrc"},
			wantName:    "eslint",
		},
		{
			name:        "Rust project",
			createFiles: []string{"Cargo.toml"},
			wantName:    "clippy",
		},
		{
			name:        "no linter",
			createFiles: []string{},
			wantName:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// テストファイル作成
			for _, file := range tt.createFiles {
				filePath := filepath.Join(tmpDir, file)
				if err := os.WriteFile(filePath, []byte(""), 0644); err != nil {
					t.Fatalf("Failed to create test file %s: %v", file, err)
				}
			}

			linterName, checkCmd, fixCmd := detectLinter(tmpDir)

			// wantNameが空でない場合のみ検証（環境依存を回避）
			if tt.wantName != "" && linterName != tt.wantName {
				t.Errorf("detectLinter() name = %v, want %v", linterName, tt.wantName)
			}

			// linterが検出された場合、checkCmdは空でないはず
			if linterName != "" && checkCmd == "" {
				t.Error("detectLinter() should return non-empty checkCmd when linter is detected")
			}

			// fixCmdは空でも問題ない（go vet等）
			_ = fixCmd
		})
	}
}

func TestMinMax(t *testing.T) {
	tests := []struct {
		a       int
		b       int
		wantMin int
		wantMax int
	}{
		{a: 5, b: 10, wantMin: 5, wantMax: 10},
		{a: 10, b: 5, wantMin: 5, wantMax: 10},
		{a: 0, b: 0, wantMin: 0, wantMax: 0},
		{a: -5, b: 5, wantMin: -5, wantMax: 5},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			gotMin := min(tt.a, tt.b)
			gotMax := max(tt.a, tt.b)

			if gotMin != tt.wantMin {
				t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, gotMin, tt.wantMin)
			}
			if gotMax != tt.wantMax {
				t.Errorf("max(%d, %d) = %d, want %d", tt.a, tt.b, gotMax, tt.wantMax)
			}
		})
	}
}
