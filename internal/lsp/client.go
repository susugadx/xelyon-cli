package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

// ServerConfig holds configuration for an LSP server
type ServerConfig struct {
	Command  string
	Args     []string
	Disabled bool
}

// Client manages multiple LSP servers (one per language)
type Client struct {
	mu      sync.RWMutex
	servers map[string]*Server      // language key -> server
	configs map[string]ServerConfig // language key -> config
	rootURI string
	output  io.Writer
	errOut  io.Writer
}

// NewClient creates a new LSP client
func NewClient(rootPath string) *Client {
	return &Client{
		servers: make(map[string]*Server),
		configs: make(map[string]ServerConfig),
		rootURI: FileToURI(rootPath),
		output:  io.Discard,
		errOut:  io.Discard,
	}
}

// SetOutput は Client の出力先を設定する。
func (c *Client) SetOutput(w io.Writer) {
	c.output = w
}

// SetErrorOutput は Client のエラー/デバッグ出力先を設定する。
func (c *Client) SetErrorOutput(w io.Writer) {
	c.errOut = w
}

// out は Client の出力先を返す。未設定時は出力を抑制する。
func (c *Client) out() io.Writer {
	if c.output != nil {
		return c.output
	}
	return io.Discard
}

func (c *Client) errorOutput() io.Writer {
	if c.errOut != nil {
		return c.errOut
	}
	return io.Discard
}

// SetConfigs sets the server configurations
func (c *Client) SetConfigs(configs map[string]ServerConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.configs = configs
}

// GetServer returns the server for a language, starting it lazily if needed
func (c *Client) GetServer(ctx context.Context, language string) (*Server, error) {
	// Map language to server key
	serverKey := LanguageServerKey(language)

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if server already exists and is running
	if server, ok := c.servers[serverKey]; ok {
		if server.IsRunning() {
			return server, nil
		}
		// Server died, clean up
		server.Close()
		delete(c.servers, serverKey)
	}

	// Get config for language
	config, ok := c.configs[serverKey]
	if !ok {
		return nil, fmt.Errorf("no LSP server configured for language: %s", language)
	}

	if config.Disabled {
		return nil, fmt.Errorf("LSP server for %s is disabled", language)
	}

	// Check if command exists
	if _, err := exec.LookPath(config.Command); err != nil {
		return nil, fmt.Errorf("LSP server command not found: %s (install it to use LSP features)", config.Command)
	}

	// Create and start server
	server := NewServer(serverKey)
	server.SetDebugOutput(c.errorOutput())

	// Use timeout context for startup
	startCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := server.Start(startCtx, config.Command, config.Args, c.rootURI); err != nil {
		return nil, fmt.Errorf("failed to start LSP server for %s: %w", language, err)
	}

	c.servers[serverKey] = server
	fmt.Fprintf(c.out(), "🔌 LSP server '%s' started for %s\n", config.Command, language)
	return server, nil
}

// GetServerForFile returns the appropriate server for a file path
func (c *Client) GetServerForFile(ctx context.Context, filePath string) (*Server, error) {
	language := DetectLanguage(filePath)
	if language == "" {
		return nil, fmt.Errorf("could not detect language for file: %s", filePath)
	}
	return c.GetServer(ctx, language)
}

// Close shuts down all LSP servers
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for lang, server := range c.servers {
		if err := server.Close(); err != nil {
			fmt.Fprintf(c.errorOutput(), "Warning: failed to close LSP server for %s: %v\n", lang, err)
		}
	}
	c.servers = make(map[string]*Server)
}

// Status returns the status of all configured servers
func (c *Client) Status() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	status := make(map[string]string)
	for lang, config := range c.configs {
		if config.Disabled {
			status[lang] = "disabled"
			continue
		}

		if server, ok := c.servers[lang]; ok && server.IsRunning() {
			status[lang] = "running"
		} else {
			// Check if command exists
			if _, err := exec.LookPath(config.Command); err != nil {
				status[lang] = fmt.Sprintf("not installed (%s)", config.Command)
			} else {
				status[lang] = "not started (lazy)"
			}
		}
	}
	return status
}

// ===== LSP Operations =====

// FindReferences finds all references to a symbol
func (c *Client) FindReferences(ctx context.Context, filePath string, line, character int, includeDeclaration bool) ([]Location, error) {
	server, err := c.GetServerForFile(ctx, filePath)
	if err != nil {
		return nil, err
	}

	// Read file content and open document
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	language := DetectLanguage(filePath)
	if err := server.OpenDocument(filePath, language, string(content)); err != nil {
		return nil, fmt.Errorf("failed to open document: %w", err)
	}

	// Wait for server to process the document
	server.WaitForDocumentReady(ctx, FileToURI(filePath), 100*time.Millisecond)

	params := ReferenceParams{
		TextDocument: TextDocumentIdentifier{URI: FileToURI(filePath)},
		Position:     Position{Line: line - 1, Character: character - 1}, // Convert to 0-indexed
		Context:      ReferenceContext{IncludeDeclaration: includeDeclaration},
	}

	resp, err := server.Call(ctx, "textDocument/references", params)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("LSP error: %s", resp.Error.Message)
	}

	if resp.Result == nil || string(resp.Result) == "null" {
		return []Location{}, nil
	}

	var locations []Location
	if err := json.Unmarshal(resp.Result, &locations); err != nil {
		return nil, fmt.Errorf("failed to parse references result: %w", err)
	}

	return locations, nil
}

// GetDiagnostics gets diagnostics (errors, warnings) for a file
func (c *Client) GetDiagnostics(ctx context.Context, filePath string) ([]Diagnostic, error) {
	server, err := c.GetServerForFile(ctx, filePath)
	if err != nil {
		return nil, err
	}

	// Read file content and open document
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	language := DetectLanguage(filePath)
	if err := server.OpenDocument(filePath, language, string(content)); err != nil {
		return nil, fmt.Errorf("failed to open document: %w", err)
	}

	// Wait for the server to analyze and publish diagnostics
	server.WaitForDocumentReady(ctx, FileToURI(filePath), 500*time.Millisecond)

	// Get diagnostics from server's cache
	return server.GetLastDiagnostics(filePath), nil
}
