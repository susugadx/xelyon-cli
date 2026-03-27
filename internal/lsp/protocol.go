package lsp

import "encoding/json"

// ===== JSON-RPC 2.0 Types =====

// Request represents a JSON-RPC 2.0 request
type Request struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// Notification represents a JSON-RPC 2.0 notification (no ID, no response expected)
type Notification struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// Response represents a JSON-RPC 2.0 response
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *ResponseError  `json:"error,omitempty"`
}

// ResponseError represents a JSON-RPC 2.0 error
type ResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// ===== LSP Basic Types =====

// Position represents a position in a text document (0-indexed)
type Position struct {
	Line      int `json:"line"`      // 0-indexed line number
	Character int `json:"character"` // 0-indexed character offset
}

// Range represents a text range in a document
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Location represents a location in a text document
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// TextDocumentIdentifier identifies a text document
type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

// TextDocumentItem represents a text document (for didOpen)
type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

// TextDocumentPositionParams represents a position in a text document
type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// ===== Initialize Request/Response =====

// InitializeParams are the parameters for the initialize request
type InitializeParams struct {
	ProcessID    int                `json:"processId"`
	RootURI      string             `json:"rootUri"`
	Capabilities ClientCapabilities `json:"capabilities"`
}

// ClientCapabilities represents the client's capabilities
type ClientCapabilities struct {
	TextDocument TextDocumentClientCapabilities `json:"textDocument,omitempty"`
}

// TextDocumentClientCapabilities represents text document capabilities
type TextDocumentClientCapabilities struct {
	References     *ReferencesCapability     `json:"references,omitempty"`
	Definition     *DefinitionCapability     `json:"definition,omitempty"`
	Implementation *ImplementationCapability `json:"implementation,omitempty"`
}

// ReferencesCapability represents references capability
type ReferencesCapability struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// DefinitionCapability represents definition capability.
type DefinitionCapability struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// ImplementationCapability represents implementation capability.
type ImplementationCapability struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// InitializeResult is the result of the initialize request
type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
}

// ServerCapabilities represents the server's capabilities
type ServerCapabilities struct {
	ReferencesProvider bool `json:"referencesProvider,omitempty"`
	TextDocumentSync   int  `json:"textDocumentSync,omitempty"` // 0=None, 1=Full, 2=Incremental
}

// ===== textDocument/references =====

// ReferenceParams are the parameters for the references request
type ReferenceParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
	Context      ReferenceContext       `json:"context"`
}

// ReferenceContext represents the context for a reference request
type ReferenceContext struct {
	IncludeDeclaration bool `json:"includeDeclaration"`
}

// ===== textDocument/didOpen =====

// DidOpenTextDocumentParams are the parameters for the didOpen notification
type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

// ===== textDocument/didClose =====

// DidCloseTextDocumentParams are the parameters for the didClose notification
type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// ===== textDocument/publishDiagnostics =====

// DiagnosticSeverity represents the severity of a diagnostic
type DiagnosticSeverity int

const (
	DiagnosticSeverityError       DiagnosticSeverity = 1
	DiagnosticSeverityWarning     DiagnosticSeverity = 2
	DiagnosticSeverityInformation DiagnosticSeverity = 3
	DiagnosticSeverityHint        DiagnosticSeverity = 4
)

// Diagnostic represents a diagnostic (error, warning, info, hint)
type Diagnostic struct {
	Range    Range              `json:"range"`
	Severity DiagnosticSeverity `json:"severity,omitempty"`
	Code     interface{}        `json:"code,omitempty"` // string or number
	Source   string             `json:"source,omitempty"`
	Message  string             `json:"message"`
}

// PublishDiagnosticsParams are the parameters for the publishDiagnostics notification
type PublishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}
