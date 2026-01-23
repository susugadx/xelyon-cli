package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/repomap"
)

func TestDetectTechStack(t *testing.T) {
	// テスト用一時ディレクトリ作成
	tmpDir, err := os.MkdirTemp("", "xelyon-init-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name     string
		files    []string
		expected []string
	}{
		{
			name:     "Go project",
			files:    []string{"go.mod"},
			expected: []string{"Go"},
		},
		{
			name:     "Node.js project",
			files:    []string{"package.json"},
			expected: []string{"Node.js"},
		},
		{
			name:     "TypeScript project",
			files:    []string{"package.json", "tsconfig.json"},
			expected: []string{"Node.js", "TypeScript"},
		},
		{
			name:     "Python project with requirements",
			files:    []string{"requirements.txt"},
			expected: []string{"Python"},
		},
		{
			name:     "Python project with pyproject",
			files:    []string{"pyproject.toml"},
			expected: []string{"Python"},
		},
		{
			name:     "Rust project",
			files:    []string{"Cargo.toml"},
			expected: []string{"Rust"},
		},
		{
			name:     "Docker project",
			files:    []string{"Dockerfile"},
			expected: []string{"Docker"},
		},
		{
			name:     "Next.js project",
			files:    []string{"package.json", "next.config.js"},
			expected: []string{"Node.js", "Next.js"},
		},
		{
			name:     "Empty project",
			files:    []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// テスト用サブディレクトリ作成
			testDir := filepath.Join(tmpDir, strings.ReplaceAll(tt.name, " ", "_"))
			if err := os.MkdirAll(testDir, 0755); err != nil {
				t.Fatalf("Failed to create test dir: %v", err)
			}

			// ファイル作成
			for _, f := range tt.files {
				filePath := filepath.Join(testDir, f)
				if err := os.WriteFile(filePath, []byte(""), 0644); err != nil {
					t.Fatalf("Failed to create file %s: %v", f, err)
				}
			}

			// 検出実行
			result := detectTechStack(testDir)

			// 検証
			if len(result) != len(tt.expected) {
				t.Errorf("detectTechStack() = %v, want %v", result, tt.expected)
				return
			}

			for i, exp := range tt.expected {
				if result[i] != exp {
					t.Errorf("detectTechStack()[%d] = %s, want %s", i, result[i], exp)
				}
			}
		})
	}
}

func TestExtToLanguage(t *testing.T) {
	tests := []struct {
		ext      string
		expected string
	}{
		// Go
		{".go", "Go"},
		// JavaScript variants
		{".js", "JavaScript"},
		{".mjs", "JavaScript"},
		{".cjs", "JavaScript"},
		// TypeScript
		{".ts", "TypeScript"},
		{".tsx", "TypeScript"},
		// JSX
		{".jsx", "React/JSX"},
		// Python
		{".py", "Python"},
		// Rust
		{".rs", "Rust"},
		// Java
		{".java", "Java"},
		// Kotlin
		{".kt", "Kotlin"},
		{".kts", "Kotlin"},
		// Ruby
		{".rb", "Ruby"},
		// PHP
		{".php", "PHP"},
		// C
		{".c", "C"},
		{".h", "C"},
		// C++
		{".cpp", "C++"},
		{".hpp", "C++"},
		{".cc", "C++"},
		{".cxx", "C++"},
		// C#
		{".cs", "C#"},
		// Swift
		{".swift", "Swift"},
		// Vue
		{".vue", "Vue"},
		// Svelte
		{".svelte", "Svelte"},
		// Unknown
		{".unknown", ""},
		{"", ""},
		{".xyz", ""},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			result := extToLanguage(tt.ext)
			if result != tt.expected {
				t.Errorf("extToLanguage(%s) = %s, want %s", tt.ext, result, tt.expected)
			}
		})
	}
}

func TestGenerateXELYONMD(t *testing.T) {
	config := &InitConfig{
		ProjectName: "test-project",
		Overview:    "This is a test project for testing purposes.",
		TechStack:   []string{"Go", "Docker"},
		CodingRules: "Use gofmt for formatting",
		RepoMapData: nil,
	}

	result := generateXELYONMD(config)

	// 必須項目が含まれているか確認
	checks := []string{
		"# test-project プロジェクト設定",
		"## 概要",
		"This is a test project",
		"## 技術スタック",
		"**Go**",
		"**Docker**",
		"## 開発ルール",
		"Use gofmt for formatting",
	}

	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("generateXELYONMD() missing: %s", check)
		}
	}
}

func TestGenerateXELYONMD_WithRepoMap(t *testing.T) {
	// RepoMap付きのテスト
	rm := &repomap.RepoMap{
		RootPath: "/test/project",
		Files: []*repomap.FileSymbols{
			{
				Path: "/test/project/main.go",
				Symbols: []repomap.Symbol{
					{Name: "main", Kind: "function", Signature: "func main()", Line: 10},
				},
			},
		},
	}

	config := &InitConfig{
		ProjectName: "test-project",
		Overview:    "Test project",
		TechStack:   []string{"Go"},
		RepoMapData: rm,
	}

	result := generateXELYONMD(config)

	// コードマップセクションが含まれているか確認
	if !strings.Contains(result, "## コードマップ") {
		t.Error("generateXELYONMD() missing code map section")
	}

	if !strings.Contains(result, "main.go") {
		t.Error("generateXELYONMD() missing file reference")
	}
}

func TestGenerateXELYONMD_EmptyConfig(t *testing.T) {
	config := &InitConfig{
		ProjectName: "empty-project",
	}

	result := generateXELYONMD(config)

	// 最低限のテンプレートが生成されるか
	if !strings.Contains(result, "# empty-project プロジェクト設定") {
		t.Error("generateXELYONMD() missing title")
	}

	if !strings.Contains(result, "## 概要") {
		t.Error("generateXELYONMD() missing overview section")
	}

	// プレースホルダーテキストが含まれる
	if !strings.Contains(result, "(プロジェクトの説明をここに記載)") {
		t.Error("generateXELYONMD() missing overview placeholder")
	}
}

func TestGenerateDirectoryTree(t *testing.T) {
	rm := &repomap.RepoMap{
		RootPath: "/test/project",
		Files: []*repomap.FileSymbols{
			{Path: "/test/project/cmd/main.go"},
			{Path: "/test/project/internal/agent/agent.go"},
			{Path: "/test/project/internal/api/client.go"},
		},
	}

	config := &InitConfig{
		ProjectName: "test-project",
		RepoMapData: rm,
	}

	result := generateDirectoryTree(config)

	// ディレクトリが含まれているか
	if !strings.Contains(result, "test-project/") {
		t.Error("generateDirectoryTree() missing project name")
	}

	if !strings.Contains(result, "cmd/") {
		t.Error("generateDirectoryTree() missing cmd directory")
	}

	if !strings.Contains(result, "internal/") {
		t.Error("generateDirectoryTree() missing internal directory")
	}
}

func TestFileExists(t *testing.T) {
	// 存在するファイル
	tmpFile, err := os.CreateTemp("", "xelyon-test")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	if !fileExists(tmpFile.Name()) {
		t.Error("fileExists() = false for existing file")
	}

	// 存在しないファイル
	if fileExists("/nonexistent/file/path") {
		t.Error("fileExists() = true for nonexistent file")
	}
}
