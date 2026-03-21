package lsp

import (
	"encoding/json"
	"testing"
)

func TestRequest_JSON(t *testing.T) {
	req := Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  map[string]string{"key": "value"},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var parsed Request
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if parsed.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want 2.0", parsed.JSONRPC)
	}
	if parsed.ID != 1 {
		t.Errorf("ID = %d, want 1", parsed.ID)
	}
	if parsed.Method != "initialize" {
		t.Errorf("Method = %q, want initialize", parsed.Method)
	}
}

func TestNotification_JSON_NoID(t *testing.T) {
	notif := Notification{
		JSONRPC: "2.0",
		Method:  "initialized",
	}

	data, err := json.Marshal(notif)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if _, ok := raw["id"]; ok {
		t.Error("notification should not have an 'id' field")
	}
}

func TestResponse_JSON_WithResult(t *testing.T) {
	result := json.RawMessage(`{"capabilities": {}}`)
	resp := Response{
		JSONRPC: "2.0",
		ID:      1,
		Result:  result,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var parsed Response
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if parsed.Error != nil {
		t.Error("expected no error in response")
	}
	if parsed.Result == nil {
		t.Error("expected non-nil result")
	}
}

func TestResponse_JSON_WithError(t *testing.T) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      2,
		Error: &ResponseError{
			Code:    -32601,
			Message: "Method not found",
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var parsed Response
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if parsed.Error == nil {
		t.Fatal("expected error in response")
	}
	if parsed.Error.Code != -32601 {
		t.Errorf("error code = %d, want -32601", parsed.Error.Code)
	}
}

func TestDiagnostic_JSON(t *testing.T) {
	diag := Diagnostic{
		Range: Range{
			Start: Position{Line: 10, Character: 5},
			End:   Position{Line: 10, Character: 15},
		},
		Severity: DiagnosticSeverityError,
		Source:   "gopls",
		Message:  "undefined: foo",
	}

	data, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var parsed Diagnostic
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if parsed.Severity != DiagnosticSeverityError {
		t.Errorf("Severity = %d, want %d", parsed.Severity, DiagnosticSeverityError)
	}
	if parsed.Message != "undefined: foo" {
		t.Errorf("Message = %q, want %q", parsed.Message, "undefined: foo")
	}
	if parsed.Range.Start.Line != 10 {
		t.Errorf("Range.Start.Line = %d, want 10", parsed.Range.Start.Line)
	}
}

func TestPublishDiagnosticsParams_JSON(t *testing.T) {
	params := PublishDiagnosticsParams{
		URI: "file:///test.go",
		Diagnostics: []Diagnostic{
			{
				Range: Range{
					Start: Position{Line: 1, Character: 0},
					End:   Position{Line: 1, Character: 10},
				},
				Severity: DiagnosticSeverityWarning,
				Message:  "unused variable",
			},
		},
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var parsed PublishDiagnosticsParams
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if parsed.URI != "file:///test.go" {
		t.Errorf("URI = %q, want file:///test.go", parsed.URI)
	}
	if len(parsed.Diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(parsed.Diagnostics))
	}
	if parsed.Diagnostics[0].Severity != DiagnosticSeverityWarning {
		t.Errorf("diagnostic severity = %d, want %d", parsed.Diagnostics[0].Severity, DiagnosticSeverityWarning)
	}
}

func TestDiagnosticSeverity_Values(t *testing.T) {
	if DiagnosticSeverityError != 1 {
		t.Errorf("Error = %d, want 1", DiagnosticSeverityError)
	}
	if DiagnosticSeverityWarning != 2 {
		t.Errorf("Warning = %d, want 2", DiagnosticSeverityWarning)
	}
	if DiagnosticSeverityInformation != 3 {
		t.Errorf("Information = %d, want 3", DiagnosticSeverityInformation)
	}
	if DiagnosticSeverityHint != 4 {
		t.Errorf("Hint = %d, want 4", DiagnosticSeverityHint)
	}
}

func TestLocation_JSON(t *testing.T) {
	loc := Location{
		URI: "file:///src/main.go",
		Range: Range{
			Start: Position{Line: 5, Character: 10},
			End:   Position{Line: 5, Character: 20},
		},
	}

	data, err := json.Marshal(loc)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var parsed Location
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if parsed.URI != "file:///src/main.go" {
		t.Errorf("URI = %q, want file:///src/main.go", parsed.URI)
	}
	if parsed.Range.Start.Character != 10 {
		t.Errorf("Start.Character = %d, want 10", parsed.Range.Start.Character)
	}
}

func TestInitializeParams_JSON(t *testing.T) {
	params := InitializeParams{
		ProcessID: 12345,
		RootURI:   "file:///workspace",
		Capabilities: ClientCapabilities{
			TextDocument: TextDocumentClientCapabilities{
				References: &ReferencesCapability{
					DynamicRegistration: true,
				},
			},
		},
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var parsed InitializeParams
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if parsed.ProcessID != 12345 {
		t.Errorf("ProcessID = %d, want 12345", parsed.ProcessID)
	}
	if parsed.RootURI != "file:///workspace" {
		t.Errorf("RootURI = %q, want file:///workspace", parsed.RootURI)
	}
	if parsed.Capabilities.TextDocument.References == nil {
		t.Fatal("expected references capability")
	}
	if !parsed.Capabilities.TextDocument.References.DynamicRegistration {
		t.Error("expected DynamicRegistration to be true")
	}
}
