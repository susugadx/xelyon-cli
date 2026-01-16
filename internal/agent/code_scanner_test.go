package agent

import (
	"testing"
)

func TestExtractDefinitionNames_Japanese(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string // Expected names to be in the result
	}{
		{
			name:     "Japanese add pattern",
			input:    "handleUser を追加して",
			contains: []string{"handleUser"},
		},
		{
			name:     "Japanese implement pattern",
			input:    "NewClient を実装する",
			contains: []string{"NewClient"},
		},
		{
			name:     "Japanese create pattern with brackets",
			input:    "「ProcessData」を作成してください",
			contains: []string{"ProcessData"},
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
					t.Errorf("ExtractDefinitionNames() did not find %q in result %v", expected, result)
				}
			}
		})
	}
}

func TestExtractDefinitionNames_English(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:     "add function pattern",
			input:    "add handleRequest function",
			contains: []string{"handleRequest"},
		},
		{
			name:     "create pattern",
			input:    "create UserService",
			contains: []string{"UserService"},
		},
		{
			name:     "implement pattern",
			input:    "implement ProcessCommand",
			contains: []string{"ProcessCommand"},
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
					t.Errorf("ExtractDefinitionNames() did not find %q in result %v", expected, result)
				}
			}
		})
	}
}

func TestExtractDefinitionNames_FuncType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:     "func pattern",
			input:    "func HandleRequest should be added",
			contains: []string{"HandleRequest"},
		},
		{
			name:     "type pattern",
			input:    "type UserConfig needs to be created",
			contains: []string{"UserConfig"},
		},
		{
			name:     "Command suffix",
			input:    "please add RefactorCommand",
			contains: []string{"RefactorCommand"},
		},
		{
			name:     "Handler suffix",
			input:    "create ErrorHandler for this",
			contains: []string{"ErrorHandler"},
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
					t.Errorf("ExtractDefinitionNames() did not find %q in result %v", expected, result)
				}
			}
		})
	}
}

func TestExtractDefinitionNames_Empty(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "no patterns",
			input: "hello world",
		},
		{
			name:  "question only",
			input: "どうすればいいですか？",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractDefinitionNames(tt.input)

			// Should return empty or nil slice
			if len(result) > 0 {
				// Check if any result is longer than 2 chars (the filter condition)
				hasValidName := false
				for _, name := range result {
					if len(name) > 2 {
						hasValidName = true
						break
					}
				}
				if hasValidName {
					t.Errorf("ExtractDefinitionNames() expected empty, got %v", result)
				}
			}
		})
	}
}

func TestExtractDefinitionNames_Deduplication(t *testing.T) {
	// Input that would match multiple patterns for the same name
	input := "add handleCommand をhandleCommand として実装して handleCommand"

	result := ExtractDefinitionNames(input)

	// Count occurrences
	count := 0
	for _, name := range result {
		if name == "handleCommand" {
			count++
		}
	}

	if count > 1 {
		t.Errorf("ExtractDefinitionNames() should deduplicate: found %d occurrences of 'handleCommand'", count)
	}
}

func TestIsCodeModificationRequest_Japanese(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "追加",
			input: "関数を追加して",
			want:  true,
		},
		{
			name:  "実装",
			input: "新しい機能を実装する",
			want:  true,
		},
		{
			name:  "作成",
			input: "ファイルを作成してください",
			want:  true,
		},
		{
			name:  "修正",
			input: "バグを修正して",
			want:  true,
		},
		{
			name:  "削除",
			input: "この行を削除して",
			want:  true,
		},
		{
			name:  "リファクタ",
			input: "コードをリファクタリングして",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsCodeModificationRequest(tt.input)
			if got != tt.want {
				t.Errorf("IsCodeModificationRequest(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsCodeModificationRequest_English(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "add",
			input: "add a new function",
			want:  true,
		},
		{
			name:  "create",
			input: "create a new file",
			want:  true,
		},
		{
			name:  "implement",
			input: "implement this feature",
			want:  true,
		},
		{
			name:  "modify",
			input: "modify the config",
			want:  true,
		},
		{
			name:  "delete",
			input: "delete this file",
			want:  true,
		},
		{
			name:  "refactor",
			input: "refactor the code",
			want:  true,
		},
		{
			name:  "fix",
			input: "fix the bug",
			want:  true,
		},
		{
			name:  "write",
			input: "write a test",
			want:  true,
		},
		{
			name:  "case insensitive - ADD",
			input: "ADD new handler",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsCodeModificationRequest(tt.input)
			if got != tt.want {
				t.Errorf("IsCodeModificationRequest(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsCodeModificationRequest_NoMatch(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "question",
			input: "what is this function for?",
		},
		{
			name:  "explain",
			input: "explain this code",
		},
		{
			name:  "read",
			input: "read the file",
		},
		{
			name:  "show",
			input: "show me the logs",
		},
		{
			name:  "Japanese question",
			input: "これはなんですか？",
		},
		{
			name:  "empty",
			input: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsCodeModificationRequest(tt.input)
			if got {
				t.Errorf("IsCodeModificationRequest(%q) = true, want false", tt.input)
			}
		})
	}
}

func TestGetFileLanguage_AllExtensions(t *testing.T) {
	tests := []struct {
		filePath string
		want     string
	}{
		// Go
		{"main.go", "go"},
		{"internal/agent/agent.go", "go"},
		{"TEST.GO", "go"}, // case insensitive

		// TypeScript
		{"app.ts", "typescript"},
		{"component.tsx", "typescript"},

		// JavaScript
		{"script.js", "javascript"},
		{"component.jsx", "javascript"},

		// Python
		{"script.py", "python"},

		// Rust
		{"main.rs", "rust"},

		// Java
		{"Main.java", "java"},

		// Ruby
		{"script.rb", "ruby"},

		// PHP
		{"index.php", "php"},

		// Unknown
		{"file.txt", "unknown"},
		{"file.md", "unknown"},
		{"file.yaml", "unknown"},
		{"noextension", "unknown"},
		{"", "unknown"},
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
