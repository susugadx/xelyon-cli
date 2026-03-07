package lsp

import (
	"context"
	"fmt"
	"strings"
	"time"

	lsplib "github.com/susugadx/xelyon-cli/internal/lsp"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// LSPClient is set by agent during initialization
var LSPClient *lsplib.Client

// LSPToolTimeout is the timeout for LSP operations
const LSPToolTimeout = 30 * time.Second

// GetDiagnosticsSummary returns a summarized string of diagnostics for a file.
// This is used by file editing tools to provide immediate feedback.
func GetDiagnosticsSummary(path string) string {
	if LSPClient == nil {
		return ""
	}

	absPath, err := common.ValidatePath(path)
	if err != nil {
		return ""
	}

	// Use a short timeout for background checks
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	diagnostics, err := LSPClient.GetDiagnostics(ctx, absPath)
	if err != nil || len(diagnostics) == 0 {
		return ""
	}

	var errors, warnings []string
	for _, d := range diagnostics {
		msg := fmt.Sprintf("  line %d: %s", d.Range.Start.Line+1, d.Message)
		switch d.Severity {
		case lsplib.DiagnosticSeverityError:
			errors = append(errors, msg)
		case lsplib.DiagnosticSeverityWarning:
			warnings = append(warnings, msg)
		}
	}

	if len(errors) == 0 && len(warnings) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n⚠️ LSP Diagnostics:")
	if len(errors) > 0 {
		sb.WriteString("\n  Errors:\n")
		sb.WriteString(strings.Join(errors, "\n"))
	}
	if len(warnings) > 0 {
		sb.WriteString("\n  Warnings:\n")
		sb.WriteString(strings.Join(warnings, "\n"))
	}

	return sb.String()
}

// DiagnosticCheckResult はコミット前診断チェックの結果
type DiagnosticCheckResult struct {
	HasErrors  bool
	ErrorCount int
	WarnCount  int
	Summary    string // AI向けの詳細テキスト
}

// CheckDiagnosticsForFiles は複数ファイルの LSP 診断を一括チェックする。
// git_commit 前の自動チェック用。LSP 未起動時は HasErrors=false を返す（ブロックしない）。
func CheckDiagnosticsForFiles(files []string) DiagnosticCheckResult {
	if LSPClient == nil {
		return DiagnosticCheckResult{}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var totalErrors, totalWarnings int
	var sb strings.Builder

	for _, file := range files {
		absPath, err := common.ValidatePath(file)
		if err != nil {
			continue
		}

		diagnostics, err := LSPClient.GetDiagnostics(ctx, absPath)
		if err != nil || len(diagnostics) == 0 {
			continue
		}

		fileHasIssue := false
		for _, d := range diagnostics {
			switch d.Severity {
			case lsplib.DiagnosticSeverityError:
				if !fileHasIssue {
					fmt.Fprintf(&sb, "\n%s:\n", file)
					fileHasIssue = true
				}
				fmt.Fprintf(&sb, "  ❌ Error [%d:%d]: %s\n",
					d.Range.Start.Line+1, d.Range.Start.Character+1, d.Message)
				totalErrors++
			case lsplib.DiagnosticSeverityWarning:
				if !fileHasIssue {
					fmt.Fprintf(&sb, "\n%s:\n", file)
					fileHasIssue = true
				}
				fmt.Fprintf(&sb, "  ⚠️ Warning [%d:%d]: %s\n",
					d.Range.Start.Line+1, d.Range.Start.Character+1, d.Message)
				totalWarnings++
			}
		}
	}

	if totalErrors == 0 && totalWarnings == 0 {
		return DiagnosticCheckResult{}
	}

	header := fmt.Sprintf("LSP Diagnostics: %d errors, %d warnings\n", totalErrors, totalWarnings)
	return DiagnosticCheckResult{
		HasErrors:  totalErrors > 0,
		ErrorCount: totalErrors,
		WarnCount:  totalWarnings,
		Summary:    header + sb.String(),
	}
}
