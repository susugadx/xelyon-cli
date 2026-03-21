package agent

import (
	"testing"
)

// --- ExtractDefinitionNames additional tests ---

func TestExtractDefinitionNames_JapaneseAddPatterns(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:     "追加 + name",
			input:    "handleConfig を追加する",
			contains: []string{"handleConfig"},
		},
		{
			name:     "実装 + name",
			input:    "ProcessTask を実装して",
			contains: []string{"ProcessTask"},
		},
		{
			name:     "作成 + name with brackets",
			input:    "「RunServer」を作成",
			contains: []string{"RunServer"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractDefinitionNames(tt.input)
			for _, expected := range tt.contains {
				found := false
				for _, name := range result {
					if name == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("ExtractDefinitionNames(%q) did not find %q in %v", tt.input, expected, result)
				}
			}
		})
	}
}

func TestExtractDefinitionNames_EnglishPatterns(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:     "func keyword",
			input:    "func ParseConfig should handle YAML",
			contains: []string{"ParseConfig"},
		},
		{
			name:     "type keyword",
			input:    "type ServerConfig needs these fields",
			contains: []string{"ServerConfig"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractDefinitionNames(tt.input)
			for _, expected := range tt.contains {
				found := false
				for _, name := range result {
					if name == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("ExtractDefinitionNames(%q) did not find %q in %v", tt.input, expected, result)
				}
			}
		})
	}
}

func TestExtractDefinitionNames_BracketPatterns(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:     "Japanese brackets 「」",
			input:    "「ConfigHandler」を追加してください",
			contains: []string{"ConfigHandler"},
		},
		{
			name:     "Japanese brackets 『』",
			input:    "『BuildSystem』を実装",
			contains: []string{"BuildSystem"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractDefinitionNames(tt.input)
			for _, expected := range tt.contains {
				found := false
				for _, name := range result {
					if name == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("ExtractDefinitionNames(%q) did not find %q in %v", tt.input, expected, result)
				}
			}
		})
	}
}

// --- IsCodeModificationRequest additional tests ---

func TestIsCodeModificationRequest_JapaneseKeywords(t *testing.T) {
	positives := []struct {
		name  string
		input string
	}{
		{name: "追加して", input: "新しいメソッドを追加して"},
		{name: "実装して", input: "この機能を実装してください"},
		{name: "書き換え", input: "コードを書き換えて"},
		{name: "変更", input: "設定を変更する"},
		{name: "定義", input: "新しい型を定義して"},
	}

	for _, tt := range positives {
		t.Run(tt.name, func(t *testing.T) {
			if !IsCodeModificationRequest(tt.input) {
				t.Errorf("IsCodeModificationRequest(%q) = false, want true", tt.input)
			}
		})
	}
}

func TestIsCodeModificationRequest_EnglishKeywords(t *testing.T) {
	positives := []struct {
		name  string
		input string
	}{
		{name: "add", input: "add a new handler"},
		{name: "create", input: "create the struct"},
		{name: "remove", input: "remove dead code"},
		{name: "update", input: "update the config parser"},
		{name: "write", input: "write unit tests for this"},
	}

	for _, tt := range positives {
		t.Run(tt.name, func(t *testing.T) {
			if !IsCodeModificationRequest(tt.input) {
				t.Errorf("IsCodeModificationRequest(%q) = false, want true", tt.input)
			}
		})
	}
}

func TestIsCodeModificationRequest_NegativeCases(t *testing.T) {
	negatives := []struct {
		name  string
		input string
	}{
		{name: "explain request", input: "explain how this works"},
		{name: "read request", input: "read the log files"},
		{name: "question", input: "what does this function do?"},
		{name: "empty", input: ""},
		{name: "Japanese question", input: "この関数は何をしていますか？"},
		{name: "search request", input: "search for the config file"},
	}

	for _, tt := range negatives {
		t.Run(tt.name, func(t *testing.T) {
			if IsCodeModificationRequest(tt.input) {
				t.Errorf("IsCodeModificationRequest(%q) = true, want false", tt.input)
			}
		})
	}
}

// --- GetFileLanguage additional tests ---

func TestGetFileLanguage_ExtensionMapping(t *testing.T) {
	tests := []struct {
		filePath string
		want     string
	}{
		{"main.go", "go"},
		{"app.py", "python"},
		{"index.js", "javascript"},
		{"component.jsx", "javascript"},
		{"app.ts", "typescript"},
		{"component.tsx", "typescript"},
		{"main.rs", "rust"},
		{"Main.java", "java"},
		{"script.rb", "ruby"},
		{"index.php", "php"},
		{"file.txt", "unknown"},
		{"", "unknown"},
		{"noext", "unknown"},
		{"path/to/deep/file.go", "go"},
		{"MAIN.GO", "go"}, // case insensitive via filepath.Ext + ToLower
	}

	for _, tt := range tests {
		t.Run(tt.filePath, func(t *testing.T) {
			got := GetFileLanguage(tt.filePath)
			if got != tt.want {
				t.Errorf("GetFileLanguage(%q) = %q, want %q", tt.filePath, got, tt.want)
			}
		})
	}
}
