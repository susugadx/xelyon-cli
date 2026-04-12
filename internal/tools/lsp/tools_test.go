package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	lsplib "github.com/susugadx/xelyon-cli/internal/lsp"
)

type toolsHelperRequest struct {
	ID     *int            `json:"id,omitempty"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

func TestToolsLSPHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_XELYON_TOOLS_LSP_HELPER") != "1" {
		return
	}
	if err := runToolsLSPHelper(os.Stdin, os.Stdout); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

func toolsLSPHelperCommand(t *testing.T) (string, []string) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}
	return exe, []string{"-test.run=TestToolsLSPHelperProcess", "--"}
}

func runToolsLSPHelper(r io.Reader, w io.Writer) error {
	reader := bufio.NewReader(r)
	for {
		contentLength, err := readToolsLSPContentLength(reader)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		body := make([]byte, contentLength)
		if _, err := io.ReadFull(reader, body); err != nil {
			return err
		}

		var req toolsHelperRequest
		if err := json.Unmarshal(body, &req); err != nil {
			continue
		}

		switch req.Method {
		case "initialize":
			if req.ID != nil {
				if err := writeToolsLSPMessage(w, map[string]any{
					"jsonrpc": "2.0",
					"id":      *req.ID,
					"result": map[string]any{
						"capabilities": map[string]any{
							"textDocumentSync": 1,
						},
					},
				}); err != nil {
					return err
				}
			}
		case "initialized":
		case "textDocument/didOpen":
			var params lsplib.DidOpenTextDocumentParams
			if err := json.Unmarshal(req.Params, &params); err != nil {
				return err
			}
			if err := writeToolsLSPMessage(w, map[string]any{
				"jsonrpc": "2.0",
				"method":  "textDocument/publishDiagnostics",
				"params": lsplib.PublishDiagnosticsParams{
					URI: params.TextDocument.URI,
					Diagnostics: []lsplib.Diagnostic{
						{
							Range: lsplib.Range{
								Start: lsplib.Position{Line: 1, Character: 0},
								End:   lsplib.Position{Line: 1, Character: 4},
							},
							Severity: lsplib.DiagnosticSeverityError,
							Message:  "missing return",
						},
						{
							Range: lsplib.Range{
								Start: lsplib.Position{Line: 3, Character: 1},
								End:   lsplib.Position{Line: 3, Character: 6},
							},
							Severity: lsplib.DiagnosticSeverityWarning,
							Message:  "unused variable",
						},
					},
				},
			}); err != nil {
				return err
			}
		case "shutdown":
			if req.ID != nil {
				if err := writeToolsLSPMessage(w, map[string]any{
					"jsonrpc": "2.0",
					"id":      *req.ID,
					"result":  map[string]any{},
				}); err != nil {
					return err
				}
			}
		case "exit":
			return nil
		}
	}
}

func readToolsLSPContentLength(r *bufio.Reader) (int, error) {
	contentLength := 0
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return 0, err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			return contentLength, nil
		}
		if strings.HasPrefix(line, "Content-Length:") {
			if _, err := fmt.Sscanf(line, "Content-Length: %d", &contentLength); err != nil {
				return 0, err
			}
		}
	}
}

func writeToolsLSPMessage(w io.Writer, msg any) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", len(data)); err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// ===== CheckDiagnosticsForFiles Tests =====

func TestCheckDiagnosticsForFiles_NoLSP(t *testing.T) {
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

func TestCheckDiagnosticsForFilesWithClient_NoLSP(t *testing.T) {
	result := CheckDiagnosticsForFilesWithClient(nil, []string{"test.go"})

	if result.HasErrors {
		t.Error("expected HasErrors=false when explicit client is nil")
	}
	if result.ErrorCount != 0 || result.WarnCount != 0 {
		t.Errorf("expected zero counts, got errors=%d warnings=%d", result.ErrorCount, result.WarnCount)
	}
	if result.Summary != "" {
		t.Errorf("expected empty Summary, got %q", result.Summary)
	}
}

func TestCheckDiagnosticsForFiles_EmptyFiles(t *testing.T) {
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

func TestGetDiagnosticsSummaryWithClient_NoLSP(t *testing.T) {
	if got := GetDiagnosticsSummaryWithClient(nil, "main.go"); got != "" {
		t.Fatalf("expected empty summary with nil client, got %q", got)
	}
}

func TestGetDiagnosticsSummary_NoLSP(t *testing.T) {
	if got := GetDiagnosticsSummary("main.go"); got != "" {
		t.Fatalf("expected empty summary without explicit client, got %q", got)
	}
}

func TestGetDiagnosticsSummaryWithClient_WithDiagnostics(t *testing.T) {
	t.Setenv("GO_WANT_XELYON_TOOLS_LSP_HELPER", "1")

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	tmpDir, err := os.MkdirTemp(cwd, "tools-lsp-summary-*")
	if err != nil {
		t.Fatalf("os.MkdirTemp() error = %v", err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := tmpDir + "/main.go"
	if err := os.WriteFile(filePath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	cmd, args := toolsLSPHelperCommand(t)
	client := lsplib.NewClient(tmpDir)
	client.SetConfigs(map[string]lsplib.ServerConfig{
		"go": {Command: cmd, Args: args},
	})

	got := GetDiagnosticsSummaryWithClient(client, filePath)
	if !strings.Contains(got, "LSP Diagnostics:") {
		t.Fatalf("summary = %q, want diagnostics header", got)
	}
	if !strings.Contains(got, "Errors:") || !strings.Contains(got, "Warnings:") {
		t.Fatalf("summary = %q, want both error and warning sections", got)
	}
	if !strings.Contains(got, "line 2: missing return") {
		t.Fatalf("summary = %q, want formatted error line", got)
	}
	client.Close()
}

func TestCheckDiagnosticsForFilesWithClient_AggregatesDiagnostics(t *testing.T) {
	t.Setenv("GO_WANT_XELYON_TOOLS_LSP_HELPER", "1")

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	tmpDir, err := os.MkdirTemp(cwd, "tools-lsp-check-*")
	if err != nil {
		t.Fatalf("os.MkdirTemp() error = %v", err)
	}
	defer os.RemoveAll(tmpDir)

	fileA := tmpDir + "/a.go"
	fileB := tmpDir + "/b.go"
	for _, path := range []string{fileA, fileB} {
		if err := os.WriteFile(path, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
			t.Fatalf("os.WriteFile(%q) error = %v", path, err)
		}
	}

	cmd, args := toolsLSPHelperCommand(t)
	client := lsplib.NewClient(tmpDir)
	client.SetConfigs(map[string]lsplib.ServerConfig{
		"go": {Command: cmd, Args: args},
	})

	result := CheckDiagnosticsForFilesWithClient(client, []string{fileA, fileB})
	if !result.HasErrors {
		t.Fatal("HasErrors = false, want true")
	}
	if result.ErrorCount != 2 {
		t.Fatalf("ErrorCount = %d, want 2", result.ErrorCount)
	}
	if result.WarnCount != 2 {
		t.Fatalf("WarnCount = %d, want 2", result.WarnCount)
	}
	if !strings.Contains(result.Summary, "LSP Diagnostics: 2 errors, 2 warnings") {
		t.Fatalf("Summary = %q, want aggregate header", result.Summary)
	}
	if !strings.Contains(result.Summary, fileA+":") || !strings.Contains(result.Summary, fileB+":") {
		t.Fatalf("Summary = %q, want both file sections", result.Summary)
	}
	client.Close()
}
