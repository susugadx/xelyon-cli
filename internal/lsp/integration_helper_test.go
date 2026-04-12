package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

type helperRequest struct {
	ID     *int            `json:"id,omitempty"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

func TestLSPHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_XELYON_LSP_HELPER") != "1" {
		return
	}
	if err := runLSPHelper(os.Stdin, os.Stdout); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

func lspHelperCommand(t *testing.T) (string, []string) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}
	return exe, []string{"-test.run=TestLSPHelperProcess", "--"}
}

func runLSPHelper(r io.Reader, w io.Writer) error {
	reader := bufio.NewReader(r)
	for {
		contentLength, err := readHelperContentLength(reader)
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

		var req helperRequest
		if err := json.Unmarshal(body, &req); err != nil {
			continue
		}

		switch req.Method {
		case "initialize":
			if req.ID != nil {
				if err := writeHelperMessage(w, map[string]any{
					"jsonrpc": "2.0",
					"id":      *req.ID,
					"result": map[string]any{
						"capabilities": map[string]any{
							"referencesProvider": true,
							"textDocumentSync":   1,
						},
					},
				}); err != nil {
					return err
				}
			}
		case "initialized":
		case "textDocument/didOpen":
			var params DidOpenTextDocumentParams
			if err := json.Unmarshal(req.Params, &params); err != nil {
				return err
			}
			if err := writeHelperMessage(w, map[string]any{
				"jsonrpc": "2.0",
				"method":  "textDocument/publishDiagnostics",
				"params": PublishDiagnosticsParams{
					URI: params.TextDocument.URI,
					Diagnostics: []Diagnostic{
						{
							Range: Range{
								Start: Position{Line: 2, Character: 4},
								End:   Position{Line: 2, Character: 10},
							},
							Severity: DiagnosticSeverityError,
							Message:  "undefined: helper",
						},
						{
							Range: Range{
								Start: Position{Line: 5, Character: 1},
								End:   Position{Line: 5, Character: 7},
							},
							Severity: DiagnosticSeverityWarning,
							Message:  "unused variable",
						},
					},
				},
			}); err != nil {
				return err
			}
		case "textDocument/references":
			if req.ID != nil {
				var params ReferenceParams
				if err := json.Unmarshal(req.Params, &params); err != nil {
					return err
				}
				locations := []Location{
					{
						URI: params.TextDocument.URI,
						Range: Range{
							Start: Position{Line: 9, Character: 1},
							End:   Position{Line: 9, Character: 8},
						},
					},
					{
						URI: params.TextDocument.URI,
						Range: Range{
							Start: Position{Line: 15, Character: 2},
							End:   Position{Line: 15, Character: 9},
						},
					},
				}
				if err := writeHelperMessage(w, map[string]any{
					"jsonrpc": "2.0",
					"id":      *req.ID,
					"result":  locations,
				}); err != nil {
					return err
				}
			}
		case "textDocument/definition":
			if req.ID != nil {
				var params TextDocumentPositionParams
				if err := json.Unmarshal(req.Params, &params); err != nil {
					return err
				}
				if err := writeHelperMessage(w, map[string]any{
					"jsonrpc": "2.0",
					"id":      *req.ID,
					"result": []Location{
						{
							URI: params.TextDocument.URI,
							Range: Range{
								Start: Position{Line: 0, Character: 0},
								End:   Position{Line: 0, Character: 6},
							},
						},
					},
				}); err != nil {
					return err
				}
			}
		case "textDocument/implementation":
			if req.ID != nil {
				var params TextDocumentPositionParams
				if err := json.Unmarshal(req.Params, &params); err != nil {
					return err
				}
				if err := writeHelperMessage(w, map[string]any{
					"jsonrpc": "2.0",
					"id":      *req.ID,
					"result": []Location{
						{
							URI: params.TextDocument.URI,
							Range: Range{
								Start: Position{Line: 20, Character: 0},
								End:   Position{Line: 20, Character: 12},
							},
						},
					},
				}); err != nil {
					return err
				}
			}
		case "shutdown":
			if req.ID != nil {
				if err := writeHelperMessage(w, map[string]any{
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

func readHelperContentLength(r *bufio.Reader) (int, error) {
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

func writeHelperMessage(w io.Writer, msg any) error {
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
