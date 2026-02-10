package lsp

import (
	"testing"

	lsplib "github.com/susugadx/xelyon-cli/internal/lsp"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func TestRegisterTools(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterTools(registry)

	// lsp_find のみが登録されていることを確認
	if tool := registry.GetTool("lsp_find"); tool == nil {
		t.Error("tool \"lsp_find\" not registered")
	}
}

// ===== CheckDiagnosticsForFiles Tests =====

func TestCheckDiagnosticsForFiles_NoLSP(t *testing.T) {
	// LSPClient が nil の場合、空の結果を返す（ブロックしない）
	originalClient := LSPClient
	LSPClient = nil
	defer func() { LSPClient = originalClient }()

	result := CheckDiagnosticsForFiles([]string{"test.go", "main.go"})

	if result.HasErrors {
		t.Error("expected HasErrors=false when LSP is unavailable")
	}
	if result.ErrorCount != 0 {
		t.Errorf("expected ErrorCount=0, got %d", result.ErrorCount)
	}
	if result.WarnCount != 0 {
		t.Errorf("expected WarnCount=0, got %d", result.WarnCount)
	}
	if result.Summary != "" {
		t.Errorf("expected empty Summary, got %q", result.Summary)
	}
}

func TestCheckDiagnosticsForFiles_EmptyFiles(t *testing.T) {
	// ファイルリストが空の場合
	originalClient := LSPClient
	LSPClient = lsplib.NewClient("/tmp")
	defer func() { LSPClient = originalClient }()

	result := CheckDiagnosticsForFiles([]string{})

	if result.HasErrors {
		t.Error("expected HasErrors=false for empty file list")
	}
	if result.ErrorCount != 0 || result.WarnCount != 0 {
		t.Errorf("expected zero counts, got errors=%d warnings=%d", result.ErrorCount, result.WarnCount)
	}
}

func TestDiagnosticCheckResult_ZeroValue(t *testing.T) {
	// ゼロ値が「問題なし」を表すことを確認
	var result DiagnosticCheckResult

	if result.HasErrors {
		t.Error("zero value HasErrors should be false")
	}
	if result.ErrorCount != 0 {
		t.Error("zero value ErrorCount should be 0")
	}
	if result.WarnCount != 0 {
		t.Error("zero value WarnCount should be 0")
	}
	if result.Summary != "" {
		t.Error("zero value Summary should be empty")
	}
}

func TestLSPToolsSafetyLevel(t *testing.T) {
	// lsp_find は SafetyHigh（読み取り専用）であることを確認
	safety := common.GetToolSafety("lsp_find")
	if safety != common.SafetyHigh {
		t.Errorf("tool \"lsp_find\" safety = %v, want SafetyHigh", safety)
	}
}
