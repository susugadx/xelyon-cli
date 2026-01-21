package tools

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/lsp"
)

func TestLSPReferencesTool_Name(t *testing.T) {
	tool := &LSPReferencesTool{}
	if got := tool.Name(); got != "lsp_references" {
		t.Errorf("Name() = %q, want %q", got, "lsp_references")
	}
}

func TestLSPDefinitionTool_Name(t *testing.T) {
	tool := &LSPDefinitionTool{}
	if got := tool.Name(); got != "lsp_definition" {
		t.Errorf("Name() = %q, want %q", got, "lsp_definition")
	}
}

func TestLSPHoverTool_Name(t *testing.T) {
	tool := &LSPHoverTool{}
	if got := tool.Name(); got != "lsp_hover" {
		t.Errorf("Name() = %q, want %q", got, "lsp_hover")
	}
}

func TestLSPReferencesTool_NoClient(t *testing.T) {
	// LSPClient が nil の場合のテスト
	originalClient := LSPClient
	LSPClient = nil
	defer func() { LSPClient = originalClient }()

	tool := &LSPReferencesTool{}
	output, change, err := tool.Run(map[string]string{
		"path":      "test.go",
		"line":      "1",
		"character": "1",
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if change != nil {
		t.Errorf("expected nil change, got %v", change)
	}
	if output == "" {
		t.Error("expected non-empty output")
	}
	// クライアントなしの場合は設定を促すメッセージ
	if !contains(output, "LSP not available") {
		t.Errorf("output should mention LSP not available, got: %s", output)
	}
}

func TestLSPDefinitionTool_NoClient(t *testing.T) {
	originalClient := LSPClient
	LSPClient = nil
	defer func() { LSPClient = originalClient }()

	tool := &LSPDefinitionTool{}
	output, change, err := tool.Run(map[string]string{
		"path":      "test.go",
		"line":      "1",
		"character": "1",
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if change != nil {
		t.Errorf("expected nil change, got %v", change)
	}
	if !contains(output, "LSP not available") {
		t.Errorf("output should mention LSP not available, got: %s", output)
	}
}

func TestLSPHoverTool_NoClient(t *testing.T) {
	originalClient := LSPClient
	LSPClient = nil
	defer func() { LSPClient = originalClient }()

	tool := &LSPHoverTool{}
	output, change, err := tool.Run(map[string]string{
		"path":      "test.go",
		"line":      "1",
		"character": "1",
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if change != nil {
		t.Errorf("expected nil change, got %v", change)
	}
	if !contains(output, "LSP not available") {
		t.Errorf("output should mention LSP not available, got: %s", output)
	}
}

func TestLSPReferencesTool_MissingPath(t *testing.T) {
	// モッククライアントをセット
	LSPClient = lsp.NewClient("/tmp")
	defer func() { LSPClient = nil }()

	tool := &LSPReferencesTool{}
	output, _, err := tool.Run(map[string]string{
		"line":      "1",
		"character": "1",
	})

	if err == nil {
		t.Error("expected error for missing path")
	}
	if !contains(output, "path is required") {
		t.Errorf("output should mention path is required, got: %s", output)
	}
}

func TestLSPReferencesTool_InvalidLine(t *testing.T) {
	LSPClient = lsp.NewClient("/tmp")
	defer func() { LSPClient = nil }()

	tool := &LSPReferencesTool{}
	output, _, err := tool.Run(map[string]string{
		"path":      "test.go",
		"line":      "abc",
		"character": "1",
	})

	if err == nil {
		t.Error("expected error for invalid line")
	}
	if !contains(output, "line must be a positive number") {
		t.Errorf("output should mention line must be positive, got: %s", output)
	}
}

func TestLSPReferencesTool_InvalidCharacter(t *testing.T) {
	LSPClient = lsp.NewClient("/tmp")
	defer func() { LSPClient = nil }()

	tool := &LSPReferencesTool{}
	output, _, err := tool.Run(map[string]string{
		"path":      "test.go",
		"line":      "1",
		"character": "xyz",
	})

	if err == nil {
		t.Error("expected error for invalid character")
	}
	if !contains(output, "character must be a positive number") {
		t.Errorf("output should mention character must be positive, got: %s", output)
	}
}

func TestLSPReferencesTool_ZeroLine(t *testing.T) {
	LSPClient = lsp.NewClient("/tmp")
	defer func() { LSPClient = nil }()

	tool := &LSPReferencesTool{}
	output, _, err := tool.Run(map[string]string{
		"path":      "test.go",
		"line":      "0",
		"character": "1",
	})

	if err == nil {
		t.Error("expected error for zero line")
	}
	if !contains(output, "line must be a positive number") {
		t.Errorf("output should mention line must be positive, got: %s", output)
	}
}

func TestLSPDefinitionTool_MissingPath(t *testing.T) {
	LSPClient = lsp.NewClient("/tmp")
	defer func() { LSPClient = nil }()

	tool := &LSPDefinitionTool{}
	output, _, err := tool.Run(map[string]string{
		"line":      "1",
		"character": "1",
	})

	if err == nil {
		t.Error("expected error for missing path")
	}
	if !contains(output, "path is required") {
		t.Errorf("output should mention path is required, got: %s", output)
	}
}

func TestLSPHoverTool_MissingPath(t *testing.T) {
	LSPClient = lsp.NewClient("/tmp")
	defer func() { LSPClient = nil }()

	tool := &LSPHoverTool{}
	output, _, err := tool.Run(map[string]string{
		"line":      "1",
		"character": "1",
	})

	if err == nil {
		t.Error("expected error for missing path")
	}
	if !contains(output, "path is required") {
		t.Errorf("output should mention path is required, got: %s", output)
	}
}

func TestRegisterLSPTools(t *testing.T) {
	registry := NewRegistry()
	RegisterLSPTools(registry)

	// 3つのLSPツールが登録されていることを確認
	tools := []string{"lsp_references", "lsp_definition", "lsp_hover"}
	for _, name := range tools {
		if tool := registry.GetTool(name); tool == nil {
			t.Errorf("tool %q not registered", name)
		}
	}
}

func TestLSPToolsSafetyLevel(t *testing.T) {
	// LSPツールはすべてSafetyHigh（読み取り専用）であることを確認
	tools := []string{"lsp_references", "lsp_definition", "lsp_hover"}
	for _, name := range tools {
		safety := GetToolSafety(name)
		if safety != SafetyHigh {
			t.Errorf("tool %q safety = %v, want SafetyHigh", name, safety)
		}
	}
}

// contains はsがsubstrを含むかチェック
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
