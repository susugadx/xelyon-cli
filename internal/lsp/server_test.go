package lsp

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

type nopWriteCloser struct {
	*bytes.Buffer
}

func (n nopWriteCloser) Close() error { return nil }

func TestNewServer(t *testing.T) {
	s := NewServer("gopls")
	if s == nil {
		t.Fatal("NewServer returned nil")
	}
	if s.name != "gopls" {
		t.Errorf("name = %q, want %q", s.name, "gopls")
	}
	if s.IsRunning() {
		t.Error("new server should not be running")
	}
}

func TestServer_Name(t *testing.T) {
	s := NewServer("rust-analyzer")
	if got := s.Name(); got != "rust-analyzer" {
		t.Errorf("Name() = %q, want %q", got, "rust-analyzer")
	}
}

func TestServer_IsRunning_Initial(t *testing.T) {
	s := NewServer("test-server")
	if s.IsRunning() {
		t.Error("IsRunning() should return false for a newly created server")
	}
}

func TestServer_SetDebugOutput(t *testing.T) {
	s := NewServer("test")

	var buf bytes.Buffer
	s.SetDebugOutput(&buf)
	if s.debugOut != &buf {
		t.Error("SetDebugOutput did not set the writer")
	}

	s.SetDebugOutput(nil)
	if s.debugOut != io.Discard {
		t.Error("SetDebugOutput(nil) should normalize to io.Discard")
	}
}

func TestServer_OpenDocument_SendsDidOpenOncePerURI(t *testing.T) {
	s := NewServer("test")
	var buf bytes.Buffer
	s.stdin = nopWriteCloser{Buffer: &buf}

	if err := s.OpenDocument("/tmp/example.go", "go", "package main\n"); err != nil {
		t.Fatalf("OpenDocument() error = %v", err)
	}
	if err := s.OpenDocument("/tmp/example.go", "go", "package main\n"); err != nil {
		t.Fatalf("second OpenDocument() error = %v", err)
	}
	if got := strings.Count(buf.String(), "textDocument/didOpen"); got != 1 {
		t.Fatalf("didOpen count = %d, want 1", got)
	}

	if err := s.CloseDocument("/tmp/example.go"); err != nil {
		t.Fatalf("CloseDocument() error = %v", err)
	}
	if err := s.OpenDocument("/tmp/example.go", "go", "package main\n"); err != nil {
		t.Fatalf("reopen OpenDocument() error = %v", err)
	}
	if got := strings.Count(buf.String(), "textDocument/didOpen"); got != 2 {
		t.Fatalf("didOpen count after reopen = %d, want 2", got)
	}
}

func TestServer_GetLastDiagnostics_Empty(t *testing.T) {
	s := NewServer("test")
	diags := s.GetLastDiagnostics("/nonexistent/file.go")
	if diags != nil {
		t.Errorf("expected nil diagnostics for unknown file, got %v", diags)
	}
}

func TestServer_GetLastDiagnostics_WithData(t *testing.T) {
	s := NewServer("test")

	uri := FileToURI("/tmp/test.go")
	expected := []Diagnostic{
		{
			Message:  "undefined: foo",
			Severity: DiagnosticSeverityError,
			Range:    Range{Start: Position{Line: 10, Character: 5}},
		},
		{
			Message:  "unused variable",
			Severity: DiagnosticSeverityWarning,
			Range:    Range{Start: Position{Line: 20, Character: 0}},
		},
	}

	s.diagMu.Lock()
	s.diagnostics[uri] = expected
	s.diagMu.Unlock()

	got := s.GetLastDiagnostics("/tmp/test.go")
	if len(got) != 2 {
		t.Fatalf("expected 2 diagnostics, got %d", len(got))
	}
	if got[0].Message != "undefined: foo" {
		t.Errorf("diagnostics[0].Message = %q, want %q", got[0].Message, "undefined: foo")
	}
	if got[1].Message != "unused variable" {
		t.Errorf("diagnostics[1].Message = %q, want %q", got[1].Message, "unused variable")
	}
}

func TestServer_ClearDiagnostics(t *testing.T) {
	s := NewServer("test")

	uri := FileToURI("/tmp/test.go")
	s.diagMu.Lock()
	s.diagnostics[uri] = []Diagnostic{
		{Message: "some error", Severity: DiagnosticSeverityError},
	}
	s.diagMu.Unlock()

	// Verify data is present before clearing
	if len(s.GetLastDiagnostics("/tmp/test.go")) != 1 {
		t.Fatal("expected 1 diagnostic before clear")
	}

	s.ClearDiagnostics("/tmp/test.go")

	got := s.GetLastDiagnostics("/tmp/test.go")
	if got != nil {
		t.Errorf("expected nil after clear, got %v", got)
	}
}

func TestServer_HandleNotification_PublishDiagnostics(t *testing.T) {
	s := NewServer("test")

	diagParams := PublishDiagnosticsParams{
		URI: "file:///test.go",
		Diagnostics: []Diagnostic{
			{
				Message:  "undefined: foo",
				Severity: DiagnosticSeverityError,
				Range:    Range{Start: Position{Line: 10}},
			},
			{
				Message:  "unused import",
				Severity: DiagnosticSeverityWarning,
				Range:    Range{Start: Position{Line: 3, Character: 2}},
			},
		},
	}

	params, err := json.Marshal(diagParams)
	if err != nil {
		t.Fatalf("failed to marshal params: %v", err)
	}

	s.handleNotification("textDocument/publishDiagnostics", params)

	s.diagMu.RLock()
	stored := s.diagnostics["file:///test.go"]
	s.diagMu.RUnlock()

	if len(stored) != 2 {
		t.Fatalf("expected 2 diagnostics, got %d", len(stored))
	}
	if stored[0].Message != "undefined: foo" {
		t.Errorf("diagnostics[0].Message = %q, want %q", stored[0].Message, "undefined: foo")
	}
	if stored[0].Severity != DiagnosticSeverityError {
		t.Errorf("diagnostics[0].Severity = %d, want %d", stored[0].Severity, DiagnosticSeverityError)
	}
	if stored[1].Message != "unused import" {
		t.Errorf("diagnostics[1].Message = %q, want %q", stored[1].Message, "unused import")
	}
}

func TestServer_HandleNotification_UnknownMethod(t *testing.T) {
	s := NewServer("test")

	// Should not panic
	s.handleNotification("window/logMessage", json.RawMessage(`{"type":3,"message":"hello"}`))

	// Diagnostics should remain empty
	s.diagMu.RLock()
	defer s.diagMu.RUnlock()
	if len(s.diagnostics) != 0 {
		t.Errorf("expected no diagnostics, got %d entries", len(s.diagnostics))
	}
}

func TestServer_HandleNotification_InvalidJSON(t *testing.T) {
	s := NewServer("test")

	// Should not panic even with invalid JSON for publishDiagnostics
	s.handleNotification("textDocument/publishDiagnostics", json.RawMessage(`{invalid json`))

	// Diagnostics should remain empty
	s.diagMu.RLock()
	defer s.diagMu.RUnlock()
	if len(s.diagnostics) != 0 {
		t.Errorf("expected no diagnostics after invalid JSON, got %d entries", len(s.diagnostics))
	}
}

func TestServer_Close_NotStarted(t *testing.T) {
	s := NewServer("test")

	// Close on an unstarted server should return nil and not panic
	err := s.Close()
	if err != nil {
		t.Errorf("Close() on unstarted server returned error: %v", err)
	}

	// Calling Close again should also be safe (idempotent via sync.Once)
	err = s.Close()
	if err != nil {
		t.Errorf("second Close() returned error: %v", err)
	}
}
