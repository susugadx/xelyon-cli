package lsp

import (
	"strings"
	"testing"

	lsplib "github.com/susugadx/xelyon-cli/internal/lsp"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
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
	if !strings.Contains(output, "LSP not available") {
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
	if !strings.Contains(output, "LSP not available") {
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
	if !strings.Contains(output, "LSP not available") {
		t.Errorf("output should mention LSP not available, got: %s", output)
	}
}

func TestLSPReferencesTool_MissingPath(t *testing.T) {
	// モッククライアントをセット
	LSPClient = lsplib.NewClient("/tmp")
	defer func() { LSPClient = nil }()

	tool := &LSPReferencesTool{}
	output, _, err := tool.Run(map[string]string{
		"line":      "1",
		"character": "1",
	})

	if err == nil {
		t.Error("expected error for missing path")
	}
	if !strings.Contains(output, "path is required") {
		t.Errorf("output should mention path is required, got: %s", output)
	}
}

func TestLSPReferencesTool_InvalidLine(t *testing.T) {
	LSPClient = lsplib.NewClient("/tmp")
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
	if !strings.Contains(output, "line must be a positive number") {
		t.Errorf("output should mention line must be positive, got: %s", output)
	}
}

func TestLSPReferencesTool_InvalidCharacter(t *testing.T) {
	LSPClient = lsplib.NewClient("/tmp")
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
	if !strings.Contains(output, "character must be a positive number") {
		t.Errorf("output should mention character must be positive, got: %s", output)
	}
}

func TestLSPReferencesTool_ZeroLine(t *testing.T) {
	LSPClient = lsplib.NewClient("/tmp")
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
	if !strings.Contains(output, "line must be a positive number") {
		t.Errorf("output should mention line must be positive, got: %s", output)
	}
}

func TestLSPDefinitionTool_MissingPath(t *testing.T) {
	LSPClient = lsplib.NewClient("/tmp")
	defer func() { LSPClient = nil }()

	tool := &LSPDefinitionTool{}
	output, _, err := tool.Run(map[string]string{
		"line":      "1",
		"character": "1",
	})

	if err == nil {
		t.Error("expected error for missing path")
	}
	if !strings.Contains(output, "path is required") {
		t.Errorf("output should mention path is required, got: %s", output)
	}
}

func TestLSPHoverTool_MissingPath(t *testing.T) {
	LSPClient = lsplib.NewClient("/tmp")
	defer func() { LSPClient = nil }()

	tool := &LSPHoverTool{}
	output, _, err := tool.Run(map[string]string{
		"line":      "1",
		"character": "1",
	})

	if err == nil {
		t.Error("expected error for missing path")
	}
	if !strings.Contains(output, "path is required") {
		t.Errorf("output should mention path is required, got: %s", output)
	}
}

func TestLSPDiagnosticsTool_Name(t *testing.T) {
	tool := &LSPDiagnosticsTool{}
	if got := tool.Name(); got != "lsp_diagnostics" {
		t.Errorf("Name() = %q, want %q", got, "lsp_diagnostics")
	}
}

func TestLSPDiagnosticsTool_NoClient(t *testing.T) {
	originalClient := LSPClient
	LSPClient = nil
	defer func() { LSPClient = originalClient }()

	tool := &LSPDiagnosticsTool{}
	output, change, err := tool.Run(map[string]string{
		"path": "test.go",
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if change != nil {
		t.Errorf("expected nil change, got %v", change)
	}
	if !strings.Contains(output, "LSP not available") {
		t.Errorf("output should mention LSP not available, got: %s", output)
	}
}

func TestLSPDiagnosticsTool_MissingPath(t *testing.T) {
	LSPClient = lsplib.NewClient("/tmp")
	defer func() { LSPClient = nil }()

	tool := &LSPDiagnosticsTool{}
	output, _, err := tool.Run(map[string]string{})

	if err == nil {
		t.Error("expected error for missing path")
	}
	if !strings.Contains(output, "path is required") {
		t.Errorf("output should mention path is required, got: %s", output)
	}
}

func TestLSPRenameTool_Name(t *testing.T) {
	tool := &LSPRenameTool{}
	if got := tool.Name(); got != "lsp_rename" {
		t.Errorf("Name() = %q, want %q", got, "lsp_rename")
	}
}

func TestLSPRenameTool_NoClient(t *testing.T) {
	originalClient := LSPClient
	LSPClient = nil
	defer func() { LSPClient = originalClient }()

	tool := &LSPRenameTool{}
	output, change, err := tool.Run(map[string]string{
		"path":      "test.go",
		"line":      "1",
		"character": "1",
		"new_name":  "newName",
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if change != nil {
		t.Errorf("expected nil change, got %v", change)
	}
	if !strings.Contains(output, "LSP not available") {
		t.Errorf("output should mention LSP not available, got: %s", output)
	}
}

func TestLSPRenameTool_MissingPath(t *testing.T) {
	LSPClient = lsplib.NewClient("/tmp")
	defer func() { LSPClient = nil }()

	tool := &LSPRenameTool{}
	output, _, err := tool.Run(map[string]string{
		"line":      "1",
		"character": "1",
		"new_name":  "newName",
	})

	if err == nil {
		t.Error("expected error for missing path")
	}
	if !strings.Contains(output, "path is required") {
		t.Errorf("output should mention path is required, got: %s", output)
	}
}

func TestLSPRenameTool_MissingNewName(t *testing.T) {
	LSPClient = lsplib.NewClient("/tmp")
	defer func() { LSPClient = nil }()

	tool := &LSPRenameTool{}
	output, _, err := tool.Run(map[string]string{
		"path":      "test.go",
		"line":      "1",
		"character": "1",
	})

	if err == nil {
		t.Error("expected error for missing new_name")
	}
	if !strings.Contains(output, "new_name is required") {
		t.Errorf("output should mention new_name is required, got: %s", output)
	}
}

func TestLSPRenameTool_InvalidLine(t *testing.T) {
	LSPClient = lsplib.NewClient("/tmp")
	defer func() { LSPClient = nil }()

	tool := &LSPRenameTool{}
	output, _, err := tool.Run(map[string]string{
		"path":      "test.go",
		"line":      "abc",
		"character": "1",
		"new_name":  "newName",
	})

	if err == nil {
		t.Error("expected error for invalid line")
	}
	if !strings.Contains(output, "line must be a positive number") {
		t.Errorf("output should mention line must be positive, got: %s", output)
	}
}

func TestRegisterTools(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterTools(registry)

	// 5つのLSPツールが登録されていることを確認
	lspTools := []string{"lsp_references", "lsp_definition", "lsp_hover", "lsp_diagnostics", "lsp_rename"}
	for _, name := range lspTools {
		if tool := registry.GetTool(name); tool == nil {
			t.Errorf("tool %q not registered", name)
		}
	}
}

func TestLSPToolsSafetyLevel(t *testing.T) {
	// LSPツールはすべてSafetyHigh（読み取り専用）であることを確認
	lspTools := []string{"lsp_references", "lsp_definition", "lsp_hover", "lsp_diagnostics", "lsp_rename"}
	for _, name := range lspTools {
		safety := common.GetToolSafety(name)
		if safety != common.SafetyHigh {
			t.Errorf("tool %q safety = %v, want SafetyHigh", name, safety)
		}
	}
}
