package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	lsplib "github.com/susugadx/xelyon-cli/internal/lsp"
)

type safetyHelperRequest struct {
	ID     *int            `json:"id,omitempty"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

func TestToolsLSPSafetyHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_XELYON_TOOLS_LSP_HELPER") != "1" {
		return
	}
	if err := runToolsLSPSafetyHelper(os.Stdin, os.Stdout); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

func toolsLSPSafetyHelperCommand(t *testing.T) (string, []string) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}
	return exe, []string{"-test.run=TestToolsLSPSafetyHelperProcess", "--"}
}

func runToolsLSPSafetyHelper(r io.Reader, w io.Writer) error {
	reader := bufio.NewReader(r)
	for {
		contentLength, err := readToolsLSPSafetyContentLength(reader)
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

		var req safetyHelperRequest
		if err := json.Unmarshal(body, &req); err != nil {
			continue
		}

		switch req.Method {
		case "initialize":
			if req.ID != nil {
				if err := writeToolsLSPSafetyMessage(w, map[string]any{
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
		case "textDocument/references":
			if req.ID == nil {
				continue
			}
			var params lsplib.ReferenceParams
			if err := json.Unmarshal(req.Params, &params); err != nil {
				return err
			}
			locations := []lsplib.Location{
				{
					URI: params.TextDocument.URI,
					Range: lsplib.Range{
						Start: lsplib.Position{Line: 6, Character: 1},
						End:   lsplib.Position{Line: 6, Character: 7},
					},
				},
				{
					URI: "file:///tmp/external_ref.go",
					Range: lsplib.Range{
						Start: lsplib.Position{Line: 3, Character: 1},
						End:   lsplib.Position{Line: 3, Character: 7},
					},
				},
			}
			if err := writeToolsLSPSafetyMessage(w, map[string]any{
				"jsonrpc": "2.0",
				"id":      *req.ID,
				"result":  locations,
			}); err != nil {
				return err
			}
		case "shutdown":
			if req.ID != nil {
				if err := writeToolsLSPSafetyMessage(w, map[string]any{
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

func readToolsLSPSafetyContentLength(r *bufio.Reader) (int, error) {
	contentLength := 0
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return 0, err
		}
		if line == "\r\n" || line == "\n" {
			return contentLength, nil
		}
		_, _ = fmt.Sscanf(line, "Content-Length: %d", &contentLength)
	}
}

func writeToolsLSPSafetyMessage(w io.Writer, msg any) error {
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

func TestCheckReferencesBeforeDelete_WrapperWithoutClient(t *testing.T) {
	refs, hasExternal, err := CheckReferencesBeforeDelete("main.go", []SymbolInfo{{Name: "main", Line: 1, Column: 1}})
	if err != nil {
		t.Fatalf("CheckReferencesBeforeDelete() error = %v", err)
	}
	if hasExternal {
		t.Fatal("hasExternal should be false without a client")
	}
	if refs != nil {
		t.Fatalf("refs = %v, want nil without client", refs)
	}
}

func TestCheckReferencesBeforeDeleteWithClient_UsesLSPReferences(t *testing.T) {
	t.Setenv("GO_WANT_XELYON_TOOLS_LSP_HELPER", "1")

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(filePath, []byte("package main\n\nfunc helper() {}\n"), 0644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	cmd, args := toolsLSPSafetyHelperCommand(t)
	client := lsplib.NewClient(tmpDir)
	client.SetConfigs(map[string]lsplib.ServerConfig{
		"go": {Command: cmd, Args: args},
	})
	t.Cleanup(client.Close)

	refs, hasExternal, err := CheckReferencesBeforeDeleteWithClient(client, filePath, []SymbolInfo{{Name: "helper", Line: 3, Column: 6}})
	if err != nil {
		t.Fatalf("CheckReferencesBeforeDeleteWithClient() error = %v", err)
	}
	if len(refs) != 2 {
		t.Fatalf("len(refs) = %d, want 2", len(refs))
	}
	if !hasExternal {
		t.Fatal("hasExternal should be true when one reference is outside the file")
	}
	if refs[0].Symbol != "helper" || refs[1].Symbol != "helper" {
		t.Fatalf("refs symbols = %+v, want helper", refs)
	}
	if refs[0].IsLocal == refs[1].IsLocal {
		t.Fatalf("expected one local and one external reference, got %+v", refs)
	}
}
